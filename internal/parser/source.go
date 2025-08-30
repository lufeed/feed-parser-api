package parser

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/lufeed/feed-parser-api/internal/cache"

	"github.com/google/uuid"
	"github.com/lufeed/feed-parser-api/internal/proxy"

	"sync"

	"github.com/lufeed/feed-parser-api/internal/logger"
	"github.com/lufeed/feed-parser-api/internal/models"
	"github.com/lufeed/feed-parser-api/internal/opengraph"
	"github.com/mmcdole/gofeed"
)

type SourceParser struct {
	ctx          context.Context
	proxyManager *proxy.Manager
}

var (
	maxRetries = 3
)

func NewSourceParser(ctx context.Context, pm *proxy.Manager) *SourceParser {
	return &SourceParser{
		ctx:          ctx,
		proxyManager: pm,
	}
}

func (s *SourceParser) Exec(sourceURL string, sendHTML bool) ([]models.Feed, error) {
	var feed *gofeed.Feed
	var err error

	for attempt := 0; attempt < maxRetries; attempt++ {
		fp := gofeed.NewParser()
		fp.UserAgent = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/134.0.0.0 Safari/537.36 Edg/134.0.0.0"
		cl, proxyID := s.proxyManager.GetProxiedClient()
		fp.Client = cl
		feed, err = fp.ParseURL(sourceURL)
		if err == nil {
			break
		}
		if !strings.Contains(err.Error(), "429") {
			return nil, err
		}
		s.proxyManager.ReleaseProxy(proxyID)

		backoffTime := time.Duration(math.Pow(2, float64(attempt+1))) * time.Second
		jitter := time.Duration(rand.Int63n(int64(backoffTime) / 2))
		retryAfter := backoffTime + jitter

		logger.GetSugaredLogger().Warnf("Got 429 error for %s (attempt %d/%d), retrying after %v",
			sourceURL, attempt+1, maxRetries, retryAfter)
		time.Sleep(retryAfter)
	}

	if feed == nil {
		return nil, fmt.Errorf("failed to parse feed URL: %s", sourceURL)
	}

	var results []models.Feed
	var mu sync.Mutex
	var wg sync.WaitGroup

	proxyCount := s.proxyManager.ProxyCount()
	sem := make(chan struct{}, proxyCount)

	maxItems := 20
	itemCount := len(feed.Items)
	if itemCount > maxItems {
		itemCount = maxItems
	}
	for _, item := range feed.Items[:itemCount] {
		sem <- struct{}{} // acquire slot
		wg.Add(1)
		go func(i *gofeed.Item) {
			defer func() {
				<-sem // release slot
				wg.Done()
			}()
			var f models.Feed
			var err error
			cacheData, cacheErr := cache.GetCache(i.Link)
			if cacheErr == nil && cacheData != "" {
				err = json.Unmarshal([]byte(cacheData), &f)
			}
			if err != nil || cacheErr != nil || cacheData == "" {
				maxItemRetries := 2
				for attempt := 0; attempt <= maxItemRetries; attempt++ {
					cl, proxyID := s.proxyManager.GetProxiedClient()
					f, err = s.parseFeedItem(cl, i, feed.Link, sendHTML)
					s.proxyManager.ReleaseProxy(proxyID)
					if err == nil {
						b, _ := json.Marshal(f)
						cache.SetCache(i.Link, b, time.Hour*24)
						break
					}
					if !isTransientError(err) {
						break
					}
					backoffTime := time.Duration(math.Pow(2, float64(attempt+1))) * time.Second
					jitter := time.Duration(rand.Int63n(int64(backoffTime) / 2))
					retryAfter := backoffTime + jitter
					logger.GetSugaredLogger().Warnf("parseFeedItem transient error for %s (attempt %d/%d), retrying after %v: %v", i.Link, attempt+1, maxItemRetries+1, retryAfter, err)
					time.Sleep(retryAfter)
				}
				if err == nil {
					b, _ := json.Marshal(f)
					cache.SetCache(i.Link, b, time.Hour*24)
				}
			}
			mu.Lock()
			results = append(results, f)
			mu.Unlock()
		}(item)
	}
	wg.Wait()

	return results, nil
}

func (s *SourceParser) parseFeedItem(cl *http.Client, item *gofeed.Item, host string, sendHTML bool) (models.Feed, error) {
	itemLink := strings.Split(item.Link, "?")[0]
	opengraphExtractor := opengraph.NewExtractor(cl, itemLink, host, false)
	wsi, err := opengraphExtractor.Exec()
	if err != nil {
		return models.Feed{}, err
	}
	if wsi.Image == "" && item.Image != nil {
		wsi.Image = item.Image.URL
	}
	if wsi.Description == "" {
		wsi.Description = item.Description
	}
	published := item.PublishedParsed
	if published == nil {
		published = item.UpdatedParsed
	}
	if published == nil {
		now := time.Now()
		published = &now
	}

	imageURL := ""
	if wsi.Image != "" {
		parsedURL, err := url.Parse(wsi.Image)
		if err == nil {
			if !parsedURL.IsAbs() {
				// Get the base URL from the feed item's URL
				baseURL, err := url.Parse(itemLink)
				if err == nil {
					// Remove any path segments after the last slash and then resolve
					basePathParts := strings.Split(baseURL.Path, "/")
					if len(basePathParts) > 1 {
						baseURL.Path = strings.Join(basePathParts[:len(basePathParts)-1], "/")
					}
					imageURL = baseURL.ResolveReference(parsedURL).String()
				}
			} else {
				imageURL = wsi.Image
			}
		}
	}

	if imageURL == "" {
		imageURL = "https://s3.eu-central-1.amazonaws.com/lufeed/feeds/lufeed-bg.png"
	}

	feedID, err := uuid.NewUUID()
	if err != nil {
		return models.Feed{}, err
	}

	feed := models.Feed{
		ID:          feedID,
		Title:       item.Title,
		Description: wsi.Description,
		URL:         itemLink,
		ImageURL:    imageURL,
		PublishedAt: *published,
	}

	if sendHTML {
		feed.HTML = &wsi.HTML
	}

	return feed, nil
}

// Helper to detect transient errors (EOF, timeouts, etc.)
func isTransientError(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	if strings.Contains(errStr, "EOF") ||
		strings.Contains(errStr, "timeout") ||
		strings.Contains(errStr, "Client.Timeout") ||
		strings.Contains(errStr, "context deadline exceeded") {
		return true
	}
	return false
}
