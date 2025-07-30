package main

import (
	"github.com/lufeed/feed-parser-api/api"
	"github.com/lufeed/feed-parser-api/internal/cache"
	"github.com/lufeed/feed-parser-api/internal/config"
	"github.com/lufeed/feed-parser-api/internal/logger"
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

	err = api.Initialize(cfg)
	if err != nil {
		return
	}
}
