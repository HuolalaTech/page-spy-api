package config

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"strconv"

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
	// 初始化 AuthConfig 如果不存在
	if config.AuthConfig == nil {
		config.AuthConfig = &AuthConfig{
			TokenExpiration: 24, // 默认24小时
		}
	}

	// 从环境变量读取密码
	if envPassword := os.Getenv("AUTH_PASSWORD"); envPassword != "" {
		config.AuthConfig.Password = envPassword
	}

	// 从环境变量读取JWT密钥
	if jwtSecret := os.Getenv("JWT_SECRET"); jwtSecret != "" {
		config.AuthConfig.JwtSecret = jwtSecret
	}

	// 从环境变量读取Token过期时间
	if expHours := os.Getenv("JWT_EXPIRATION_HOURS"); expHours != "" {
		if hours, err := strconv.Atoi(expHours); err == nil && hours > 0 {
			config.AuthConfig.TokenExpiration = hours
		}
	}
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
