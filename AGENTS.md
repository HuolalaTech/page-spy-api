# AGENTS.md

This file gives AI coding agents the repository-specific context needed to make safe, reviewable changes. It applies to the entire repository unless a deeper directory contains its own `AGENTS.md`.

## 1. Project summary

Page Spy API is the Go backend for [Page Spy Web](https://github.com/HuolalaTech/page-spy-web). It provides:

- HTTP APIs for rooms and debug logs.
- WebSocket-based real-time room messaging.
- SQLite or MySQL metadata persistence.
- Local filesystem or S3-compatible log-body storage.
- Internal JSON-RPC and HTTP proxying for multi-instance deployments.
- Optional hosting of the Page Spy Web frontend.

The module path is:

```text
github.com/HuolalaTech/page-spy-api
```

The repository uses Go `1.23` and declares the `go1.23.5` toolchain.

## 2. Read these documents first

English is the default documentation language:

- `README.md`: project overview and quick start.
- `docs/USAGE.md`: configuration, HTTP API, WebSocket protocol, and deployment.
- `docs/DEVELOPMENT.md`: build, test, integration-test, and troubleshooting workflows.
- `docs/ARCHITECTURE.md`: dependency graph, data flow, room model, RPC, and storage design.

Chinese equivalents use the `_ZH.md` suffix.

When implementation and documentation disagree, verify behavior from code and update both language versions in the same change.

## 3. Repository map

| Path | Responsibility |
| --- | --- |
| `api/event` | Shared event address, package, listener, and emitter contracts. |
| `api/room` | Shared room, connection, message, error, and interface contracts. |
| `bin` | Integrated executable that embeds `bin/dist/*`. |
| `config` | Configuration models, defaults, file loading, and auth environment overrides. |
| `container` | `go.uber.org/dig` dependency graph. |
| `data` | GORM models and SQLite/MySQL implementation. |
| `event` | Local event bus and remote event RPC adapter. |
| `proxy` | Cross-node HTTP reverse proxy. |
| `room` | Local rooms, remote rooms, managers, and room RPC services. |
| `rpc` | Node identity, RPC server/client, and cluster result aggregation. |
| `serve/middleware` | Request logging, CORS, authentication, errors, and cache. |
| `serve/route` | HTTP routes and log-domain orchestration. |
| `serve/socket` | WebSocket upgrade, sessions, and message loops. |
| `storage` | Local file and S3-compatible storage implementations. |
| `task` | Periodic background tasks. |
| `test/docker` | MySQL/Adminer manual integration environment. |
| `test/websocket_event_test` | Browser WebSocket smoke-test page. |

## 4. Before making changes

1. Run `git status --short`.
2. Preserve unrelated user changes; never reset or overwrite them.
3. Read the implementation, its interfaces, and all callers before changing a contract.
4. Identify whether the behavior is local-only or participates in multi-node RPC/proxy flows.
5. Check whether the change affects HTTP, WebSocket, JSON, database, storage, or configuration compatibility.
6. Plan tests before changing concurrent room, event, task, or socket code.

Prefer the smallest change that completely fixes the problem. Do not combine unrelated cleanup with a functional fix.

## 5. Build and test commands

### 5.1 Frontend embed requirement

`bin/main.go` contains:

```go
//go:embed dist/*
```

Commands that load `./bin` fail when `bin/dist` has no regular files. `bin/dist` is generated/ignored content and must not be committed.

With real frontend assets:

```bash
go build -o page-spy-api ./bin
go test ./...
go test -race ./...
go vet ./...
```

Without frontend assets, validate backend packages explicitly:

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

For concurrency-related changes, run the same list with `go test -race`. Run the same list with `go vet` for static analysis.

Format changed Go files:

```bash
gofmt -w path/to/file.go path/to/file_test.go
```

Always run:

```bash
git diff --check
```

Do not claim `go test ./...` passed if it failed only because `bin/dist/*` is missing. Report that limitation and the backend-package result separately.

## 6. Runtime and configuration rules

- The process reads `config.json` from its current working directory.
- The current implementation does not support a `-c` configuration flag.
- Missing `config.json` is created from `config/defaultConfig.json`.
- Authentication environment variables are `AUTH_PASSWORD`, `JWT_SECRET`, and `JWT_EXPIRATION_HOURS`.
- Authentication environment values may be written back to `config.json`.
- Never commit real passwords, JWT secrets, MySQL DSNs, S3 credentials, or production addresses.
- Local runtime data is stored in `data/data.db`, `data.db`, and `log/`.
- The presence of `storageConfig`, even if empty, selects remote S3 storage.
- A non-empty `databaseConfig.mysqlUrl` selects MySQL; otherwise SQLite is used.

Configuration changes are compatibility-sensitive. Preserve existing JSON field names and defaults unless the task explicitly includes a migration.

## 7. Important architecture contracts

### 7.1 Dependency injection and lifecycle

The project uses a process-global `dig.Container`.

Several constructors have side effects:

- `rpc.NewRpcManager` starts an RPC listener.
- `socket.NewManager` starts room managers and registers RPC services.
- `data.NewData` may restore data and start a sync task.
- `route.NewCore` registers cleanup tasks and an RPC service.
- `serve.Run` starts the Echo server.

Do not treat these constructors as pure functions in tests. Avoid accidentally constructing multiple listeners or leaking goroutines.

### 7.2 Address and identifier formats

Preserve these formats:

```text
room/connection address: <local-id>.<machine-id>
log file ID:            <machine-id>.<md5>
```

Single-instance machine ID is `local`. Multi-instance IDs (`A0`, `A1`, and so on) are derived from the sorted RPC address list.

Changing these formats affects routing, persisted data, API clients, and multi-node compatibility.

### 7.3 HTTP API

Public routes include authentication verification, room creation/check/join, and log uploads. Management routes use JWT middleware only when a password is configured.

Preserve:

- `/api/v1` route paths and methods.
- `common.Response` JSON shape.
- Query parameter names.
- One-based pagination.
- Unix-second semantics for `from` and `to`.
- Repeated query parameters for multi-delete.

Do not silently move a route between public and protected groups. Authentication-default changes require an explicit migration and release note.

### 7.4 WebSocket protocol

Client-sendable message types are:

```text
ping
broadcast
message
updateRoomInfo
```

Server messages also include:

```text
connect
join
leave
start
close
pong
error
```

When adding or changing a message:

1. Update types in `api/room/room.go`.
2. Update `NewMessageContent`.
3. Review `IsPublicMessageType`.
4. Review both `localRoom` and `remoteRoom`.
5. Verify event-package serialization.
6. Add local, cross-node, malformed-input, and shutdown tests.

### 7.5 Room and event concurrency

Rooms, event listeners, WebSocket reader/writer loops, and background cleanup run concurrently.

- Never close a channel from multiple paths without `sync.Once` or equivalent ownership.
- Never return a mutable internal slice, map, pointer, or nested struct without considering aliasing.
- Keep locks out of network calls and potentially blocking channel sends.
- Use snapshots before iterating shared collections.
- Make cleanup and `Close` operations idempotent.
- Ensure context cancellation can unblock reads and writes.
- Run `go test -race` for every concurrency change.

### 7.6 Data and storage consistency

`data.DataApi` owns metadata; `storage.StorageApi` owns log bodies.

The current create flow saves the body before metadata. The delete flow removes the body before metadata. There is no transaction across those boundaries.

When changing these flows:

- Define compensation for partial failure.
- Preserve stable file IDs and object keys.
- Do not hide missing-object errors as successful writes.
- Close all readers returned by storage.
- Test SQLite and MySQL dialect-specific SQL separately.
- Test local and S3-compatible storage separately.

Multi-instance deployments should use shared MySQL. Do not introduce new writes that let multiple SQLite instances overwrite one shared S3 snapshot.

### 7.7 Cluster behavior

- All nodes must use the same ordered `rpcAddress` set.
- All nodes must use the same HTTP port because the proxy combines peer IPs with the local configured port.
- Room and event routing use RPC.
- Log list aggregation uses RPC.
- Remote log download and deletion use HTTP proxying.

Every cluster change needs at least a two-node scenario analysis. Check loops, partial failures, ordering, pagination, and deduplication.

## 8. Implementation guidelines

- Return errors for invalid external input; do not panic.
- Wrap errors with useful operation context while avoiding secrets.
- Do not log passwords, JWTs, authorization headers, or full credential-bearing DSNs.
- Use constant-time comparison for secrets where appropriate.
- Bound request bodies, message sizes, and blocking work when compatibility requirements allow it.
- Reuse concurrency-safe clients such as database and S3 clients instead of constructing them per operation.
- Prefer interfaces already present at package boundaries.
- Keep transport parsing in `serve`, domain behavior in `room`/`event`/`serve/route`, and persistence in `data`/`storage`.
- Preserve public function names, including legacy misspellings, unless a compatibility plan exists.
- Add a new correctly named API as a wrapper before deprecating an old exported name.

## 9. Test expectations

Add regression tests for every bug fix.

Minimum expectations by change:

| Change | Required verification |
| --- | --- |
| Pure helper or validation | Focused unit tests and affected-package tests. |
| HTTP middleware/route | Success, invalid input, auth, and response-shape tests. |
| WebSocket/room/event | Unit tests, shutdown/error tests, and `go test -race`. |
| Database | SQLite tests plus MySQL integration for dialect-specific behavior. |
| Storage | Local fake/unit tests plus isolated S3 integration when applicable. |
| RPC/cluster | Local RPC test and a documented two-node scenario. |
| Configuration | Default, missing, override, and invalid-value tests. |

Avoid long fixed sleeps. Prefer channels, wait groups, explicit hooks, and context deadlines.

## 10. Documentation policy

English is the default:

```text
README.md
docs/USAGE.md
docs/DEVELOPMENT.md
docs/ARCHITECTURE.md
```

Chinese translations:

```text
README_ZH.md
docs/USAGE_ZH.md
docs/DEVELOPMENT_ZH.md
docs/ARCHITECTURE_ZH.md
```

For any user-visible behavior, configuration, API, build, or architecture change:

1. Update the English document.
2. Update the matching Chinese document in the same commit.
3. Keep headings and examples aligned.
4. Keep language-switch links valid.
5. Do not document a command until it has been verified against the current code.

## 11. Generated and sensitive files

Do not commit:

- `bin/dist/`
- built binaries
- `data.db` or `data/`
- `log/`
- debug binaries
- vendor output unless the task explicitly requires vendoring
- configuration files containing real credentials

Avoid destructive cleanup of these paths unless the user explicitly requests it and the exact target has been verified.

## 12. Definition of done

Before handing off a change:

```text
[ ] Scope matches the user request
[ ] Unrelated worktree changes are preserved
[ ] Public compatibility was evaluated
[ ] Errors and cleanup paths were reviewed
[ ] Changed Go files are gofmt-formatted
[ ] Focused tests pass
[ ] Race tests pass for concurrent code
[ ] go vet passes for affected packages
[ ] git diff --check passes
[ ] English and Chinese docs are synchronized
[ ] No secrets or generated runtime data are included
[ ] Known validation limitations are reported explicitly
```
