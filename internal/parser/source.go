package parser

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"math"
	"math/rand"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/lufeed/feed-parser-api/internal/logger"
	"github.com/lufeed/feed-parser-api/internal/models"
	"github.com/lufeed/feed-parser-api/internal/opengraph"
	"github.com/mmcdole/gofeed"
)

type SourceParser struct {
	ctx    context.Context
	fp     *gofeed.Parser
	client *http.Client
}

var (
	maxRetries = 3
)

func NewSourceParser(ctx context.Context, fp *gofeed.Parser, client *http.Client) *SourceParser {
	return &SourceParser{
		ctx:    ctx,
		fp:     fp,
		client: client,
	}
}

func (s *SourceParser) Exec(sourceURL string) ([]models.Feed, error) {
	var feed *gofeed.Feed
	var err error

	for attempt := 0; attempt < maxRetries; attempt++ {
		feed, err = s.fp.ParseURL(sourceURL)
		if err == nil {
			continue
		}
		if !strings.Contains(err.Error(), "429") {
			return nil, err
		}

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
	for _, item := range feed.Items {
		f, err := s.parseFeedItem(item, feed.Link)
		if err != nil {
			continue
		}
		results = append(results, f)
		time.Sleep(time.Duration(3) * time.Millisecond)
	}

	return results, nil
}

func (s *SourceParser) parseFeedItem(item *gofeed.Item, host string) (models.Feed, error) {
	itemLink := strings.Split(item.Link, "?")[0]
	opengraphExtractor := opengraph.NewExtractor(s.client, itemLink, host, false)
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

	return models.Feed{
		ID:          feedID,
		Title:       item.Title,
		Description: wsi.Description,
		URL:         itemLink,
		ImageURL:    imageURL,
		PublishedAt: *published,
	}, nil
}
