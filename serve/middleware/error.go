package middleware

import (
	"net/http"

	"github.com/HuolalaTech/page-spy-api/serve/common"
	"github.com/labstack/echo/v4"
)

func Error() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			err := next(c)
			if err != nil {
				return c.JSON(http.StatusInternalServerError, common.NewErrorResponse(err))
			}
			return nil
		}
	}
}
