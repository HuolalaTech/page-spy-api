package middleware

import (
	"net/http"
	"strings"

	"github.com/HuolalaTech/page-spy-api/config"
	"github.com/HuolalaTech/page-spy-api/serve/common"
	"github.com/labstack/echo/v4"
)

// Auth 中间件用于验证请求的认证信息
func Auth(cfg *config.Config) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// 判断是否处于无密码模式，若是则跳过认证
			if !IsPasswordSet(cfg) {
				return next(c)
			}

			// 初始化JWT密钥 - 只在实际需要时执行
			if len(jwtSecret) == 0 {
				InitJWTSecret(cfg)
			}

			// 获取Authorization头
			authHeader := c.Request().Header.Get("Authorization")
			if authHeader == "" {
				return c.JSON(http.StatusUnauthorized, common.NewErrorResponseWithCode("Authentication token not provided", "MISSING_TOKEN"))
			}

			// Bearer Token格式验证
			parts := strings.Split(authHeader, " ")
			if len(parts) != 2 || parts[0] != "Bearer" {
				return c.JSON(http.StatusUnauthorized, common.NewErrorResponseWithCode("Invalid authentication token format", "INVALID_TOKEN_FORMAT"))
			}

			token := parts[1]

			// 解析和验证JWT令牌
			claims, err := ParseToken(token)
			if err != nil {
				return c.JSON(http.StatusUnauthorized, common.NewErrorResponseWithCode("Authentication token expired or invalid", "EXPIRED_OR_INVALID_TOKEN"))
			}

			// 将JWT声明存储在上下文中，以便后续使用
			c.Set("jwtClaims", claims)

			return next(c)
		}
	}
}

// IsPasswordSet 检查是否已设置密码
func IsPasswordSet(cfg *config.Config) bool {
	return cfg.AuthConfig != nil && cfg.AuthConfig.Password != ""
}

// VerifyPassword 验证密码是否正确
func VerifyPassword(cfg *config.Config, password string) bool {
	return cfg.AuthConfig != nil && cfg.AuthConfig.Password == password
}
