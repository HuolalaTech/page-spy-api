[main-repo]: https://github.com/HuolalaTech/page-spy-web
[api-ver-img]: https://img.shields.io/github/v/tag/HuolalaTech/page-spy-api?label=version
[api-ver-url]: https://github.com/HuolalaTech/page-spy-api/tags
[api-go-img]: https://img.shields.io/github/go-mod/go-version/HuolalaTech/page-spy-api?label=go
[api-go-url]: https://github.com/HuolalaTech/page-spy-api/blob/master/go.mod

<div align="center">
<img src="./.github/assets/logo.svg" height="100" />

<h1>Page Spy API</h1>

<p>PageSpy is a developer platform for debugging web page.</p>

[![API Version][api-ver-img]][api-ver-url]
[![Go Version][api-go-img]][api-go-url]

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
