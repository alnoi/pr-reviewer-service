package v1

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

// NewRouter собирает Echo и регистрирует хендлеры, сгенерированные oapi-codegen.
func NewRouter(handler ServerInterface) *echo.Echo {
	e := echo.New()

	RegisterHandlers(e, handler)

	e.GET("/health", func(c echo.Context) error {
		return c.NoContent(http.StatusOK)
	})

	return e
}
