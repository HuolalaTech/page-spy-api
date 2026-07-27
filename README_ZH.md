[web-repo]: https://github.com/HuolalaTech/page-spy-web
[license-badge]: https://img.shields.io/github/license/HuolalaTech/page-spy-api?label=License
[license-link]: https://github.com/HuolalaTech/page-spy-api/blob/master/LICENSE
[version-badge]: https://img.shields.io/github/v/tag/HuolalaTech/page-spy-api?label=version
[version-link]: https://github.com/HuolalaTech/page-spy-api/tags
[go-badge]: https://img.shields.io/github/go-mod/go-version/HuolalaTech/page-spy-api?label=go
[go-link]: https://github.com/HuolalaTech/page-spy-api/blob/master/go.mod

<div align="center">
  <img src="./.github/assets/logo.svg" height="100" alt="PageSpy 标志" />

  <h1>Page Spy API</h1>

  <p>PageSpy 远程网页调试平台的后端服务。</p>

  [![License][license-badge]][license-link]
  [![Version][version-badge]][version-link]
  [![Go][go-badge]][go-link]

  [English](./README.md) | 中文
</div>

## 项目简介

Page Spy API 为 [Page Spy Web][web-repo] 提供后端能力，包括基于房间的 WebSocket 通信、调试日志存储与查询、多实例路由，以及可选的 Page Spy Web 静态资源托管。

支持以下使用方式：

- 将前端嵌入同一个二进制的完整服务。
- 嵌入自定义 Go 入口的纯后端服务。
- SQLite 和本地文件系统组成的单节点部署。
- MySQL 和 S3 兼容存储组成的多节点部署。

## 功能特性

- 通过 WebSocket 创建、查询房间并进行广播或单播。
- 调试日志上传、tag、分组、分页、下载和删除。
- 可选的密码认证及 JWT 管理接口保护。
- 默认使用 SQLite，可选 MySQL。
- 默认使用本地文件系统，可选 S3 兼容对象存储。
- 多实例内部 JSON-RPC 和 HTTP 请求代理。
- 支持 Page Spy Web 的 SPA fallback 静态资源托管。

## 项目文档

| 文档 | English | 中文 |
| --- | --- | --- |
| 使用指南 | [Usage](./docs/USAGE.md) | [使用指南](./docs/USAGE_ZH.md) |
| 开发与测试 | [Development](./docs/DEVELOPMENT.md) | [开发与测试](./docs/DEVELOPMENT_ZH.md) |
| 项目结构与设计 | [Architecture](./docs/ARCHITECTURE.md) | [项目结构与设计](./docs/ARCHITECTURE_ZH.md) |

## 快速开始

### 环境要求

- Go 1.23，模块声明的工具链为 `go1.23.5`。
- 构建集成管理页面的服务时，需要在 `bin/dist` 准备 page-spy-web 构建产物。

安装依赖：

```bash
go mod download
```

### 准备前端

`bin/main.go` 会嵌入 `bin/dist/*`。请将 [page-spy-web][web-repo] 的构建产物复制到：

```text
bin/dist/
├── index.html
└── assets/
```

如果只验证后端构建，可以临时创建一个 `bin/dist/index.html` 来满足 embed 匹配。

### 构建并启动

```bash
go build -o page-spy-api ./bin
./page-spy-api
```

服务从当前工作目录读取 `config.json`。文件不存在时会自动创建：

```json
{
  "port": "6752"
}
```

启动后访问 `http://localhost:6752`。

## 基础配置

```json
{
  "port": "6752",
  "maxRoomNumber": 500,
  "maxLogFileSizeOfMB": 10240,
  "maxLogLifeTimeOfHour": 720,
  "authConfig": {
    "password": "replace-with-a-strong-password",
    "jwtSecret": "replace-with-a-long-random-secret",
    "tokenExpiration": 24
  }
}
```

也可以使用以下环境变量配置认证：

```text
AUTH_PASSWORD
JWT_SECRET
JWT_EXPIRATION_HOURS
```

MySQL、S3、CORS 和多实例配置请查看[使用指南](./docs/USAGE_ZH.md)。

## API 概览

HTTP API 统一位于 `/api/v1`。

| 能力 | 主要接口 |
| --- | --- |
| 认证 | `POST /auth/verify`、`GET /auth/status` |
| 房间 | `POST /room/create`、`GET /room/list`、`GET /room/check` |
| WebSocket | `GET /ws/room/join` |
| 日志 | `/log/upload`、`/jsonLog/upload`、`/log/list`、`/log/download`、`/log/delete` |
| 日志组 | `/logGroup/upload`、`/logGroup/list`、`/logGroup/files`、`/logGroup/delete` |

创建房间：

```bash
curl -sS \
  -X POST \
  -H 'Content-Type: application/json' \
  -d '{"useSecret":false}' \
  'http://localhost:6752/api/v1/room/create?name=demo&group=default'
```

使用返回的 `data.address` 建立连接：

```text
ws://localhost:6752/api/v1/ws/room/join?address=<room-address>&name=debugger&userId=user-1
```

完整接口和消息协议示例请查看[使用指南](./docs/USAGE_ZH.md)。

## 纯后端嵌入

```go
package main

import (
	"log"

	"github.com/HuolalaTech/page-spy-api/config"
	"github.com/HuolalaTech/page-spy-api/container"
	"github.com/HuolalaTech/page-spy-api/serve"
)

func main() {
	app := container.Container()
	if err := app.Provide(func() *config.StaticConfig {
		return nil
	}); err != nil {
		log.Fatal(err)
	}

	serve.Run()
}
```

如果需要在同一进程托管 Page Spy Web，通过 `config.StaticConfig` 提供包含 `dist` 的 `fs.FS`。

## 开发

准备好 `bin/dist` 后执行：

```bash
go test ./...
go test -race ./...
go vet ./...
```

没有前端资源时，使用[开发与测试指南](./docs/DEVELOPMENT_ZH.md)中的后端包列表。

仓库还提供：

- `test/docker`：MySQL 与 Adminer 集成环境。
- `test/websocket_event_test`：浏览器 WebSocket 冒烟测试页。

## 安全部署提示

默认配置没有设置密码。将服务暴露到不可信网络前：

- 设置 `AUTH_PASSWORD` 和稳定的 `JWT_SECRET`。
- 通过可信网络或代理限制 HTTP 与内部 RPC。
- 为 HTTP 和 WebSocket 配置 TLS。
- 在反向代理限制请求体大小和速率。
- 保护可能包含明文凭据的 `config.json`。
- 多实例部署使用共享 MySQL。

详情请查看[生产部署注意事项](./docs/USAGE_ZH.md#10-生产部署注意事项)。

## 相关项目

- [HuolalaTech/page-spy-web][web-repo] — PageSpy 前端与网页调试客户端。

## License

[MIT](./LICENSE)
