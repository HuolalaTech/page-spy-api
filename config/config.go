package config

import "io/fs"

type Config struct {
	MachineInfo *MachineInfo `json:"machineInfo"`
	Port        string       `json:"port"`
	Origin      string       `json:"origin"`
}

type Address struct {
	Ip   string `json:"ip"`
	Port string `json:"port"`
}

type MachineInfo struct {
	MachineAddress map[string]*Address `json:"machineAddress"`
}
type StaticConfig struct {
	DirName string
	Files   fs.FS
}
