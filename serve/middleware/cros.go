package middleware

import (
	"github.com/HuolalaTech/page-spy-api/config"
	echo "github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

func isEmptyArray(a []string) bool {
	return a == nil || len(a) <= 0
}

func CORS(c *config.Config) echo.MiddlewareFunc {
	config := middleware.CORSConfig{
		AllowOrigins:     []string{},
		AllowMethods:     []string{"HEAD", "POST", "GET", "OPTIONS", "PUT", "DELETE", "UPDATE"},
		AllowHeaders:     []string{"Origin", "Authorization", "Content-Length", "X-Request-Id", "Content-Type", "Referer", "User-Agent", "Host"},
		ExposeHeaders:    []string{"X-Request-Id"},
		AllowCredentials: true,
		MaxAge:           12 * 60 * 60,
		AllowOriginFunc: func(origin string) (bool, error) {
			if c.CrosConfig == nil {
				return true, nil
			}

			return false, nil
		},
	}

	if c.CrosConfig != nil {
		if !isEmptyArray(c.CrosConfig.AllowOrigins) {
			config.AllowOrigins = c.CrosConfig.AllowOrigins
		}

		if !isEmptyArray(c.CrosConfig.AllowMethods) {
			config.AllowOrigins = c.CrosConfig.AllowMethods
		}

		if !isEmptyArray(c.CrosConfig.AllowHeaders) {
			config.AllowOrigins = c.CrosConfig.AllowHeaders
		}

		if !isEmptyArray(c.CrosConfig.ExposeHeaders) {
			config.AllowOrigins = c.CrosConfig.ExposeHeaders
		}
	}

	return middleware.CORSWithConfig(config)
}
