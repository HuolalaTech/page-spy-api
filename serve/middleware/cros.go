package middleware

import (
	"net/url"
	"strings"

	"github.com/HuolalaTech/page-spy-api/config"
	echo "github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

func CORS(c *config.Config) echo.MiddlewareFunc {
	return middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins:     []string{},
		AllowMethods:     []string{"HEAD", "POST", "GET", "OPTIONS", "PUT", "DELETE", "UPDATE"},
		AllowHeaders:     []string{"Origin", "Authorization", "Content-Length", "X-Request-Id", "Content-Type", "Referer", "User-Agent", "Host"},
		ExposeHeaders:    []string{"X-Request-Id"},
		AllowCredentials: true,
		MaxAge:           12 * 60 * 60,
		AllowOriginFunc: func(origin string) (bool, error) {
			u, err := url.Parse(origin)
			if err != nil {
				return false, err
			}

			if c.Origin != "" {
				allow := strings.HasSuffix(u.Hostname(), "")
				return allow, nil
			} else {
				return true, nil
			}
		},
	})
}
