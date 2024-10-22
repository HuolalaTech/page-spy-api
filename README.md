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

<p>PageSpy is a developer platform for debugging web page.</p>

[![license][license-img]][license-url]
[![API Version][api-ver-img]][api-ver-url]
[![Go Version][api-go-img]][api-go-url]


<a href="https://www.producthunt.com/posts/pagespy?utm_source=badge-featured&utm_medium=badge&utm_souce=badge-pagespy" target="_blank"><img src="https://api.producthunt.com/widgets/embed-image/v1/featured.svg?post_id=429852&theme=light" alt="PageSpy - Remote&#0032;debugging&#0032;as&#0032;seamless&#0032;as&#0032;local&#0032;debugging&#0046; | Product Hunt" height="36" /></a> <a href="https://news.ycombinator.com/item?id=38679798" target="_blank"><img src="https://hackernews-badge.vercel.app/api?id=38679798" alt="PageSpy - Remote&#0032;debugging&#0032;as&#0032;seamless&#0032;as&#0032;local&#0032;debugging&#0046; | Hacker News" height="36" /></a>

English | [中文](./README_ZH.md)

</div>

## What's this

The repo is the backend service for [HuolalaTech/page-spy-web][main-repo], which includes static resource serving, HTTP service, and WebSocket service.

## How to use

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
		// page-spy-web build dist static proxy, if no need you can return nil
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
