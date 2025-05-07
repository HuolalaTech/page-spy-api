package config

import (
	"crypto/rand"
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"time"

	"encoding/base64"

	"github.com/labstack/gommon/log"
)

const ConfigFileName = "config.json"

//go:embed defaultConfig.json
var DefaultConfigJsonByte []byte

func LoadConfig() (*Config, error) {
	err := checkLocalConfigFile()
	if err != nil {
		return nil, err
	}

	config, err := loadLocalConfigFile()
	if err != nil {
		return nil, err
	}

	// 从环境变量加载认证配置
	loadAuthConfigFromEnv(config)

	return config, nil
}

// 从环境变量加载认证配置
func loadAuthConfigFromEnv(config *Config) {
	// 如果存在环境变量认证配置，才初始化 AuthConfig
	// 检查是否有任何相关的环境变量设置
	envPassword := os.Getenv("AUTH_PASSWORD")
	jwtSecret := os.Getenv("JWT_SECRET")
	expHours := os.Getenv("JWT_EXPIRATION_HOURS")

	// 只有在环境变量中指定了认证相关配置时，才创建 authConfig
	if envPassword != "" || jwtSecret != "" || expHours != "" {
		// 初始化 AuthConfig 如果不存在
		if config.AuthConfig == nil {
			config.AuthConfig = &AuthConfig{
				TokenExpiration: 24, // 默认24小时
			}
		}

		// 设置环境变量中的认证参数
		if envPassword != "" {
			config.AuthConfig.Password = envPassword
		}

		if jwtSecret != "" {
			config.AuthConfig.JwtSecret = jwtSecret
		} else if envPassword != "" && config.AuthConfig.JwtSecret == "" {
			// 如果设置了密码但没有JWT密钥，生成一个随机的JWT密钥
			randomSecret := generateRandomSecret(32)
			config.AuthConfig.JwtSecret = randomSecret
			// 保存到配置文件
			saveConfigToFile(config)
		}

		if expHours != "" {
			if hours, err := strconv.Atoi(expHours); err == nil && hours > 0 {
				config.AuthConfig.TokenExpiration = hours
			}
		}
	}
}

// 生成随机密钥（Base64编码）
func generateRandomSecret(length int) string {
	key := make([]byte, length)
	_, err := rand.Read(key)
	if err != nil {
		// 如果随机生成失败，使用时间戳作为备选方案
		now := time.Now().UnixNano()
		for i := 0; i < length; i++ {
			key[i] = byte((now >> (i % 8)) & 0xff)
		}
	}
	return base64.StdEncoding.EncodeToString(key)
}

// 保存配置到文件
func saveConfigToFile(config *Config) error {
	// 使用互斥锁防止并发写入
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(ConfigFileName, data, 0644)
}

func checkLocalConfigFile() error {
	_, err := os.Stat(ConfigFileName)
	if os.IsNotExist(err) {
		log.Warnf("config file %s not exist", ConfigFileName)
		file, err := os.Create(ConfigFileName)
		if err != nil {
			return fmt.Errorf("create config file %s error %w", ConfigFileName, err)
		}
		defer file.Close()
		_, err = file.Write(DefaultConfigJsonByte)
		if err != nil {
			return fmt.Errorf("write config file %s error %w", ConfigFileName, err)
		}
	}
	return nil
}

func loadLocalConfigFile() (*Config, error) {
	config := &Config{}
	f, err := os.Open(ConfigFileName)
	if err != nil {
		return nil, fmt.Errorf("read config.json error %w", err)
	}
	defer f.Close()
	encoder := json.NewDecoder(f)
	err = encoder.Decode(config)
	if err != nil {
		return nil, fmt.Errorf("decode config.json error %w", err)
	}
	return config, nil
}
