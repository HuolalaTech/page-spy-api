package middleware

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"os"
	"time"

	"github.com/HuolalaTech/page-spy-api/config"
	"github.com/golang-jwt/jwt/v5"
)

// 用于签名JWT的密钥
var jwtSecret []byte

// Claims JWT声明结构
type Claims struct {
	jwt.RegisteredClaims
}

// InitJWTSecret 初始化JWT密钥
func InitJWTSecret(cfg *config.Config) {
	// 优先使用环境变量中的密钥
	envSecret := os.Getenv("JWT_SECRET")
	if envSecret != "" {
		jwtSecret = []byte(envSecret)
		return
	}

	// 其次使用配置文件中的密钥
	if cfg.AuthConfig != nil && cfg.AuthConfig.JwtSecret != "" {
		jwtSecret = []byte(cfg.AuthConfig.JwtSecret)
		return
	}

	// 如果是首次启动（没有AuthConfig配置）
	// 我们不自动创建JwtSecret，而是等到用户设置或跳过密码设置时再创建
	if cfg.AuthConfig == nil {
		// 使用临时密钥，但不保存到配置文件
		jwtSecret = generateRandomKey(32)
		return
	}

	// 如果有AuthConfig但没有JwtSecret（或者JwtSecret为空）
	// 我们只在内存中使用临时密钥，不修改配置文件
	// 当用户设置或跳过密码设置时才会保存到配置文件
	if cfg.AuthConfig.JwtSecret == "" {
		jwtSecret = generateRandomKey(32)
		return
	}
}

// 生成随机密钥
func generateRandomKey(length int) []byte {
	key := make([]byte, length)
	_, err := rand.Read(key)
	if err != nil {
		// 如果随机生成失败，使用时间戳作为备选方案
		now := time.Now().UnixNano()
		for i := 0; i < length; i++ {
			key[i] = byte((now >> (i % 8)) & 0xff)
		}
	}
	return key
}

// GetJWTExpirationHours 获取JWT过期时间
func GetJWTExpirationHours(cfg *config.Config) int {
	// 从配置中获取过期时间
	if cfg.AuthConfig != nil && cfg.AuthConfig.TokenExpiration > 0 {
		return cfg.AuthConfig.TokenExpiration
	}

	// 默认24小时
	return 24
}

// GenerateToken 生成JWT令牌
func GenerateToken(cfg *config.Config) (string, int, error) {
	// 确保JWT密钥已初始化
	if len(jwtSecret) == 0 {
		InitJWTSecret(cfg)

		// 如果jwtSecret还是空的，可能是因为使用了环境变量设置密码但没有设置JWT_SECRET
		// 此时我们需要生成一个临时密钥并保存到配置文件
		if len(jwtSecret) == 0 && cfg.AuthConfig != nil && cfg.AuthConfig.Password != "" {
			newSecret := generateRandomKey(32)
			jwtSecret = newSecret

			// 如果不是通过环境变量JWT_SECRET设置的，就将密钥保存到配置文件
			if os.Getenv("JWT_SECRET") == "" {
				// 将密钥保存到配置中
				cfg.AuthConfig.JwtSecret = base64.StdEncoding.EncodeToString(newSecret)
				// 保存配置到文件
				SaveConfigToFile(cfg)
			}
		}
	}

	// 确定过期时间
	expirationHours := GetJWTExpirationHours(cfg)
	expirationTime := time.Now().Add(time.Hour * time.Duration(expirationHours))

	// 创建声明
	claims := &Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    "page-spy",
		},
	}

	// 创建令牌
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// 签名令牌
	tokenString, err := token.SignedString(jwtSecret)
	if err != nil {
		return "", 0, err
	}

	return tokenString, expirationHours, nil
}

// ParseToken 解析和验证JWT令牌
func ParseToken(tokenString string) (*Claims, error) {
	// 确保JWT密钥已初始化
	if len(jwtSecret) == 0 {
		return nil, fmt.Errorf("JWT secret not initialized")
	}

	// 解析JWT令牌
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		// 验证签名方法
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return jwtSecret, nil
	})

	if err != nil {
		return nil, err
	}

	// 提取声明
	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, fmt.Errorf("invalid token")
}
