package middleware

import (
	"encoding/base64"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"sync"

	"github.com/HuolalaTech/page-spy-api/config"
	"github.com/HuolalaTech/page-spy-api/serve/common"
	"github.com/labstack/echo/v4"
)

// configMutex 用于保护配置文件的并发写入
var configMutex = &sync.Mutex{}

// SaveConfigToFile 将配置保存到文件
func SaveConfigToFile(cfg *config.Config) error {
	configMutex.Lock()
	defer configMutex.Unlock()

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}

	return ioutil.WriteFile(config.ConfigFileName, data, 0644)
}

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
	// 检查环境变量中是否设置了密码
	if envPassword := os.Getenv("AUTH_PASSWORD"); envPassword != "" {
		return true
	}

	// 检查配置文件中是否设置了密码
	return cfg.AuthConfig != nil && cfg.AuthConfig.Password != ""
}

// IsFirstStart 检查是否是系统首次启动
func IsFirstStart(cfg *config.Config) bool {
	return cfg.AuthConfig == nil
}

// VerifyPassword 验证密码是否正确
func VerifyPassword(cfg *config.Config, password string) bool {
	// 优先检查环境变量中的密码
	if envPassword := os.Getenv("AUTH_PASSWORD"); envPassword != "" {
		return password == envPassword
	}

	// 其次检查配置文件中的密码
	if cfg.AuthConfig != nil {
		return password == cfg.AuthConfig.Password
	}

	return false
}

// SetPassword 设置密码到配置文件
func SetPassword(cfg *config.Config, password string) error {
	// 如果环境变量中已设置密码，则不允许通过接口修改
	if envPassword := os.Getenv("AUTH_PASSWORD"); envPassword != "" {
		return echo.NewHTTPError(http.StatusBadRequest, "System is using password from environment variable, cannot set via API")
	}

	// 初始化AuthConfig如果不存在
	if cfg.AuthConfig == nil {
		cfg.AuthConfig = &config.AuthConfig{
			TokenExpiration: 24, // 默认24小时
		}
	}

	// 设置密码
	cfg.AuthConfig.Password = password

	// 确保JWTSecret已设置
	if cfg.AuthConfig.JwtSecret == "" {
		// 生成随机密钥并设置到JwtSecret
		newSecret := generateRandomKey(32)
		jwtSecret = newSecret // 设置全局变量
		cfg.AuthConfig.JwtSecret = base64.StdEncoding.EncodeToString(newSecret)
	}

	// 保存配置到文件
	return SaveConfigToFile(cfg)
}

// SkipPasswordSetup 跳过密码设置（设置为空密码模式）
func SkipPasswordSetup(cfg *config.Config) error {
	// 如果环境变量中已设置密码，则不允许通过接口修改
	if envPassword := os.Getenv("AUTH_PASSWORD"); envPassword != "" {
		return echo.NewHTTPError(http.StatusBadRequest, "System is using password from environment variable, cannot skip password setup")
	}

	// 初始化AuthConfig如果不存在
	if cfg.AuthConfig == nil {
		cfg.AuthConfig = &config.AuthConfig{
			TokenExpiration: 24, // 默认24小时
		}
	}

	// 设置为空密码
	cfg.AuthConfig.Password = ""

	// 确保JWTSecret已设置，即使是无密码模式
	if cfg.AuthConfig.JwtSecret == "" {
		// 生成随机密钥并设置到JwtSecret
		newSecret := generateRandomKey(32)
		jwtSecret = newSecret // 设置全局变量
		cfg.AuthConfig.JwtSecret = base64.StdEncoding.EncodeToString(newSecret)
	}

	// 保存配置到文件
	return SaveConfigToFile(cfg)
}
