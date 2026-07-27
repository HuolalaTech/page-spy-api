[web-repo]: https://github.com/HuolalaTech/page-spy-web
[license-badge]: https://img.shields.io/github/license/HuolalaTech/page-spy-api?label=License
[license-link]: https://github.com/HuolalaTech/page-spy-api/blob/master/LICENSE
[version-badge]: https://img.shields.io/github/v/tag/HuolalaTech/page-spy-api?label=version
[version-link]: https://github.com/HuolalaTech/page-spy-api/tags
[go-badge]: https://img.shields.io/github/go-mod/go-version/HuolalaTech/page-spy-api?label=go
[go-link]: https://github.com/HuolalaTech/page-spy-api/blob/master/go.mod

<div align="center">
  <img src="./.github/assets/logo.svg" height="100" alt="PageSpy logo" />

  <h1>Page Spy API</h1>

  <p>The backend service for PageSpy remote web debugging.</p>

  [![License][license-badge]][license-link]
  [![Version][version-badge]][version-link]
  [![Go][go-badge]][go-link]

  English | [中文](./README_ZH.md)
</div>

## Overview

Page Spy API powers [Page Spy Web][web-repo]. It provides room-based WebSocket communication, debug-log storage and search, optional multi-instance routing, and optional hosting of the Page Spy Web frontend.

Use it as:

- A complete server with the frontend embedded in one binary.
- A backend-only service embedded in a custom Go entry point.
- A single-node SQLite/filesystem deployment.
- A multi-node MySQL/S3-compatible deployment.

## Features

- Real-time room creation, discovery, broadcast, and direct messaging over WebSocket.
- Debug-log upload, tags, groups, pagination, download, and deletion.
- Optional password authentication with JWT-protected management APIs.
- SQLite by default, with optional MySQL.
- Local filesystem by default, with optional S3-compatible object storage.
- Internal JSON-RPC and HTTP proxying for multi-instance deployments.
- SPA fallback hosting for Page Spy Web.

## Documentation

| Document | English | 中文 |
| --- | --- | --- |
| User guide | [Usage](./docs/USAGE.md) | [使用指南](./docs/USAGE_ZH.md) |
| Development and testing | [Development](./docs/DEVELOPMENT.md) | [开发与测试](./docs/DEVELOPMENT_ZH.md) |
| Architecture and project structure | [Architecture](./docs/ARCHITECTURE.md) | [项目结构与设计](./docs/ARCHITECTURE_ZH.md) |

## Quick start

### Requirements

- Go 1.23. The module declares the `go1.23.5` toolchain.
- A page-spy-web build in `bin/dist` when building the integrated UI.

Install dependencies:

```bash
go mod download
```

### Prepare the frontend

`bin/main.go` embeds `bin/dist/*`. Copy the [page-spy-web][web-repo] build output into:

```text
bin/dist/
├── index.html
└── assets/
```

For a backend-only build check, a temporary `bin/dist/index.html` is enough to satisfy the embed pattern.

### Build and run

```bash
go build -o page-spy-api ./bin
./page-spy-api
```

The service reads `config.json` from its current working directory. If the file does not exist, it creates:

```json
{
  "port": "6752"
}
```

Open `http://localhost:6752`.

## Basic configuration

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

Authentication can also be configured through:

```text
AUTH_PASSWORD
JWT_SECRET
JWT_EXPIRATION_HOURS
```

See the [user guide](./docs/USAGE.md) for MySQL, S3, CORS, and multi-instance configuration.

## API overview

The HTTP API is under `/api/v1`.

| Capability | Main endpoints |
| --- | --- |
| Authentication | `POST /auth/verify`, `GET /auth/status` |
| Rooms | `POST /room/create`, `GET /room/list`, `GET /room/check` |
| WebSocket | `GET /ws/room/join` |
| Logs | `/log/upload`, `/jsonLog/upload`, `/log/list`, `/log/download`, `/log/delete` |
| Log groups | `/logGroup/upload`, `/logGroup/list`, `/logGroup/files`, `/logGroup/delete` |

Example room creation:

```bash
curl -sS \
  -X POST \
  -H 'Content-Type: application/json' \
  -d '{"useSecret":false}' \
  'http://localhost:6752/api/v1/room/create?name=demo&group=default'
```

Use the returned `data.address` to connect:

```text
ws://localhost:6752/api/v1/ws/room/join?address=<room-address>&name=debugger&userId=user-1
```

Protocol examples and the complete API reference are in the [user guide](./docs/USAGE.md).

## Backend-only embedding

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

Provide an embedded `fs.FS` through `config.StaticConfig` to host Page Spy Web from the same process.

## Development

With `bin/dist` available:

```bash
go test ./...
go test -race ./...
go vet ./...
```

Without frontend assets, use the backend-only package list documented in the [development guide](./docs/DEVELOPMENT.md).

The repository also contains:

- `test/docker`: MySQL and Adminer integration environment.
- `test/websocket_event_test`: browser-based WebSocket smoke-test page.

## Security notes

The default configuration does not set a password. Before exposing the service to an untrusted network:

- Set `AUTH_PASSWORD` and a stable `JWT_SECRET`.
- Put HTTP and internal RPC behind trusted network controls.
- Terminate HTTP and WebSocket traffic with TLS.
- Limit request sizes and rates at a reverse proxy.
- Protect `config.json`, which may contain plaintext credentials.
- Use shared MySQL for multi-instance deployments.

See [Production checklist](./docs/USAGE.md#10-production-checklist) for details.

## Related project

- [HuolalaTech/page-spy-web][web-repo] — PageSpy frontend and web debugging client.

## License

[MIT](./LICENSE)
