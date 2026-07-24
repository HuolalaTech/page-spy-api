# Page Spy API User Guide

English | [中文](./USAGE_ZH.md)

This guide is for operators, frontend integrators, and API consumers. It covers building, configuring, running, authenticating, and integrating with the Page Spy HTTP and WebSocket APIs.

## 1. What the service provides

Page Spy API is the backend service for PageSpy. It provides:

- Room creation, discovery, and real-time WebSocket message forwarding.
- Debug-log upload, search, download, grouping, and deletion.
- SQLite or MySQL metadata storage.
- Local filesystem or S3-compatible object storage.
- HTTP JSON-RPC communication and request proxying between instances.
- Optional hosting of the Page Spy Web frontend.

The default HTTP port is `6752`. All HTTP APIs are under `/api/v1`.

## 2. Quick start

### 2.1 Requirements

- Go `1.23`; the repository declares the `go1.23.5` toolchain.
- To build the complete service with its web UI, first build
  [page-spy-web](https://github.com/HuolalaTech/page-spy-web).
- MySQL and S3-compatible services are optional and only required when enabled.

Download dependencies:

```bash
go mod download
```

### 2.2 Prepare frontend assets

`bin/main.go` uses `//go:embed dist/*`. Therefore, `bin/dist` must contain at least one regular file. Otherwise, commands that load the `bin` package, including `go build ./...` and `go test ./...`, fail.

For a complete deployment, copy the page-spy-web build output to:

```text
bin/dist/
├── index.html
└── assets/
```

For a backend-only build check, you may temporarily add `bin/dist/index.html`. A placeholder only satisfies the embed pattern; it does not provide the real management UI.

### 2.3 Build

From the repository root:

```bash
go build -o page-spy-api ./bin
```

The repository's `build.sh` builds `bin/main.go` directly. Using `-o` is recommended so the output name is explicit.

### 2.4 Run

The service always reads `config.json` from its current working directory. This version does not support a `-c` flag or another CLI configuration path.

Run from the repository root:

```bash
./page-spy-api
```

If `config.json` does not exist, the service creates:

```json
{
  "port": "6752"
}
```

The service is then available at:

```text
http://localhost:6752
```

When a complete `bin/dist` is embedded, `/` serves Page Spy Web. Otherwise, use the `/api/v1` APIs directly.

## 3. Configuration

### 3.1 Complete example

```json
{
  "port": "6752",
  "debug": false,
  "notAllowedDeleteLog": false,
  "maxRoomNumber": 500,
  "maxLogFileSizeOfMB": 10240,
  "maxLogLifeTimeOfHour": 720,
  "corsConfig": {
    "allowOrigins": ["https://pagespy.example.com"],
    "allowMethods": ["GET", "POST", "PUT", "DELETE", "OPTIONS"],
    "allowHeaders": ["Authorization", "Content-Type", "X-Request-ID"],
    "exposeHeaders": ["X-Request-ID"]
  },
  "authConfig": {
    "password": "replace-with-a-strong-password",
    "jwtSecret": "replace-with-a-long-random-secret",
    "tokenExpiration": 24
  },
  "databaseConfig": {
    "mysqlUrl": ""
  }
}
```

### 3.2 Configuration reference

| Field | Default | Description |
| --- | --- | --- |
| `port` | `6752` | HTTP and WebSocket port. |
| `debug` | `false` | Enables GORM database logging. |
| `notAllowedDeleteLog` | `false` | Rejects log and log-group deletion when `true`. |
| `maxRoomNumber` | `500` | Maximum number of local rooms per instance. Values at or below zero use the default. |
| `maxLogFileSizeOfMB` | `10240` | Total local log capacity in MB. |
| `maxLogLifeTimeOfHour` | `720` | Maximum local log age in hours. |
| `corsConfig` | unset | All origins are accepted when unset; otherwise the configured CORS lists are used. |
| `authConfig.password` | empty | Password for protected APIs. Protected routes bypass authentication when empty. |
| `authConfig.jwtSecret` | temporary random value | JWT signing secret. Set a stable value in production. |
| `authConfig.tokenExpiration` | `24` | JWT lifetime in hours. |
| `databaseConfig.mysqlUrl` | empty | Uses SQLite when empty and MySQL when set. |
| `storageConfig` | unset | Uses `./log` when unset. The presence of this object enables S3 storage. |
| `rpcAddress` | empty | RPC nodes for a multi-instance deployment. Empty means single-instance mode. |
| `selfRpcAddress` | auto-detected | Address of the current node within `rpcAddress`. |

### 3.3 Authentication environment variables

Authentication settings can be supplied through environment variables:

| Environment variable | Configuration field |
| --- | --- |
| `AUTH_PASSWORD` | `authConfig.password` |
| `JWT_SECRET` | `authConfig.jwtSecret` |
| `JWT_EXPIRATION_HOURS` | `authConfig.tokenExpiration` |

Example:

```bash
AUTH_PASSWORD='change-me' \
JWT_SECRET='change-this-to-a-long-random-value' \
JWT_EXPIRATION_HOURS='24' \
./page-spy-api
```

The current implementation writes environment-derived authentication settings back to `config.json`. Restrict access to that file and never commit real passwords or JWT secrets.

### 3.4 MySQL

MySQL DSN format:

```text
username:password@tcp(host:port)/database?charset=utf8mb4&parseTime=True&loc=Local
```

Example:

```json
{
  "databaseConfig": {
    "mysqlUrl": "pagespy:pagespy123@tcp(127.0.0.1:3306)/pagespy?charset=utf8mb4&parseTime=True&loc=Local"
  }
}
```

Without a MySQL DSN, SQLite is used. Its default path is `data/data.db`. If `data.db` already exists in the working directory, that file takes precedence.

### 3.5 S3-compatible object storage

Adding `storageConfig` stores log bodies in S3:

```json
{
  "storageConfig": {
    "logDir": "log",
    "baseDir": "page-spy",
    "keyId": "ACCESS_KEY",
    "secret": "SECRET_KEY",
    "region": "us-east-1",
    "endpoint": "https://s3.example.com",
    "bucket": "page-spy",
    "s3ForcePathStyle": true
  }
}
```

Log object keys have the form:

```text
<baseDir>/<logDir>/<fileId>
```

Do not add an empty `storageConfig` as a placeholder. Its presence switches the application to remote storage.

### 3.6 Multi-instance deployment

Example:

```json
{
  "port": "6752",
  "rpcAddress": [
    {"ip": "10.0.0.11", "port": "7752"},
    {"ip": "10.0.0.12", "port": "7752"}
  ],
  "selfRpcAddress": {
    "ip": "10.0.0.11",
    "port": "7752"
  },
  "databaseConfig": {
    "mysqlUrl": "pagespy:password@tcp(mysql:3306)/pagespy?charset=utf8mb4&parseTime=True&loc=Local"
  }
}
```

All nodes must use the same `rpcAddress` list and the same HTTP `port`. Each node's `selfRpcAddress` must appear in the list. Production clusters should use shared MySQL and expose RPC ports only on a trusted network.

## 4. HTTP response format

Success:

```json
{
  "code": "success",
  "data": {},
  "success": true,
  "message": ""
}
```

Failure:

```json
{
  "code": "error",
  "data": null,
  "success": false,
  "message": "error detail"
}
```

Most API errors return HTTP `400`. Missing, malformed, or expired Bearer Tokens return HTTP `401`.

## 5. Authentication

### 5.1 Obtain a token

```bash
curl -sS \
  -H 'Content-Type: application/json' \
  -d '{"password":"change-me"}' \
  http://localhost:6752/api/v1/auth/verify
```

The successful response contains a Bearer Token in `data.token`:

```json
{
  "code": "success",
  "data": {
    "message": "Authentication successful",
    "token": "<jwt>",
    "expiresIn": 86400
  },
  "success": true,
  "message": ""
}
```

### 5.2 Call a protected endpoint

```bash
curl -sS \
  -H "Authorization: Bearer <jwt>" \
  'http://localhost:6752/api/v1/room/list'
```

When `authConfig.password` is unset, protected routes are allowed without a token and `/auth/verify` returns `PASSWORD_REQUIRED`. Configure authentication explicitly before exposing the service to an untrusted network.

## 6. Room API

### 6.1 Create a room

```bash
curl -sS \
  -X POST \
  -H 'Content-Type: application/json' \
  -d '{"useSecret":true,"secret":"room-password"}' \
  'http://localhost:6752/api/v1/room/create?name=demo&group=default&env=test'
```

The returned `data.address` is the room address:

```text
<room-local-id>.<machine-id>
```

The machine ID is `local` in single-instance mode.

### 6.2 Check a room password

```bash
curl -sS \
  'http://localhost:6752/api/v1/room/check?address=<room-address>&secret=room-password'
```

### 6.3 List rooms

```bash
curl -sS \
  -H "Authorization: Bearer <jwt>" \
  'http://localhost:6752/api/v1/room/list?env=test'
```

Query parameters become room-tag filters. All filters must match, and values use case-insensitive substring matching.

## 7. WebSocket integration

Endpoint:

```text
ws://localhost:6752/api/v1/ws/room/join
```

Query parameters:

| Parameter | Required | Description |
| --- | --- | --- |
| `address` | yes | Room address returned by the room creation endpoint. |
| `name` | no | Display name for this connection. |
| `userId` | no | Application-level user ID. |
| `group` | no | Room group. |
| `secret` | for password rooms | Room password. |
| `forceCreate` | no | When `true`, attempts to create a missing room. |
| `useSecret` | with `forceCreate` | Creates a password-protected room when `true`. |

Connect with `websocat`:

```bash
websocat \
  'ws://localhost:6752/api/v1/ws/room/join?address=<room-address>&name=debugger&userId=user-1&secret=room-password'
```

After a successful connection, the server sends a `connect` message containing the current connection and existing room connections.

### 7.1 Ping

```json
{
  "type": "ping",
  "createdAt": 1753344000000,
  "requestId": "ping-1",
  "content": {}
}
```

The server returns `pong` with the same `requestId`.

### 7.2 Broadcast

```json
{
  "type": "broadcast",
  "createdAt": 1753344000000,
  "requestId": "broadcast-1",
  "content": {
    "data": {"event":"refresh"},
    "includeSelf": false
  }
}
```

The server sets `from` from the current WebSocket connection.

### 7.3 Direct message

Use a connection from the `connect` or `join` message as the target:

```json
{
  "type": "message",
  "createdAt": 1753344000000,
  "requestId": "message-1",
  "content": {
    "data": {"command":"inspect"},
    "to": {
      "address": "<target-connection-address>",
      "userId": "target-user",
      "name": "target"
    }
  }
}
```

### 7.4 Update room information

```json
{
  "type": "updateRoomInfo",
  "createdAt": 1753344000000,
  "requestId": "update-1",
  "content": {
    "info": {
      "name": "new-name",
      "group": "new-group",
      "tags": {
        "env": "staging"
      }
    }
  }
}
```

Clients may send only:

- `ping`
- `broadcast`
- `message`
- `updateRoomInfo`

The server may also send `connect`, `join`, `leave`, `start`, `close`, `pong`, and `error`.

## 8. Log API

“Protected” means a Bearer Token is required when a system password is configured.

| Method | Path | Access | Description |
| --- | --- | --- | --- |
| `POST` | `/api/v1/log/upload` | public | Upload one multipart log. |
| `POST` | `/api/v1/jsonLog/upload` | public | Upload a JSON or text request body. |
| `POST` | `/api/v1/logGroup/upload` | public | Upload a multipart file to a log group. |
| `GET` | `/api/v1/log/list` | protected | List logs with pagination. |
| `GET` | `/api/v1/logGroup/list` | protected | List log groups with pagination. |
| `GET` | `/api/v1/logGroup/files` | protected | List files in a log group. |
| `GET` | `/api/v1/log/count` | protected | Count logs by month and tag. |
| `GET` | `/api/v1/log/download` | protected | Download a log body. |
| `DELETE` | `/api/v1/log/delete` | protected | Delete one or more logs. |
| `DELETE` | `/api/v1/logGroup/delete` | protected | Delete one or more log groups. |

### 8.1 Upload

Multipart:

```bash
curl -sS \
  -F 'log=@./debug.json' \
  'http://localhost:6752/api/v1/log/upload?env=test&device=android'
```

Raw request body:

```bash
curl -sS \
  -H 'Content-Type: application/json' \
  --data-binary '@./debug.json' \
  'http://localhost:6752/api/v1/jsonLog/upload?name=debug.json&env=test'
```

Upload to a group:

```bash
curl -sS \
  -F 'log=@./network.json' \
  'http://localhost:6752/api/v1/logGroup/upload?groupId=session-001&env=test'
```

Query parameters other than `page`, `size`, `from`, and `to` are stored or matched as log tags.

### 8.2 Query

```bash
curl -sS \
  -H "Authorization: Bearer <jwt>" \
  'http://localhost:6752/api/v1/log/list?page=1&size=20&env=test'
```

`page` and `size` must be positive integers. Optional `from` and `to` values are Unix timestamps in seconds:

```bash
curl -sS \
  -H "Authorization: Bearer <jwt>" \
  'http://localhost:6752/api/v1/log/list?page=1&size=20&from=1751328000&to=1754006400'
```

List groups and group files:

```bash
curl -sS \
  -H "Authorization: Bearer <jwt>" \
  'http://localhost:6752/api/v1/logGroup/list?page=1&size=20'

curl -sS \
  -H "Authorization: Bearer <jwt>" \
  'http://localhost:6752/api/v1/logGroup/files?groupId=session-001'
```

### 8.3 Download and delete

```bash
curl -fL \
  -H "Authorization: Bearer <jwt>" \
  -o debug.json \
  'http://localhost:6752/api/v1/log/download?fileId=<file-id>'
```

Repeat `fileId` to delete multiple logs:

```bash
curl -sS \
  -X DELETE \
  -H "Authorization: Bearer <jwt>" \
  'http://localhost:6752/api/v1/log/delete?fileId=<id-1>&fileId=<id-2>'
```

Delete a log group:

```bash
curl -sS \
  -X DELETE \
  -H "Authorization: Bearer <jwt>" \
  'http://localhost:6752/api/v1/logGroup/delete?groupId=session-001'
```

Both delete endpoints reject requests when `notAllowedDeleteLog=true`.

## 9. Runtime data and maintenance

Local mode creates:

```text
data/data.db     SQLite metadata
log/<fileId>     log bodies
```

The local cleanup task runs every ten minutes:

- When total size exceeds `maxLogFileSizeOfMB`, it deletes the oldest logs first.
- It deletes logs older than `maxLogLifeTimeOfHour`.

In remote-storage mode, the service attempts to restore a SQLite file from object storage and syncs the local database file every five minutes. Do not let several instances overwrite the same SQLite snapshot; use shared MySQL for a multi-instance deployment.

## 10. Production checklist

- Set `AUTH_PASSWORD` and a stable, sufficiently long `JWT_SECRET`.
- Put HTTP and RPC services behind a reverse proxy or a trusted network boundary.
- Terminate HTTP and WebSocket traffic with TLS.
- Apply body-size, concurrency, and rate limits to public upload endpoints.
- Avoid retaining room passwords in URLs, access logs, or monitoring systems.
- Use shared MySQL for multiple instances and keep every RPC address list identical.
- Back up the database and configure separate lifecycle and access policies for the S3 bucket.
- Restrict access to `config.json`, which contains plaintext credentials.
