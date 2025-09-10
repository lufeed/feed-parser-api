package main

import (
	"context"
	"encoding/json"

	"github.com/lufeed/feed-parser-api/internal/cache"
	"github.com/lufeed/feed-parser-api/internal/config"
	"github.com/lufeed/feed-parser-api/internal/logger"
	"github.com/lufeed/feed-parser-api/internal/models"
	"github.com/lufeed/feed-parser-api/internal/parser"
	"github.com/lufeed/feed-parser-api/internal/proxy"
	"go.uber.org/zap"
)

func main() {
	config.Initialize()

	cfg := config.GetConfig()

	err := logger.Initialize(cfg)
	if err != nil {
		return
	}

	err = cache.Initialize(cfg)
	if err != nil {
		logger.GetLogger().Error("Cache initialization failed", zap.Error(err))
		return
	}

	proxyManager := proxy.NewManager(cfg)
	ctx := context.Background()

	go listenSourceRequests(ctx, proxyManager)
	go listenURLRequests(ctx, proxyManager)

	select {} // block forever
}

type parseSourceRequest struct {
	URL      string `json:"url"`
	SendHTML bool   `json:"send_html"`
	FeedID   string `json:"feed_id"`
	FeedName string `json:"feed_name"`
	UserID   string `json:"user_id"`
}

type parseURLRequest struct {
	RequestID string `json:"request_id"`
	URL       string `json:"url"`
	SendHTML  bool   `json:"send_html"`
	UserID    string `json:"user_id"`
}

func listenSourceRequests(ctx context.Context, pm *proxy.Manager) {
	pubsub := cache.Subscribe("parse_source_requests")
	for msg := range pubsub.Channel() {
		var req parseSourceRequest
		if err := json.Unmarshal([]byte(msg.Payload), &req); err != nil {
			logger.GetSugaredLogger().Errorf("Invalid parse_source_request: %v", err)
			continue
		}
		sp := parser.NewSourceParser(ctx, pm)
		sp.Exec(req.URL, req.SendHTML, func(item models.Feed) {
			item.FeedID = req.FeedID
			item.FeedName = req.FeedName
			item.UserID = req.UserID
			b, _ := json.Marshal(item)
			cache.Publish("parse_source_results", b)
			logger.GetSugaredLogger().Infof("Published source %s", item.FeedName)
		})
		// Optionally publish a done message
		// cache.Publish("parse_source_results:"+req.RequestID, []byte(`{"done":true}`))
	}
}

func listenURLRequests(ctx context.Context, pm *proxy.Manager) {
	pubsub := cache.Subscribe("parse_url_requests")
	for msg := range pubsub.Channel() {
		var req parseURLRequest
		if err := json.Unmarshal([]byte(msg.Payload), &req); err != nil {
			logger.GetSugaredLogger().Errorf("Invalid parse_url_request: %v", err)
			continue
		}
		up := parser.NewURLParser(ctx, pm)
		up.Exec(req.URL, req.SendHTML, func(source models.Source) {
			source.UserID = req.UserID
			source.RequestID = req.RequestID
			b, _ := json.Marshal(source)
			cache.Publish("parse_url_results", b)
			logger.GetSugaredLogger().Infof("Published url %s", req.URL)
		})
		// Optionally publish a done message
		// cache.Publish("parse_url_results:"+req.RequestID, []byte(`{"done":true}`))
	}
}
