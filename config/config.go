package config

import (
	"io/fs"
)

type CrosConfig struct {
	AllowOrigins  []string `json:"allowOrigins"`
	AllowMethods  []string `json:"allowMethods"`
	AllowHeaders  []string `json:"allowHeaders"`
	ExposeHeaders []string `json:"exposeHeaders"`
}

type Config struct {
	Port       string      `json:"port"`
	RpcAddress []*Address  `json:"rpcAddress"`
	CrosConfig *CrosConfig `json:"crosConfig"`
}

type Address struct {
	Ip   string `json:"ip"`
	Port string `json:"port"`
}

type StaticConfig struct {
	DirName string
	Files   fs.FS
}
