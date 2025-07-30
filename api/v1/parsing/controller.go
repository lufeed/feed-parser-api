package parsing

import (
	"github.com/labstack/echo/v4"
	"github.com/lufeed/feed-parser-api/internal/types"
	"net/http"
)

type controllerImpl struct {
	service service
}

func newController(service service) types.Registerer {
	return controllerImpl{
		service: service,
	}
}

func (c controllerImpl) Register(group *echo.Group) {

	group.POST("/url", c.parseUrl)
	group.POST("/source", c.parseSource)
}

func (c controllerImpl) parseUrl(ctx echo.Context) error {
	var body requestBody
	err := ctx.Bind(&body)
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, err.Error())
	}

	data, err := c.service.parseUrl(ctx.Request().Context(), body.URL, body.SendHTML)
	if err != nil {
		return echo.NewHTTPError(data.StatusCode(), err.Error())
	}

	return ctx.JSON(data.StatusCode(), data)
}

func (c controllerImpl) parseSource(ctx echo.Context) error {
	var body requestBody
	err := ctx.Bind(&body)
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, err.Error())
	}

	data, err := c.service.parseSource(ctx.Request().Context(), body.URL, body.SendHTML)
	if err != nil {
		return echo.NewHTTPError(data.StatusCode(), err.Error())
	}

	return ctx.JSON(data.StatusCode(), data)
}
