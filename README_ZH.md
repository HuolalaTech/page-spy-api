[main-repo]: https://github.com/HuolalaTech/page-spy-web
[license-img]: https://img.shields.io/github/license/HuolalaTech/page-spy-api?label=License
[license-url]: https://github.com/HuolalaTech/page-spy-api/blob/main/LICENSE
[api-ver-img]: https://img.shields.io/github/v/tag/HuolalaTech/page-spy-api?label=version
[api-ver-url]: https://github.com/HuolalaTech/page-spy-api/tags
[api-go-img]: https://img.shields.io/github/go-mod/go-version/HuolalaTech/page-spy-api?label=go
[api-go-url]: https://github.com/HuolalaTech/page-spy-api/blob/master/go.mod

<div align="center">
<img src="./.github/assets/logo.svg" height="100" />

<h1>Page Spy API</h1>
<p>PageSpy 是一款远程调试网页的工具。</p>

[![license][license-img]][license-url]
[![API Version][api-ver-img]][api-ver-url]
[![Go Version][api-go-img]][api-go-url]

<a href="https://www.producthunt.com/posts/pagespy?utm_source=badge-featured&utm_medium=badge&utm_souce=badge-pagespy" target="_blank"><img src="https://api.producthunt.com/widgets/embed-image/v1/featured.svg?post_id=429852&theme=light" alt="PageSpy - Remote&#0032;debugging&#0032;as&#0032;seamless&#0032;as&#0032;local&#0032;debugging&#0046; | Product Hunt" height="36" /></a> <a href="https://news.ycombinator.com/item?id=38679798" target="_blank"><img src="https://hackernews-badge.vercel.app/api?id=38679798" alt="PageSpy - Remote&#0032;debugging&#0032;as&#0032;seamless&#0032;as&#0032;local&#0032;debugging&#0046; | Hacker News" height="36" /></a>

[English](./README.md) | 中文

</div>

## 简介

该仓库是 [HuolalaTech/page-spy-web][main-repo] 的后端服务，其中包括静态资源服务，http 服务以及 websocket 服务。

## 如何使用

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

- config 项目配置
- container 依赖注入
- event 事件结构定义
- logger 日志接口
- metric 打点接口
- room 房间接口
- rpc 多机器 rpc 接口
- serve http websocket 服务
