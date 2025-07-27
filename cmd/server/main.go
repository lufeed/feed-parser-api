package main

import (
	"github.com/lufeed/feed-parser-api/api"
	"github.com/lufeed/feed-parser-api/internal/config"
	"github.com/lufeed/feed-parser-api/internal/logger"
)

func main() {
	config.Initialize()

	cfg := config.GetConfig()

	err := logger.Initialize(cfg)
	if err != nil {
		return
	}

	err = api.Initialize(cfg)
	if err != nil {
		return
	}
}
