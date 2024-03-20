package config

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"os"

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

	return loadLocalConfigFile()
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
