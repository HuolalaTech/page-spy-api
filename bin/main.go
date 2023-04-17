package main

import (
	"embed"

	"github.com/HuolalaTech/page-spy-api/config"
	"github.com/HuolalaTech/page-spy-api/serve"
)

//go:embed dist/*
var publicContent embed.FS

func main() {
	serve.Run(&config.StaticConfig{
		DirName: "dist",
		Files:   publicContent,
	})
}
