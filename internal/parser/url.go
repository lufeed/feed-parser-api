package parser

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"github.com/lufeed/feed-parser-api/internal/logger"
	"github.com/lufeed/feed-parser-api/internal/models"
	"github.com/lufeed/feed-parser-api/internal/opengraph"
	"github.com/mmcdole/gofeed"
	"go.uber.org/zap"
	"html"
	"net/http"
	"net/url"
	"strings"
)

type URLParser struct {
	ctx    context.Context
	fp     *gofeed.Parser
	client *http.Client
}

func NewURLParser(ctx context.Context, fp *gofeed.Parser, client *http.Client) *URLParser {
	return &URLParser{
		ctx:    ctx,
		fp:     fp,
		client: client,
	}
}

func (p *URLParser) Exec(sourceUrl string) (models.Source, error) {
	feed, err := p.fp.ParseURL(sourceUrl)
	if err != nil {
		logger.GetSugaredLogger().Warnf("Cannot parse URL: %s error: %s", sourceUrl, err.Error())
		return models.Source{}, err
	}

	id, err := uuid.NewUUID()
	if err != nil {
		return models.Source{}, err
	}

	newSource := models.Source{
		ID:          id,
		Name:        strings.TrimSpace(html.UnescapeString(feed.Title)),
		Description: feed.Description,
		FeedURL:     sourceUrl,
		HomeURL:     strings.Split(feed.Link, "?")[0],
	}

	opengraphExtractor := opengraph.NewExtractor(p.client, newSource.HomeURL, newSource.HomeURL, true)
	wsi, err := opengraphExtractor.Exec()
	if wsi.Description != "" {
		newSource.Description = html.UnescapeString(wsi.Description)
	} else {
		newSource.Description = html.UnescapeString(feed.Title)
	}
	newSource.ImageURL = wsi.Image
	if newSource.ImageURL == "" && feed.Image != nil && feed.Image.URL != "" {
		newSource.ImageURL = feed.Image.URL
	}
	newSource.ImageURL = p.getImageUrl(newSource.HomeURL, newSource.ImageURL, "covers")
	newSource.IconURL = p.getImageUrl(newSource.HomeURL, wsi.Icon, "icons")

	if newSource.Name == "" {
		newSource.Name = strings.TrimSpace(wsi.Title)
		if newSource.Name == "" {
			newSource.Name = "Unknown Title"
		}
	}

	if newSource.ImageURL == "" {
		newSource.ImageURL = "https://s3.eu-central-1.amazonaws.com/lufeed/sources/covers/lufeed-bg.png"
	}
	if newSource.IconURL == "" {
		newSource.IconURL = "https://s3.eu-central-1.amazonaws.com/lufeed/sources/icons/lf-icon.png"
	}

	return newSource, nil
}

func (p *URLParser) getImageUrl(homeUrl, originalURL string, imageType string) string {
	imageURL := ""
	if originalURL != "" {
		parsedURL, err := url.Parse(originalURL)
		if err == nil {
			if !parsedURL.IsAbs() {
				// Get the base URL from the feed item's URL
				baseURL, err := url.Parse(homeUrl)
				if err == nil {
					// Remove any path segments after the last slash and then resolve
					basePathParts := strings.Split(baseURL.Path, "/")
					if len(basePathParts) > 1 {
						baseURL.Path = strings.Join(basePathParts[:len(basePathParts)-1], "/")
					}
					imageURL = baseURL.ResolveReference(parsedURL).String()
				}
			} else {
				imageURL = originalURL
			}
		}

		// Test image URL before downloading
		headResp, err := p.client.Head(imageURL)
		if err != nil {
			logger.GetSugaredLogger().With(zap.String("url", imageURL)).Errorf("Error checking image URL: %s", err.Error())
			imageURL = ""
		} else {
			headResp.Body.Close()
			if headResp.StatusCode != http.StatusOK {
				logger.GetSugaredLogger().With(zap.String("url", imageURL), zap.Int("status", headResp.StatusCode)).Debugf("Image URL returned non-200 status code")
				imageURL = ""
			}
		}

		if imageURL == "" {
			parsedHomeURL, err := url.Parse(homeUrl)
			if err == nil && imageType == "icons" {
				imageURL = fmt.Sprintf("%s://%s/favicon.ico", parsedHomeURL.Scheme, parsedHomeURL.Host)
			}
		}
	}
	return imageURL
}
