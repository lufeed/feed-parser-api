package parsing

import (
	"github.com/labstack/echo/v4"
	"github.com/lufeed/feed-parser-api/internal/config"
	"github.com/lufeed/feed-parser-api/internal/proxy"
)

func Initialize(group *echo.Group) {
	pm := proxy.NewManager(config.GetConfig())
	s := newService(pm)
	c := newController(s)

	c.Register(group)
}
