# README

page-spy-api is the backend service for page-spy-web, which includes static resource serving, HTTP service, and WebSocket service.

# How to use

```golang
package main

import (
	"embed"
	"log"

	"github.com/HuolalaTech/page-spy-api/config"
	"github.com/HuolalaTech/page-spy-api/container"
	"github.com/HuolalaTech/page-spy-api/serve"
)

//go:embed dist/*
var publicContent embed.FS

func main() {
	container := container.Container()
	err := container.Provide(func() *config.StaticConfig {
		// page-spy-web 构建 dist 结构静态资源代理，如果只使用后端可以 return nil
		return &config.StaticConfig{
			DirName: "dist",
			Files:   publicContent,
		}
	})

	if err != nil {
		log.Fatal(err)
	}

	serve.Run()
}

```

## Directory Structure
- config: Project configuration.
- container: Dependency injection.
- event: Event structure definitions.
- logger: Logging interface.
- metric: Metrics interface.
- room: Room interface.
- rpc: Multi-machine RPC interface.
- serve: HTTP and WebSocket services.