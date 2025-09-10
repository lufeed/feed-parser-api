package parser

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/lufeed/feed-parser-api/internal/browser"
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

// FeedItemHandler is a callback for each parsed feed item
// If nil, no callback is invoked (API mode)
type FeedItemHandler func(item models.Feed)

func (s *SourceParser) Exec(sourceURL string, sendHTML bool, onItem FeedItemHandler) ([]models.Feed, error) {
	var feed *gofeed.Feed
	var err error
	logger.GetSugaredLogger().Infof("Parsing feed %s", sourceURL)

	for attempt := 0; attempt < maxRetries; attempt++ {
		fp := gofeed.NewParser()
		fp.UserAgent = browser.GetUserAgent()
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

			cacheData, err := cache.GetCache(i.Link)
			if err == nil && cacheData != "" {
				err = json.Unmarshal([]byte(cacheData), &f)
				if err != nil {
					// fallback to parsing if unmarshal fails
					cl, proxyID := s.proxyManager.GetProxiedClient()
					f, err = s.parseFeedItem(cl, i, feed.Link, sendHTML)
					s.proxyManager.ReleaseProxy(proxyID)
					b, _ := json.Marshal(f)
					cache.SetCache(i.Link, b, time.Hour*24)
				}
			} else {
				cl, proxyID := s.proxyManager.GetProxiedClient()
				f, err = s.parseFeedItem(cl, i, feed.Link, sendHTML)
				s.proxyManager.ReleaseProxy(proxyID)
				b, _ := json.Marshal(f)
				cache.SetCache(i.Link, b, time.Hour*24)
			}
			if onItem != nil {
				onItem(f)
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
