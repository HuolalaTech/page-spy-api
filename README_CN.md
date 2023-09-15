# README

page-spy-api 是 page-spy-web 的后端服务，其中包括静态资源服务，http服务以及 websocket 服务。

# 如何使用

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

## 目录结构

- config 		项目配置
- container 	依赖注入
- event 		事件结构定义
- logger 		日志接口
- metric 		打点接口
- room 		房间接口
- rpc			多机器 rpc 接口
- serve		http websocket 服务
