package middleware

import (
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/lufeed/feed-parser-api/internal/config"
)

func APIKeyAuth(cfg *config.AppConfig) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			authHeader := c.Request().Header.Get("Authorization")
			if authHeader == "" {
				return echo.NewHTTPError(http.StatusUnauthorized, "Missing authorization header")
			}

			if !strings.HasPrefix(authHeader, "Bearer ") {
				return echo.NewHTTPError(http.StatusUnauthorized, "Invalid authorization header format")
			}

			apiKey := strings.TrimPrefix(authHeader, "Bearer ")
			if apiKey == "" {
				return echo.NewHTTPError(http.StatusUnauthorized, "Missing API key")
			}

			if !isValidAPIKey(apiKey, cfg) {
				return echo.NewHTTPError(http.StatusUnauthorized, "Invalid API key")
			}

			return next(c)
		}
	}
}

func isValidAPIKey(apiKey string, cfg *config.AppConfig) bool {
	for _, validKey := range cfg.Auth.APIKeys {
		if apiKey == validKey {
			return true
		}
	}
	return false
}
