package v1

import (
	"github.com/labstack/echo/v4"
	"github.com/lufeed/feed-parser-api/api/v1/parsing"
	"github.com/lufeed/feed-parser-api/internal/config"
)

func SetupRoutes(group *echo.Group, cfg *config.AppConfig) {
	parsing.Initialize(group.Group("/parsing"))
}
