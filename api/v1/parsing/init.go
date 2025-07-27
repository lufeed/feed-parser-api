package parsing

import (
	"github.com/labstack/echo/v4"
)

func Initialize(group *echo.Group) {
	s := newService()
	c := newController(s)

	c.Register(group)
}
