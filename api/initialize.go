package api

import (
	"fmt"
	"github.com/labstack/echo/v4"
	echoMiddleware "github.com/labstack/echo/v4/middleware"
	v1 "github.com/lufeed/feed-parser-api/api/v1"
	"github.com/lufeed/feed-parser-api/internal/config"
	"github.com/lufeed/feed-parser-api/internal/logger"
	"golang.org/x/time/rate"
	"net/http"
)

func Initialize(cfg *config.AppConfig) error {
	e := echo.New()
	e.Use(echoMiddleware.Recover())
	e.Use(echoMiddleware.RequestID())
	e.Use(echoMiddleware.Logger())

	e.Use(echoMiddleware.CORSWithConfig(echoMiddleware.CORSConfig{
		AllowMethods: []string{"GET", "POST"},
		AllowHeaders: []string{"Content-Type", "Authorization", "X-Requested-With"},
	}))

	e.Use(echoMiddleware.RateLimiter(echoMiddleware.NewRateLimiterMemoryStore(rate.Limit(20))))
	e.Pre(echoMiddleware.RemoveTrailingSlash())

	e.GET("/ping", func(c echo.Context) error {
		return c.JSON(http.StatusOK, echo.Map{
			"message": "pong",
			"service": cfg.Service.Name,
		})
	})

	apiGroup := e.Group(cfg.Server.RootPath)
	v1.SetupRoutes(apiGroup.Group("/v1"), cfg)

	logger.GetSugaredLogger().Infof("Starting server on address %s...", cfg.Server.Host)
	e.Logger.Fatal(e.Start(fmt.Sprintf("%s:%s", cfg.Server.Host, cfg.Server.Port)))
	return nil
}
