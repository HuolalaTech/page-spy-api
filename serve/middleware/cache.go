package middleware

import (
	"mime"
	"path/filepath"
	"strings"

	"github.com/labstack/echo/v4"
)

func isHtml(ct string) bool {
	return strings.Contains(ct, "text/html")
}

func Cache() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			ct := mime.TypeByExtension(filepath.Ext(c.Request().RequestURI))
			contentControl := c.Response().Header().Get("Cache-Control")
			if ct != "" && !isHtml(ct) && contentControl == "" {
				c.Response().Header().Set("Cache-Control", "max-age=2592000")
			}

			err := next(c)
			return err
		}
	}
}
