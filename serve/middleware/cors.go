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
		AllowMethods:     []string{"HEAD", "POST", "GET", "OPTIONS", "PUT", "DELETE", "PATCH"},
		AllowHeaders:     []string{"Origin", "Authorization", "Content-Length", "X-Request-Id", "Content-Type", "Referer", "User-Agent", "Host"},
		ExposeHeaders:    []string{"X-Request-Id"},
		AllowCredentials: true,
		MaxAge:           12 * 60 * 60,
		AllowOriginFunc: func(origin string) (bool, error) {
			if c.CorsConfig == nil {
				return true, nil
			}

			return false, nil
		},
	}

	if c.CorsConfig != nil {
		if !isEmptyArray(c.CorsConfig.AllowOrigins) {
			config.AllowOrigins = c.CorsConfig.AllowOrigins
			config.AllowOriginFunc = nil
		}

		if !isEmptyArray(c.CorsConfig.AllowMethods) {
			config.AllowMethods = c.CorsConfig.AllowMethods
		}

		if !isEmptyArray(c.CorsConfig.AllowHeaders) {
			config.AllowHeaders = c.CorsConfig.AllowHeaders
		}

		if !isEmptyArray(c.CorsConfig.ExposeHeaders) {
			config.ExposeHeaders = c.CorsConfig.ExposeHeaders
		}
	}

	return middleware.CORSWithConfig(config)
}
