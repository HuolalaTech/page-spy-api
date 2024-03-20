package config

import (
	"io/fs"
)

type CorsConfig struct {
	AllowOrigins  []string `json:"allowOrigins"`
	AllowMethods  []string `json:"allowMethods"`
	AllowHeaders  []string `json:"allowHeaders"`
	ExposeHeaders []string `json:"exposeHeaders"`
}

type Config struct {
	Port       string      `json:"port"`
	RpcAddress []*Address  `json:"rpcAddress"`
	CorsConfig *CorsConfig `json:"corsConfig"`
	// max log file size, unit is mb
	MaxLogFileSize int64 `json:"maxLogFileSize"`
	// max log file size, unit is day
	MaxLogLifeTime int64 `json:"maxLogLifeTime"`
}

type Address struct {
	Ip   string `json:"ip"`
	Port string `json:"port"`
}

type StaticConfig struct {
	DirName string
	Files   fs.FS
	GitHash string
	Version string
}
