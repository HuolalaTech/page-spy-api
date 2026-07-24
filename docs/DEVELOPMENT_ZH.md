# Page Spy API 开发与测试指南

[English](./DEVELOPMENT.md) | 中文

本文面向项目开发者，说明本地环境、构建方式、代码组织约定、测试命令、手工验证和常见问题。

## 1. 开发环境

### 1.1 必需工具

- Go `1.23`，推荐使用 `go.mod` 声明的 `go1.23.5` 工具链。
- Git。
- 可选：Docker 与 Docker Compose，用于 MySQL 集成验证。
- 可选：`curl`、`websocat`，用于 HTTP 和 WebSocket 手工验证。

检查环境：

```bash
go version
git version
docker version
docker compose version
```

### 1.2 初始化

```bash
git clone https://github.com/HuolalaTech/page-spy-api.git
cd page-spy-api
go mod download
```

服务运行时会在当前目录读取或创建 `config.json`，并可能生成：

```text
data/data.db
log/
```

这些是运行数据，不应作为源码提交。

## 2. 本地构建与运行

### 2.1 完整入口

`bin/main.go` 会嵌入 `bin/dist/*`。先把 page-spy-web 的构建产物复制到 `bin/dist`，再执行：

```bash
go build -o page-spy-api ./bin
./page-spy-api
```

如果仅需验证 Go 后端，可以临时准备最小静态文件：

```bash
mkdir -p bin/dist
printf '<!doctype html><title>Page Spy API</title>\n' > bin/dist/index.html
go build -o page-spy-api ./bin
```

`bin/dist` 在 `.gitignore` 中，不应把临时文件提交到仓库。

### 2.2 作为库嵌入

Page Spy API 的启动入口可以由其他 Go 程序提供。依赖注入容器要求注册一个 `*config.StaticConfig` provider；纯后端模式返回 `nil`：

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

如果需要托管前端，将 `StaticConfig.Files` 设置为包含 `dist` 目录的 `fs.FS`。

### 2.3 Debug 模式

在 `config.json` 中设置：

```json
{
  "port": "6752",
  "debug": true
}
```

Debug 模式目前主要开启 GORM SQL 日志。应用日志统一写到标准输出。

## 3. 代码质量检查

### 3.1 格式化

检查已跟踪 Go 文件：

```bash
gofmt -l $(git ls-files '*.go')
```

格式化修改过的文件：

```bash
gofmt -w path/to/file.go path/to/file_test.go
```

提交前检查空白错误：

```bash
git diff --check
```

### 3.2 测试

准备好 `bin/dist` 后，可运行完整测试：

```bash
go test ./...
go test -race ./...
```

如果没有前端构建产物，可跳过 `bin` 包，验证所有后端包：

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

竞态检测使用相同包列表并增加 `-race`：

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

测试覆盖以所处分支为准；如果没有对应的 `_test.go`，上述命令只提供编译检查。新增或修改业务逻辑时应同步增加测试，尤其是房间并发、WebSocket 生命周期、数据库查询和存储失败路径。

### 3.3 静态检查

有前端资源：

```bash
go vet ./...
```

无前端资源：

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

## 4. 单元测试建议

### 4.1 测试位置

- 包内测试：`package room`，适合验证未导出状态和并发细节。
- 外部测试：`package room_test`，适合验证公开 API 和兼容性。
- 测试文件与实现放在同一目录，命名为 `*_test.go`。

### 4.2 优先覆盖范围

| 模块 | 优先测试 |
| --- | --- |
| `serve/middleware` | 认证开启/关闭、JWT 过期、敏感信息日志处理。 |
| `room` | 并发 Join/Leave/Close、超时清理、错误密码、消息路由。 |
| `event` | 本地/远程分发、监听器增删、context 取消。 |
| `serve/socket` | 连接关闭、房间关闭、读写错误、消息格式错误。 |
| `data` | SQLite/MySQL 查询一致性、分页、tag、时间范围和软删除。 |
| `storage` | 本地文件、S3 key、对象不存在和部分失败。 |
| `serve/route` | HTTP 参数、鉴权、错误响应和代理分支。 |
| `task` | 重复启动/关闭、ticker 停止和 panic 恢复。 |

### 4.3 使用接口替身

项目已经为主要边界定义接口：

- `data.DataApi`
- `storage.StorageApi`
- `event.EventEmitter`
- `room.Room` / `room.RemoteRoom`
- `metric.Metric`

单元测试优先使用小型 fake 实现，不要为纯业务分支启动真实 S3 或 RPC 服务。数据库查询可以使用临时 SQLite：

```go
db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
```

涉及 goroutine 的测试：

- 使用 channel 或 `sync.WaitGroup` 等待确定事件。
- 避免依赖固定的长时间 `time.Sleep`。
- 同时执行 `go test -race`。
- 为阻塞操作设置 context timeout，防止测试永远等待。

## 5. HTTP 手工测试

启动服务后先检查默认无密码模式：

```bash
curl -i http://localhost:6752/api/v1/auth/status
```

创建房间：

```bash
curl -sS \
  -X POST \
  -H 'Content-Type: application/json' \
  -d '{"useSecret":false}' \
  'http://localhost:6752/api/v1/room/create?name=dev-room&group=test'
```

上传并查询日志：

```bash
curl -sS \
  -H 'Content-Type: application/json' \
  --data-binary '{"message":"development smoke test"}' \
  'http://localhost:6752/api/v1/jsonLog/upload?name=smoke.json&env=dev'

curl -sS \
  'http://localhost:6752/api/v1/log/list?page=1&size=10&env=dev'
```

启用密码后，先调用 `/api/v1/auth/verify` 获取 Token，再为受保护请求增加：

```text
Authorization: Bearer <token>
```

## 6. WebSocket 手工测试

### 6.1 命令行

创建房间并记录返回的 `data.address`，然后执行：

```bash
websocat \
  'ws://localhost:6752/api/v1/ws/room/join?address=<room-address>&name=developer&userId=dev-1'
```

发送：

```json
{"type":"ping","createdAt":1753344000000,"requestId":"smoke-1","content":{}}
```

预期收到相同 `requestId` 的 `pong`。

### 6.2 浏览器测试页

仓库提供：

```text
test/websocket_event_test/index.html
```

启动静态服务器：

```bash
python3 -m http.server 8081 --directory test/websocket_event_test
```

访问 `http://localhost:8081`。

测试时应使用创建房间接口实际返回的 `<local-id>.<machine-id>` 地址。页面中的初始示例值不一定是合法房间地址；自定义更新消息时，协议类型应使用 `updateRoomInfo`。

## 7. MySQL 集成验证

仓库提供 MySQL 8.0 和 Adminer 的 Compose 环境：

```bash
docker compose -f test/docker/docker-compose.yml up -d
docker compose -f test/docker/docker-compose.yml ps
```

等待 MySQL 健康启动后，将 `test/docker/config-mysql.json` 的内容复制到仓库根目录 `config.json`，或合并其中的 `databaseConfig`。

当前程序不支持 `-c` 参数，以下形式无效：

```text
./page-spy-api -c test/docker/config-mysql.json
```

启动应用后验证上传、列表和统计接口。Adminer 地址为：

```text
http://localhost:8080
```

停止环境：

```bash
docker compose -f test/docker/docker-compose.yml down
```

删除数据库卷会清空测试数据：

```bash
docker compose -f test/docker/docker-compose.yml down -v
```

## 8. S3 集成验证

准备一个隔离 bucket 或 S3 兼容测试实例，在 `config.json` 中设置 `storageConfig` 后验证：

1. 上传日志后，对象出现在 `<baseDir>/<logDir>/<fileId>`。
2. `/log/download` 返回与上传内容一致的数据。
3. `/log/delete` 删除对象及数据库记录。
4. 重启服务时能够按当前实现加载远程数据库快照。
5. 错误 endpoint、错误凭据和不存在对象返回可诊断错误。

不要对生产 bucket 运行删除测试。

## 9. 多实例验证

至少启动两个不同工作目录的实例，每个目录使用独立 `config.json`：

```text
node-a/config.json
node-b/config.json
```

两边的 `rpcAddress` 必须完全一致，`selfRpcAddress` 分别指向自身。HTTP 和 RPC 端口均不能冲突。

建议验证：

1. 在 A 节点创建房间，从 B 节点查询并加入。
2. 两个节点上的 WebSocket 客户端互发广播和单播消息。
3. 在 A 上传日志，从 B 查询和下载。
4. 停止一个节点，观察查询和 RPC 错误。
5. 恢复节点后重新执行完整流程。

多实例应使用共享 MySQL。不要让多个节点使用各自 SQLite 文件并向同一个 S3 对象写数据库快照。

## 10. 修改后的最小检查清单

提交前至少执行：

```text
[ ] gofmt 无输出
[ ] git diff --check 通过
[ ] 相关包 go test 通过
[ ] 并发改动 go test -race 通过
[ ] go vet 通过
[ ] HTTP/WS 协议改动完成手工验证
[ ] 配置或数据模型改动包含迁移与兼容说明
[ ] 没有提交 config.json 中的真实密码、JWT 密钥或 S3 凭据
```

## 11. 常见问题

### `pattern dist/*: no matching files found`

原因：`bin/main.go` 的 embed 模式没有匹配文件。

处理：构建 page-spy-web 并复制到 `bin/dist`，或为纯后端编译临时创建 `bin/dist/index.html`。

### 修改配置后没有生效

确认服务的当前工作目录。程序只读取该目录下固定名称的 `config.json`，不会解析 `-c`。

### MySQL 已启动但连接失败

- 等待容器完成初始化。
- 检查 DSN 中主机名：宿主机运行应用通常使用 `127.0.0.1`；同一 Compose 网络内使用服务名 `mysql`。
- DSN 必须包含 `parseTime=True`。

### 房间地址无效

地址必须是 `<local-id>.<machine-id>`，且只能包含一个用于分隔的点。应使用创建房间接口返回的地址，不要自行使用普通名称代替。

### 受保护接口不要求 Token

这是当前无密码模式的行为。设置 `authConfig.password` 或 `AUTH_PASSWORD` 后才会启用 Bearer Token 校验。
