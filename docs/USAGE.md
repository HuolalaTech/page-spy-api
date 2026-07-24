# Page Spy API 使用指南

本文面向部署人员、前端接入方和 API 使用方，说明 Page Spy API 的构建、配置、启动、认证、HTTP API 与 WebSocket 接入方式。

## 1. 服务能力

Page Spy API 是 PageSpy 的后端服务，提供：

- 房间创建、查询和 WebSocket 实时消息转发。
- 调试日志上传、查询、下载、分组和删除。
- SQLite 或 MySQL 元数据存储。
- 本地文件系统或 S3 兼容对象存储。
- 多实例之间的 HTTP JSON-RPC 通信和请求代理。
- 可选的 Page Spy Web 静态资源托管。

默认 HTTP 端口为 `6752`，所有 HTTP API 都位于 `/api/v1` 下。

## 2. 快速开始

### 2.1 环境要求

- Go `1.23`；仓库声明的工具链版本为 `go1.23.5`。
- 如需构建包含管理页面的完整服务，需要先构建
  [page-spy-web](https://github.com/HuolalaTech/page-spy-web)。
- 如需使用 MySQL 或 S3，需要准备对应的外部服务。

安装依赖：

```bash
go mod download
```

### 2.2 准备前端静态资源

`bin/main.go` 使用 `//go:embed dist/*`，因此 `bin/dist` 中必须至少存在一个普通文件，否则 `go build ./...` 和 `go test ./...` 会在加载 `bin` 包时失败。

完整部署时，将 page-spy-web 的构建产物复制到：

```text
bin/dist/
├── index.html
└── assets/
```

只验证后端构建时，也可以临时放置一个 `bin/dist/index.html`；该文件只用于满足嵌入要求，不会提供完整管理页面。

### 2.3 构建

推荐从仓库根目录执行：

```bash
go build -o page-spy-api ./bin
```

仓库中的 `build.sh` 等价于直接构建 `bin/main.go`，但默认输出文件名由 Go 决定。显式指定 `-o` 更方便部署。

### 2.4 启动

服务固定读取当前工作目录下的 `config.json`，当前版本没有 `-c` 或其他命令行配置参数。

从仓库根目录启动：

```bash
./page-spy-api
```

如果当前目录不存在 `config.json`，服务会自动创建以下默认配置：

```json
{
  "port": "6752"
}
```

启动成功后访问：

```text
http://localhost:6752
```

有完整 `bin/dist` 时，根路径提供 Page Spy Web 页面；否则只使用 `/api/v1` API。

## 3. 配置

### 3.1 完整配置示例

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

### 3.2 配置项

| 配置项 | 默认值 | 说明 |
| --- | --- | --- |
| `port` | `6752` | HTTP 与 WebSocket 服务端口。 |
| `debug` | `false` | 开启后 GORM 输出数据库日志。 |
| `notAllowedDeleteLog` | `false` | 为 `true` 时禁止日志和日志组删除。 |
| `maxRoomNumber` | `500` | 单实例最大本地房间数。小于等于 0 时使用默认值。 |
| `maxLogFileSizeOfMB` | `10240` | 本地日志总容量上限，单位 MB。 |
| `maxLogLifeTimeOfHour` | `720` | 本地日志最长保留时间，单位小时。 |
| `corsConfig` | 未设置 | 未设置时允许任意 Origin；设置后使用给定 CORS 列表。 |
| `authConfig.password` | 空 | 管理 API 密码；为空时受保护路由会跳过认证。 |
| `authConfig.jwtSecret` | 临时随机值 | JWT 签名密钥。生产环境应显式设置并保持稳定。 |
| `authConfig.tokenExpiration` | `24` | JWT 有效期，单位小时。 |
| `databaseConfig.mysqlUrl` | 空 | 为空时使用 SQLite；非空时使用 MySQL DSN。 |
| `storageConfig` | 未设置 | 未设置时使用 `./log`；只要设置该对象，就启用 S3 存储。 |
| `rpcAddress` | 空 | 多实例 RPC 节点列表。为空时使用单实例模式。 |
| `selfRpcAddress` | 自动识别 | 当前节点在 `rpcAddress` 中的地址。 |

### 3.3 认证环境变量

认证配置可以由环境变量覆盖：

| 环境变量 | 对应配置 |
| --- | --- |
| `AUTH_PASSWORD` | `authConfig.password` |
| `JWT_SECRET` | `authConfig.jwtSecret` |
| `JWT_EXPIRATION_HOURS` | `authConfig.tokenExpiration` |

示例：

```bash
AUTH_PASSWORD='change-me' \
JWT_SECRET='change-this-to-a-long-random-value' \
JWT_EXPIRATION_HOURS='24' \
./page-spy-api
```

注意：当前实现会把环境变量生成的认证配置写回工作目录的 `config.json`。请限制配置文件访问权限，不要把真实密码和 JWT 密钥提交到版本库。

### 3.4 MySQL

MySQL DSN 格式：

```text
用户名:密码@tcp(主机:端口)/数据库名?charset=utf8mb4&parseTime=True&loc=Local
```

示例：

```json
{
  "databaseConfig": {
    "mysqlUrl": "pagespy:pagespy123@tcp(127.0.0.1:3306)/pagespy?charset=utf8mb4&parseTime=True&loc=Local"
  }
}
```

未配置 DSN 时，SQLite 文件默认位于 `data/data.db`。如果工作目录已经存在 `data.db`，服务会优先使用该文件。

### 3.5 S3 兼容对象存储

配置 `storageConfig` 后，日志正文保存到 S3：

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

对象 key 结构为：

```text
<baseDir>/<logDir>/<fileId>
```

`storageConfig` 不能作为空对象占位；只要存在就会切换到远程存储。

### 3.6 多实例

示例：

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

所有节点必须使用相同的 `rpcAddress` 列表和 HTTP `port`，且当前节点的 `selfRpcAddress` 必须能在列表中找到。生产多实例部署应使用共享 MySQL，并确保 RPC 端口只在可信网络内可达。

## 4. HTTP 响应格式

成功响应：

```json
{
  "code": "success",
  "data": {},
  "success": true,
  "message": ""
}
```

失败响应：

```json
{
  "code": "error",
  "data": null,
  "success": false,
  "message": "error detail"
}
```

普通 API 错误通常返回 HTTP `400`；缺失、格式错误或过期的 Bearer Token 返回 HTTP `401`。

## 5. 认证

### 5.1 获取 Token

```bash
curl -sS \
  -H 'Content-Type: application/json' \
  -d '{"password":"change-me"}' \
  http://localhost:6752/api/v1/auth/verify
```

成功响应中的 `data.token` 是 Bearer Token：

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

### 5.2 调用受保护接口

```bash
curl -sS \
  -H "Authorization: Bearer <jwt>" \
  'http://localhost:6752/api/v1/room/list'
```

未设置 `authConfig.password` 时，受保护接口会直接放行，此时 `/auth/verify` 会返回 `PASSWORD_REQUIRED`。公网部署前应显式配置认证，并通过反向代理进一步限制访问。

## 6. 房间 API

### 6.1 创建房间

```bash
curl -sS \
  -X POST \
  -H 'Content-Type: application/json' \
  -d '{"useSecret":true,"secret":"room-password"}' \
  'http://localhost:6752/api/v1/room/create?name=demo&group=default&env=test'
```

返回的 `data.address` 是房间地址，格式为：

```text
<room-local-id>.<machine-id>
```

单实例的 machine ID 为 `local`。

### 6.2 检查房间密码

```bash
curl -sS \
  'http://localhost:6752/api/v1/room/check?address=<room-address>&secret=room-password'
```

### 6.3 查询房间

```bash
curl -sS \
  -H "Authorization: Bearer <jwt>" \
  'http://localhost:6752/api/v1/room/list?env=test'
```

查询参数会作为房间 tag 过滤条件，多个 tag 之间是“同时满足”关系，value 使用不区分大小写的包含匹配。

## 7. WebSocket 接入

连接地址：

```text
ws://localhost:6752/api/v1/ws/room/join
```

查询参数：

| 参数 | 必填 | 说明 |
| --- | --- | --- |
| `address` | 是 | 创建房间后返回的房间地址。 |
| `name` | 否 | 当前连接显示名称。 |
| `userId` | 否 | 业务侧用户标识。 |
| `group` | 否 | 房间分组。 |
| `secret` | 密码房间必填 | 房间密码。 |
| `forceCreate` | 否 | 为 `true` 时，房间不存在则尝试创建。 |
| `useSecret` | forceCreate 时可用 | 为 `true` 时创建密码房间。 |

使用 `websocat` 连接：

```bash
websocat \
  'ws://localhost:6752/api/v1/ws/room/join?address=<room-address>&name=debugger&userId=user-1&secret=room-password'
```

连接成功后，服务首先发送 `connect` 消息，其中包含当前连接和房间内已有连接。

### 7.1 Ping

```json
{
  "type": "ping",
  "createdAt": 1753344000000,
  "requestId": "ping-1",
  "content": {}
}
```

服务使用相同 `requestId` 返回 `pong`。

### 7.2 广播

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

`from` 由服务端根据当前 WebSocket 连接写入。

### 7.3 单播

目标地址来自 `connect`、`join` 或房间连接列表中的 connection：

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

### 7.4 更新房间信息

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

客户端可发送的消息类型只有：

- `ping`
- `broadcast`
- `message`
- `updateRoomInfo`

服务端还会发送 `connect`、`join`、`leave`、`start`、`close`、`pong` 和 `error`。

## 8. 日志 API

下表中的“鉴权”表示设置系统密码后需要 Bearer Token。

| 方法 | 路径 | 鉴权 | 说明 |
| --- | --- | --- | --- |
| `POST` | `/api/v1/log/upload` | 否 | multipart 上传单个日志。 |
| `POST` | `/api/v1/jsonLog/upload` | 否 | 请求体直接上传 JSON/文本日志。 |
| `POST` | `/api/v1/logGroup/upload` | 否 | multipart 上传日志组文件。 |
| `GET` | `/api/v1/log/list` | 是 | 分页查询日志。 |
| `GET` | `/api/v1/logGroup/list` | 是 | 分页查询日志组。 |
| `GET` | `/api/v1/logGroup/files` | 是 | 查询日志组内文件。 |
| `GET` | `/api/v1/log/count` | 是 | 按月份和指定 tag 统计日志。 |
| `GET` | `/api/v1/log/download` | 是 | 下载日志正文。 |
| `DELETE` | `/api/v1/log/delete` | 是 | 删除一个或多个日志。 |
| `DELETE` | `/api/v1/logGroup/delete` | 是 | 删除一个或多个日志组。 |

### 8.1 上传日志

multipart：

```bash
curl -sS \
  -F 'log=@./debug.json' \
  'http://localhost:6752/api/v1/log/upload?env=test&device=android'
```

直接上传请求体：

```bash
curl -sS \
  -H 'Content-Type: application/json' \
  --data-binary '@./debug.json' \
  'http://localhost:6752/api/v1/jsonLog/upload?name=debug.json&env=test'
```

上传到日志组：

```bash
curl -sS \
  -F 'log=@./network.json' \
  'http://localhost:6752/api/v1/logGroup/upload?groupId=session-001&env=test'
```

除 `page`、`size`、`from` 和 `to` 外，查询参数会作为日志 tag 保存或过滤。

### 8.2 查询

```bash
curl -sS \
  -H "Authorization: Bearer <jwt>" \
  'http://localhost:6752/api/v1/log/list?page=1&size=20&env=test'
```

`page` 和 `size` 必须是大于 0 的整数。可选的 `from`、`to` 是 Unix 秒时间戳：

```bash
curl -sS \
  -H "Authorization: Bearer <jwt>" \
  'http://localhost:6752/api/v1/log/list?page=1&size=20&from=1751328000&to=1754006400'
```

查询日志组及其文件：

```bash
curl -sS \
  -H "Authorization: Bearer <jwt>" \
  'http://localhost:6752/api/v1/logGroup/list?page=1&size=20'

curl -sS \
  -H "Authorization: Bearer <jwt>" \
  'http://localhost:6752/api/v1/logGroup/files?groupId=session-001'
```

### 8.3 下载和删除

```bash
curl -fL \
  -H "Authorization: Bearer <jwt>" \
  -o debug.json \
  'http://localhost:6752/api/v1/log/download?fileId=<file-id>'
```

删除多个日志时重复传递 `fileId`：

```bash
curl -sS \
  -X DELETE \
  -H "Authorization: Bearer <jwt>" \
  'http://localhost:6752/api/v1/log/delete?fileId=<id-1>&fileId=<id-2>'
```

删除日志组：

```bash
curl -sS \
  -X DELETE \
  -H "Authorization: Bearer <jwt>" \
  'http://localhost:6752/api/v1/logGroup/delete?groupId=session-001'
```

`notAllowedDeleteLog=true` 时，两个删除接口都会拒绝请求。

## 9. 运行数据与维护

本地模式会生成：

```text
data/data.db     SQLite 元数据
log/<fileId>     日志正文
```

本地日志清理任务每 10 分钟执行一次：

- 总大小超过 `maxLogFileSizeOfMB` 时，从最旧日志开始删除。
- 创建时间超过 `maxLogLifeTimeOfHour` 时删除。

远程存储模式下，服务会尝试从对象存储恢复 SQLite 文件，并每 5 分钟同步一次本地数据库文件。多实例部署不要依赖多个节点各自上传 SQLite 快照，应使用共享 MySQL。

## 10. 生产部署注意事项

- 显式设置 `AUTH_PASSWORD` 和稳定、足够长的 `JWT_SECRET`。
- 将 HTTP 与 RPC 服务放在反向代理或可信网络之后。
- 使用 TLS 终止 WebSocket 和 HTTP 流量。
- 限制公开上传接口的请求大小、并发和速率。
- 不要在 URL、访问日志或监控系统中长期保留房间密码。
- 多实例使用共享 MySQL；所有节点保持相同的 RPC 地址列表。
- 定期备份数据库，并为 S3 bucket 配置独立的生命周期和访问控制。
- `config.json` 包含明文凭据，应限制文件权限并排除在配置分发日志之外。
