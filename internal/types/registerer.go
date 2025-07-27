package types

import (
	"github.com/labstack/echo/v4"
)

type Registerer interface {
	Register(group *echo.Group)
}
