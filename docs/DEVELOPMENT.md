# Page Spy API Development and Testing Guide

English | [中文](./DEVELOPMENT_ZH.md)

This guide covers local setup, builds, code-quality checks, automated tests, manual verification, and common development issues.

## 1. Development environment

### 1.1 Required tools

- Go `1.23`; using the `go1.23.5` toolchain declared in `go.mod` is recommended.
- Git.
- Optional: Docker and Docker Compose for MySQL integration testing.
- Optional: `curl` and `websocat` for HTTP and WebSocket smoke tests.

Check your environment:

```bash
go version
git version
docker version
docker compose version
```

### 1.2 Initialize the repository

```bash
git clone https://github.com/HuolalaTech/page-spy-api.git
cd page-spy-api
go mod download
```

At runtime, the service reads or creates `config.json` in its current working directory. It may also create:

```text
data/data.db
log/
```

These are runtime artifacts and must not be committed as source.

## 2. Build and run locally

### 2.1 Complete entry point

`bin/main.go` embeds `bin/dist/*`. Copy the page-spy-web build output into `bin/dist`, then run:

```bash
go build -o page-spy-api ./bin
./page-spy-api
```

For a backend-only build check, create a temporary minimal asset:

```bash
mkdir -p bin/dist
printf '<!doctype html><title>Page Spy API</title>\n' > bin/dist/index.html
go build -o page-spy-api ./bin
```

`bin/dist` is ignored by Git. Do not commit placeholder or generated frontend files.

### 2.2 Embed as a Go module

A custom program can provide the Page Spy API entry point. The dependency-injection container requires a `*config.StaticConfig` provider. Return `nil` for backend-only mode:

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

To host the frontend, set `StaticConfig.Files` to an `fs.FS` that contains a `dist` directory.

### 2.3 Debug mode

Set the following in `config.json`:

```json
{
  "port": "6752",
  "debug": true
}
```

Debug mode currently enables GORM SQL logging. Application logs are written to standard output.

## 3. Code-quality checks

### 3.1 Formatting

List unformatted tracked Go files:

```bash
gofmt -l $(git ls-files '*.go')
```

Format changed files:

```bash
gofmt -w path/to/file.go path/to/file_test.go
```

Check whitespace errors before committing:

```bash
git diff --check
```

### 3.2 Tests

When `bin/dist` is available:

```bash
go test ./...
go test -race ./...
```

Without frontend assets, skip the `bin` package and verify all backend packages:

```bash
go test \
  ./api/... \
  ./config \
  ./container \
  ./data \
  ./event \
  ./logger \
  ./metric \
  ./proxy \
  ./room \
  ./rpc \
  ./serve/... \
  ./state \
  ./static \
  ./storage \
  ./task \
  ./util
```

Use the same package list for race detection:

```bash
go test -race \
  ./api/... \
  ./config \
  ./container \
  ./data \
  ./event \
  ./logger \
  ./metric \
  ./proxy \
  ./room \
  ./rpc \
  ./serve/... \
  ./state \
  ./static \
  ./storage \
  ./task \
  ./util
```

Coverage depends on the branch under test. If a package has no `_test.go` files, these commands provide compilation checks only. Add tests with every behavior change, especially for room concurrency, WebSocket lifecycle, database queries, and storage failures.

### 3.3 Static analysis

With frontend assets:

```bash
go vet ./...
```

Without frontend assets:

```bash
go vet \
  ./api/... \
  ./config \
  ./container \
  ./data \
  ./event \
  ./logger \
  ./metric \
  ./proxy \
  ./room \
  ./rpc \
  ./serve/... \
  ./state \
  ./static \
  ./storage \
  ./task \
  ./util
```

## 4. Unit-test guidance

### 4.1 Test package choice

- Use the implementation package, such as `package room`, when testing unexported state or concurrency details.
- Use an external package, such as `package room_test`, when testing only the public API and compatibility.
- Place tests next to their implementation and name them `*_test.go`.

### 4.2 High-priority coverage

| Module | Important tests |
| --- | --- |
| `serve/middleware` | Authentication on/off, JWT expiry, and sensitive request logging. |
| `room` | Concurrent Join/Leave/Close, timeout cleanup, wrong passwords, and message routing. |
| `event` | Local/remote delivery, listener changes, and context cancellation. |
| `serve/socket` | Client disconnect, room closure, read/write errors, and malformed messages. |
| `data` | SQLite/MySQL consistency, pagination, tags, time ranges, and soft deletion. |
| `storage` | Local files, S3 keys, missing objects, and partial failures. |
| `serve/route` | HTTP parameters, authentication, error responses, and proxy branches. |
| `task` | Repeated start/close, ticker shutdown, and panic recovery. |

### 4.3 Use boundary interfaces

The main boundaries already expose interfaces:

- `data.DataApi`
- `storage.StorageApi`
- `event.EventEmitter`
- `room.Room` / `room.RemoteRoom`
- `metric.Metric`

Prefer small fake implementations for unit tests instead of starting real S3 or RPC services. Database-query tests can use temporary in-memory SQLite:

```go
db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
```

For tests involving goroutines:

- Wait on channels or `sync.WaitGroup` for deterministic events.
- Avoid relying on long, fixed `time.Sleep` calls.
- Run `go test -race`.
- Set context timeouts on blocking work so tests cannot hang forever.

## 5. HTTP smoke tests

Start the service and check the default password-free mode:

```bash
curl -i http://localhost:6752/api/v1/auth/status
```

Create a room:

```bash
curl -sS \
  -X POST \
  -H 'Content-Type: application/json' \
  -d '{"useSecret":false}' \
  'http://localhost:6752/api/v1/room/create?name=dev-room&group=test'
```

Upload and list a log:

```bash
curl -sS \
  -H 'Content-Type: application/json' \
  --data-binary '{"message":"development smoke test"}' \
  'http://localhost:6752/api/v1/jsonLog/upload?name=smoke.json&env=dev'

curl -sS \
  'http://localhost:6752/api/v1/log/list?page=1&size=10&env=dev'
```

After enabling a password, obtain a token from `/api/v1/auth/verify` and add this header to protected requests:

```text
Authorization: Bearer <token>
```

## 6. WebSocket smoke tests

### 6.1 Command line

Create a room, copy its `data.address`, and connect:

```bash
websocat \
  'ws://localhost:6752/api/v1/ws/room/join?address=<room-address>&name=developer&userId=dev-1'
```

Send:

```json
{"type":"ping","createdAt":1753344000000,"requestId":"smoke-1","content":{}}
```

Expect a `pong` with the same `requestId`.

### 6.2 Browser test page

The repository contains:

```text
test/websocket_event_test/index.html
```

Serve it locally:

```bash
python3 -m http.server 8081 --directory test/websocket_event_test
```

Open `http://localhost:8081`.

Use the real `<local-id>.<machine-id>` address returned by the room creation endpoint. The initial placeholder on the page may not be a valid room address. When composing a room-update message manually, use the protocol type `updateRoomInfo`.

## 7. MySQL integration

The repository includes a MySQL 8.0 and Adminer Compose environment:

```bash
docker compose -f test/docker/docker-compose.yml up -d
docker compose -f test/docker/docker-compose.yml ps
```

After MySQL is ready, copy `test/docker/config-mysql.json` to the repository-root `config.json`, or merge its `databaseConfig` field into your configuration.

The application does not support `-c`; this command is invalid:

```text
./page-spy-api -c test/docker/config-mysql.json
```

Start the application and verify upload, listing, and count endpoints. Adminer is available at:

```text
http://localhost:8080
```

Stop the environment:

```bash
docker compose -f test/docker/docker-compose.yml down
```

Delete the test database volume only when its data is no longer needed:

```bash
docker compose -f test/docker/docker-compose.yml down -v
```

## 8. S3 integration

Use an isolated bucket or S3-compatible test server. Configure `storageConfig`, then verify:

1. Uploading a log creates `<baseDir>/<logDir>/<fileId>`.
2. `/log/download` returns the exact uploaded bytes.
3. `/log/delete` removes both the object and metadata.
4. Restart behavior follows the current remote database-snapshot implementation.
5. Invalid endpoints, credentials, and missing objects produce diagnosable errors.

Never run deletion tests against a production bucket.

## 9. Multi-instance testing

Start at least two instances from separate working directories, each with its own `config.json`:

```text
node-a/config.json
node-b/config.json
```

Both configurations must contain an identical `rpcAddress` list. Their `selfRpcAddress` values must point to their respective nodes. HTTP and RPC ports must not conflict on the same host.

Verify:

1. Create a room on node A, then list and join it through node B.
2. Exchange broadcast and direct messages between clients on different nodes.
3. Upload a log on A, then query and download it through B.
4. Stop one node and inspect query and RPC failures.
5. Restart it and repeat the complete flow.

Use shared MySQL for multi-instance tests. Do not let multiple nodes maintain separate SQLite files while writing the same S3 database snapshot.

## 10. Minimum pre-commit checklist

```text
[ ] gofmt reports no files
[ ] git diff --check passes
[ ] tests for affected packages pass
[ ] concurrent changes pass go test -race
[ ] go vet passes
[ ] HTTP/WebSocket protocol changes have manual verification
[ ] configuration or data-model changes include migration notes
[ ] no real password, JWT secret, or S3 credential is committed
```

## 11. Troubleshooting

### `pattern dist/*: no matching files found`

Cause: the embed pattern in `bin/main.go` matched no files.

Fix: build page-spy-web and copy it into `bin/dist`, or temporarily create `bin/dist/index.html` for a backend build.

### Configuration changes are ignored

Check the service's working directory. It reads only a file named `config.json` from that directory and does not parse `-c`.

### MySQL is running but the application cannot connect

- Wait for container initialization to complete.
- Use `127.0.0.1` when the application runs on the host, or `mysql` when it runs in the same Compose network.
- Include `parseTime=True` in the DSN.

### The room address is rejected

Room addresses must have the form `<local-id>.<machine-id>` with exactly one separator. Use the address returned by the create-room API instead of inventing a plain room name.

### A protected endpoint accepts requests without a token

That is the current password-free behavior. Set `authConfig.password` or `AUTH_PASSWORD` to enable Bearer Token validation.
