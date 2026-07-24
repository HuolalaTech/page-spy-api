# Page Spy API 代码审查报告

## 1. 审查信息

- 审查日期：2026-07-24
- 审查版本：`master@8e8b087`
- 审查范围：Bug、安全、性能、并发、数据一致性、集群行为、构建与测试
- 审查方式：静态代码审查、编译检查、`go vet`、race 构建尝试、`govulncheck`
- 代码修改：初始审查仅记录问题；第 8 节记录了随后实施的兼容性修复

## 2. 总体结论

当前版本不建议直接暴露到公网，也不建议在未配置共享 MySQL 的情况下使用多节点部署。

最优先处理的问题是：

1. 默认配置下管理接口匿名开放。
2. 内部 RPC 监听所有网卡且没有认证或 TLS。
3. 公开上传、房间创建和 WebSocket 消息没有大小与速率限制。
4. 房间密码通过 URL 传递并被完整记录到访问日志。
5. WebSocket 生命周期存在监听器、连接和 goroutine 泄漏。
6. 多节点批量删除会形成反向代理循环。
7. S3、SQLite 和多节点组合存在数据库快照互相覆盖风险。

## 3. 审查发现

### CR-001：默认配置下管理接口匿名开放

- 严重度：P0 / 严重
- 分类：安全、配置
- 证据：
  - [serve/middleware/auth.go:16](../serve/middleware/auth.go#L16) 在未设置密码时直接跳过认证。
  - [config/defaultConfig.json:1](../config/defaultConfig.json#L1) 默认配置只包含端口。
  - [serve/run.go:29](../serve/run.go#L29) HTTP 服务监听所有网卡。
  - [serve/route/route.go:252](../serve/route/route.go#L252) 默认允许删除日志。
- 影响：网络可达的匿名用户可以列举、下载和删除所有日志。
- 建议：认证默认 fail-closed；无认证模式必须显式启用；默认禁止删除日志。
- 兼容性：修复安全默认值会改变现有部署行为，需要迁移和发布说明。

### CR-002：内部 RPC 对外监听且没有认证

- 严重度：P0 / 严重
- 分类：安全、集群
- 证据：
  - [rpc/rpc.go:53](../rpc/rpc.go#L53) 使用 `http.ListenAndServe(":"+port)`。
  - [room/rpc_local_manager.go:68](../room/rpc_local_manager.go#L68) 暴露房间查询、创建、加入和删除能力。
  - [event/rpc_event.go:45](../event/rpc_event.go#L45) 暴露消息注入能力。
- 影响：攻击者可以调用内部控制面接口、获取房间内部信息、操纵房间或注入消息。
- 建议：绑定明确的内网地址；使用 mTLS、请求签名或服务网格认证；限制允许调用的来源。
- 兼容性：增加认证或 TLS 会影响集群部署配置，需要迁移方案。

### CR-003：公开请求没有资源上限

- 严重度：P0 / 严重
- 分类：安全、性能、可用性
- 证据：
  - [serve/route/route.go:293](../serve/route/route.go#L293) 三个公开上传接口使用无上限的 `io.ReadAll`。
  - [serve/socket/socket.go:308](../serve/socket/socket.go#L308) 房间创建请求体使用无上限的 `io.ReadAll`。
  - [serve/socket/socket.go:96](../serve/socket/socket.go#L96) WebSocket 没有 `SetReadLimit`。
  - [serve/run.go:29](../serve/run.go#L29) HTTP 服务未设置 read、write、header 和 idle timeout。
- 影响：可造成内存、临时磁盘、日志磁盘和 goroutine 耗尽。
- 建议：增加请求体上限、单文件上限、并发限制、速率限制和 HTTP/WebSocket deadline。
- 兼容性：会拒绝此前能够提交的超大或超慢请求，需要先确定正式限制。

### CR-004：房间密码泄露到访问日志

- 严重度：P1 / 高
- 分类：安全、日志
- 证据：
  - [serve/socket/socket.go:352](../serve/socket/socket.go#L352) 从 URL 查询参数读取房间密码。
  - [serve/socket/socket.go:412](../serve/socket/socket.go#L412) 密码检查接口同样从 URL 读取密码。
  - [serve/middleware/logger.go:38](../serve/middleware/logger.go#L38) 记录完整 `RequestURI`。
- 影响：密码可能出现在应用日志、反向代理日志、APM 和集中日志平台。
- 建议：短期先对日志中的 `secret` 做脱敏；长期改为请求体、受保护 Header 或 WebSocket 首帧传递。
- 兼容性：日志脱敏不影响功能；修改密码传输方式会影响客户端协议。

### CR-005：认证秘密和数据库密码以明文保存或输出

- 严重度：P1 / 高
- 分类：安全、秘密管理
- 证据：
  - [config/load.go:43](../config/load.go#L43) 环境变量中的密码和 JWT 密钥会写入配置结构。
  - [config/load.go:75](../config/load.go#L75) 随后将配置保存到磁盘。
  - [config/config.go:73](../config/config.go#L73) 配置文件权限为 `0644`。
  - [data/db.go:60](../data/db.go#L60) 完整记录 MySQL DSN。
- 影响：本机其他用户、备份系统或日志平台可能获得认证密码、JWT 密钥和数据库密码。
- 建议：立即对 DSN 日志脱敏并将配置权限改为 `0600`；后续迁移到密码哈希和专用 secret provider。
- 兼容性：日志脱敏和收紧文件权限不影响功能；密码哈希格式需要迁移。

### CR-006：WebSocket 加入失败时泄漏监听器

- 严重度：P1 / 高
- 分类：Bug、性能、可用性
- 证据：
  - [room/rpc_remote_manager.go:242](../room/rpc_remote_manager.go#L242) 先启动远程房间并注册监听器。
  - [room/rpc_remote_manager.go:247](../room/rpc_remote_manager.go#L247) 随后才执行房间加入和密码校验。
  - 加入失败路径没有调用 `Close` 或移除监听器。
- 影响：反复提交错误密码可持续增长监听器和内存占用。
- 建议：先完成加入验证再注册监听器，或在失败路径中 `defer` 清理。
- 兼容性：不改变正常加入流程和对外协议，可直接修复。

### CR-007：context 无法中断阻塞的 WebSocket 读取

- 严重度：P1 / 高
- 分类：Bug、并发、可用性
- 证据：
  - [serve/socket/socket.go:108](../serve/socket/socket.go#L108) 使用阻塞的 `ReadJSON`。
  - [serve/socket/socket.go:244](../serve/socket/socket.go#L244) 进入读取后，`cancelCtx.Done()` 和 `room.Done()` 无法被重新检查。
- 影响：房间关闭后，连接和 goroutine 仍可能一直存活，直到客户端主动发送数据或断开。
- 建议：使用独立 read-pump；context 取消时主动关闭连接；设置 read deadline。
- 兼容性：不会改变正常消息协议，只会正确关闭已经结束的会话，可直接修复。

### CR-008：多节点批量删除形成代理循环

- 严重度：P1 / 高
- 分类：Bug、集群、可用性
- 证据：
  - [serve/route/route.go:257](../serve/route/route.go#L257) 逐个处理文件 ID。
  - [serve/route/route.go:263](../serve/route/route.go#L263) 遇到非本机文件时代理包含所有 ID 的原始请求。
- 影响：包含不同节点文件的请求会在节点之间循环代理，同时可能只完成部分删除。
- 建议：按 machine ID 对文件分组，每个节点只接收本节点的 ID，并汇总逐项结果。
- 兼容性：可以保持原路由、参数和响应结构，属于接口兼容的错误修复。

### CR-009：多节点分页结果不符合全局分页语义

- 严重度：P1 / 高
- 分类：Bug、集群、性能
- 证据：
  - [serve/route/core.go:186](../serve/route/core.go#L186) 每个节点分别执行同一个 page/size 查询后直接合并。
  - [rpc/util.go:18](../rpc/util.go#L18) RPC 请求逐节点串行执行。
- 影响：
  - 第一页最多返回 `节点数 × size` 条。
  - 后续页会遗漏全局排序中的数据。
  - 任一节点故障会导致整个查询失败。
  - 延迟随节点数量线性增长。
- 建议：并行获取候选集，完成全局排序、去重后再分页；定义部分节点失败策略。
- 兼容性：不改变接口形式，但会修正当前可观察到的错误结果，发布前应验证客户端是否依赖错误行为。

### CR-010：S3、多节点和 SQLite 组合可能丢失数据

- 严重度：P1 / 高
- 分类：Bug、数据一致性、集群
- 证据：
  - [data/db.go:86](../data/db.go#L86) 远程存储模式仍使用每个实例自己的数据库。
  - [data/db.go:170](../data/db.go#L170) 定时读取整个本地 SQLite 文件并上传到固定对象。
- 影响：
  - 多个实例会互相覆盖数据库快照，最后写入者覆盖其他实例的数据。
  - 直接复制活动 SQLite 主文件不保证包含 WAL 中尚未 checkpoint 的数据。
- 建议：多节点强制使用共享数据库；SQLite 备份使用 SQLite backup API 或一致性快照。
- 兼容性：增加部署校验可能让不安全的旧配置无法启动，需要迁移说明。

### CR-011：日志文件和数据库元数据不是原子更新

- 严重度：P1 / 高
- 分类：Bug、数据一致性
- 证据：
  - [serve/route/core.go:63](../serve/route/core.go#L63) 上传时先保存文件，再创建数据库记录。
  - [serve/route/core.go:238](../serve/route/core.go#L238) 删除时先删除文件，再删除数据库记录。
- 影响：
  - 数据库写入失败会留下孤儿文件。
  - 数据库删除失败会留下指向不存在文件的记录。
  - 并发日志组更新可能发生丢失更新。
- 建议：引入 `pending/saved/deleting` 状态、补偿任务或 outbox；数据库更新使用事务和唯一约束。
- 兼容性：可以保持对外 API 不变，但需要数据迁移和故障恢复测试。

### CR-012：查询不存在的日志组会触发 panic

- 严重度：P2 / 中
- 分类：Bug、稳定性
- 证据：[serve/route/core.go:198](../serve/route/core.go#L198) 未检查 `FindLogGroup` 返回的 `nil` 就访问 `logGroup.Logs`。
- 影响：请求不存在的 `groupId` 会中断当前 HTTP 请求。
- 建议：返回现有错误响应格式的 not-found 错误。
- 兼容性：只改变原本 panic 的异常路径，可直接修复。

### CR-013：MySQL 模式使用 SQLite 专属函数

- 严重度：P2 / 中
- 分类：Bug、数据库兼容
- 证据：[data/db.go:361](../data/db.go#L361) `CountLogsGroup` 固定使用 `strftime`。
- 影响：MySQL 部署中的 `/log/count` 查询失败。
- 建议：按 GORM dialect 选择 SQLite `strftime` 或 MySQL `DATE_FORMAT`。
- 兼容性：SQLite 行为不变，MySQL 从失败变为可用，可直接修复。

### CR-014：任务关闭必然 panic

- 严重度：P2 / 中
- 分类：Bug、生命周期
- 证据：
  - [task/task.go:29](../task/task.go#L29) `Close` 对 `done` 执行 `close`。
  - [task/task.go:49](../task/task.go#L49) `NewTask` 没有初始化 `done`。
- 影响：调用 `Task.Close` 或 `TaskManager.Close` 时发生 `panic: close of nil channel`。
- 建议：初始化 `done`，使用 `sync.Once` 保证幂等关闭，并停止 ticker。
- 兼容性：不改变任务正常执行行为，可直接修复。

### CR-015：房间关闭和共享状态存在数据竞争

- 严重度：P2 / 中
- 分类：Bug、并发
- 证据：
  - [room/basic_room.go:23](../room/basic_room.go#L23) “检查状态后关闭 channel”不是原子操作。
  - [room/local_room.go:55](../room/local_room.go#L55) 返回未复制的连接 slice。
  - [room/local_room.go:160](../room/local_room.go#L160) `ActiveAt` 等状态在多个 goroutine 间无统一同步。
  - [room/remote_room.go:71](../room/remote_room.go#L71) 单播消息只检查 `To`，没有检查 `To.Address`。
  - [rpc/address.go:155](../rpc/address.go#L155) 后续会直接解引用该地址。
- 影响：并发关闭可能二次关闭 channel；房间列表、用户列表和活跃时间可能发生 race；畸形单播消息可以触发 panic。
- 建议：使用 `sync.Once`；锁内更新状态；返回 slice/map 快照。
- 兼容性：不改变正常业务语义，可直接修复。

### CR-016：事件监听器去重判断无效

- 严重度：P2 / 中
- 分类：Bug、性能
- 证据：[event/local_event.go:34](../event/local_event.go#L34) 比较的是两个临时接口变量的地址，而不是监听器本身。
- 影响：同一监听器可以被重复注册，导致重复消息和额外内存占用。
- 建议：直接比较可比较的接口值，或使用稳定 listener ID。
- 兼容性：正常场景不应依赖重复回调，可直接修复。

### CR-017：API 泄露内部错误并错误分类状态码

- 严重度：P2 / 中
- 分类：安全、API
- 证据：
  - [serve/common/api.go:24](../serve/common/api.go#L24) 将任意 `err.Error()` 返回客户端。
  - [serve/middleware/error.go:17](../serve/middleware/error.go#L17) 除少数房间错误外都返回 HTTP 400。
- 影响：可能泄露数据库、文件路径、S3 和内部拓扑信息；内部故障被误报为客户端错误。
- 建议：内部记录详细错误，对外返回稳定错误码和通用消息。
- 兼容性：会改变客户端可见的错误文本和部分 HTTP 状态码，需要 API 兼容评估。

### CR-018：远程存储不执行日志保留和容量清理

- 严重度：P2 / 中
- 分类：Bug、成本、运维
- 证据：[serve/route/core.go:332](../serve/route/core.go#L332) 只在本地存储模式注册清理任务。
- 影响：S3 日志不会应用 `MaxLogLifeTimeOfHour` 或容量配置，存储成本持续增长。
- 建议：为远程存储增加保留策略，优先使用 S3 lifecycle。
- 兼容性：会删除此前永久保留的对象，需要显式启用或迁移说明。

### CR-019：S3 客户端重复创建

- 严重度：P3 / 低
- 分类：性能
- 证据：[storage/s3.go:25](../storage/s3.go#L25) 每次操作都创建新的 session 和 client。
- 影响：增加对象存储请求延迟、分配和连接管理开销。
- 建议：在 `RemoteApi` 初始化时创建并复用 `s3.S3` 客户端。
- 兼容性：不改变请求和存储语义，可直接修复。

### CR-020：集群查询和数据库读取存在额外开销

- 严重度：P3 / 低
- 分类：性能
- 证据：
  - [data/db.go:327](../data/db.go#L327) 日志组分页会预加载组内全部日志。
  - [data/db.go:177](../data/db.go#L177) 数据库同步将整个 SQLite 文件读入内存。
  - [serve/middleware/logger.go:29](../serve/middleware/logger.go#L29) 每个请求遍历全部路由。
- 影响：大日志组、大数据库或高 QPS 下产生额外内存和 CPU 消耗。
- 建议：列表查询只返回摘要；数据库备份采用流式复制；直接使用 `c.Path()` 作为路由标签。
- 兼容性：路由标签和内部实现可直接优化；移除日志组响应中的 `Logs` 字段内容需要兼容评估。

### CR-021：完整构建失败且缺少测试

- 严重度：P2 / 中
- 分类：构建、质量
- 证据：
  - [bin/main.go:12](../bin/main.go#L12) 要求嵌入 `bin/dist/*`，但仓库中不存在该目录。
  - 仓库没有 Go `_test.go` 文件。
- 影响：
  - `go test ./...`、完整 `go vet ./...` 和 race 构建失败。
  - 并发、鉴权、上传、集群代理和数据库兼容没有回归保护。
- 建议：提供可独立构建的后端入口或明确生成静态资源的构建步骤；补充单元和集成测试。
- 兼容性：新增构建入口和测试不会影响现有功能。

### CR-022：依赖存在已知可达漏洞

- 严重度：P1 / 高
- 分类：供应链、安全
- `govulncheck` 结果：
  - `golang.org/x/text v0.26.0`：GO-2026-5970，无效输入导致无限循环，修复于 `v0.39.0`。
  - `golang.org/x/net v0.41.0`：GO-2026-5026，IDNA/Punycode 校验问题，修复于 `v0.55.0`。
  - `github.com/golang-jwt/jwt v3.2.2+incompatible`：GO-2025-3553，Header 解析过量内存分配。
  - `github.com/imroc/req/v2 v2.1.0`：GO-2024-3098，畸形 URL 可能产生非预期请求。
- 建议：升级直接和间接依赖，并在 CI 中增加 `govulncheck`。
- 兼容性：依赖升级通常不改变项目 API，但不能保证运行行为完全不变，需要回归测试。

## 4. 不产生功能不兼容改动的修复清单

### 4.1 兼容性判定标准

本节“可直接修复”表示：

- 不修改 HTTP 路由、方法和参数名称。
- 不修改 WebSocket 和 RPC 正常消息格式。
- 不修改成功响应字段和正常请求结果。
- 不修改认证开关、部署要求、文件 ID 和存储路径。
- 只修复 panic、泄漏、数据竞争、日志泄密或内部性能问题。

### 4.2 可直接修复

| 顺序 | 修复项 | 对应问题 | 兼容性说明 | 建议验证 |
| --- | --- | --- | --- | --- |
| 1 | 访问日志对 `secret`、`token` 等查询参数脱敏 | CR-004 | 只改变日志内容 | 日志单测 |
| 2 | MySQL DSN 日志脱敏 | CR-005 | 不改变数据库连接 | 启动日志测试 |
| 3 | 配置文件保存权限改为 `0600` | CR-005 | 不改变配置格式和读取行为 | 文件权限测试 |
| 4 | 加入房间失败时关闭 remote room 并移除监听器 | CR-006 | 成功路径与协议不变 | 错误密码压力测试 |
| 5 | context 取消时主动关闭 WebSocket 连接 | CR-007 | 只关闭已经结束的会话 | 房间关闭集成测试 |
| 6 | 按 machine ID 分组处理批量删除 | CR-008 | 路由和参数不变，修复代理循环 | 双节点混合删除测试 |
| 7 | `ListFilesInGroup` 增加 nil 检查 | CR-012 | 异常路径从 panic 变为现有格式错误响应 | 不存在 group 测试 |
| 8 | 按数据库 dialect 生成月份聚合表达式 | CR-013 | SQLite 结果不变，MySQL 从失败变为可用 | SQLite/MySQL 对照测试 |
| 9 | 初始化 `Task.done`、幂等关闭并停止 ticker | CR-014 | 任务正常运行行为不变 | 生命周期单测 |
| 10 | `basicRoom.close` 使用 `sync.Once` | CR-015 | 正常关闭消息不变 | 并发关闭 race 测试 |
| 11 | 对房间 `Info`、连接列表和时间字段补齐锁与快照复制 | CR-015 | 返回结构和内容不变 | `go test -race` |
| 12 | 修复监听器去重比较并在读取时复制 slice | CR-016 | 消除非预期重复回调 | 重复注册单测 |
| 13 | 校验单播消息的 `To.Address` 和目标 machine 是否存在 | CR-015 | 只影响原本会 panic 的畸形消息 | WebSocket 畸形消息测试 |
| 14 | JWT 密钥初始化改为 `sync.Once` 或构造时注入 | CR-015 | Token 格式和密钥不变 | 并发鉴权 race 测试 |
| 15 | 初始化并复用 S3 session/client | CR-019 | 对象 key、ACL 和请求语义不变 | S3 集成测试 |
| 16 | SQLite 备份从 `os.ReadFile` 改为流式、一致性备份 | CR-010、CR-020 | 单节点备份对象语义不变 | 备份恢复测试 |
| 17 | 请求日志直接使用 Echo 已匹配的路由路径，取消逐路由遍历 | CR-020 | 仅内部性能优化 | metrics 标签回归 |
| 18 | 增加后端包、任务、房间、鉴权、代理和数据库测试 | CR-021 | 测试代码不改变运行行为 | CI |
| 19 | 增加一个不依赖静态资源的可选后端构建入口 | CR-021 | 新增能力，不修改原入口 | 双入口构建测试 |

### 4.3 接口形式兼容，但需要回归验证

以下修复不改变接口字段或调用方式，但会修正当前可观察到的错误结果，或升级第三方实现，因此不列入“零行为变化”：

| 修复项 | 对应问题 | 可能变化 |
| --- | --- | --- |
| 全局排序后再执行集群分页 | CR-009 | 返回条数和后续页内容会从错误结果变为正确结果 |
| 并行执行跨节点查询 | CR-009 | 节点请求顺序和失败时序变化 |
| 文件与元数据增加状态机和补偿任务 | CR-011 | 故障场景下的可见状态变化 |
| 升级 `x/net`、`x/text`、Echo/JWT 和 `req` | CR-022 | 第三方库对畸形输入、重定向和协议细节的处理变化 |
| 文件写入使用临时文件和原子 rename | CR-011 | 失败路径和并发重复上传行为改善 |

## 5. 明确不属于无兼容风险的修复

以下项目需要单独设计、灰度和迁移，不能作为“不会产生不兼容改动”的修复直接发布：

1. 默认强制开启认证或默认禁止删除。
2. 为 RPC 增加认证、TLS 或修改监听地址。
3. 把房间密码从 URL 改到 Header、请求体或 WebSocket 首帧。
4. 给上传、WebSocket、HTTP 请求增加硬大小或时间上限。
5. 将明文密码配置迁移为哈希。
6. 修改 CORS 或 WebSocket Origin 默认策略。
7. 将文件 ID 从 MD5 改为 SHA-256 或随机 ID。
8. 强制 S3 多节点部署必须使用 MySQL。
9. 为已有 S3 对象启用自动保留和删除策略。
10. 修改公开错误文本和 HTTP 状态码。
11. 从日志组列表响应中移除完整 `Logs` 数据。

## 6. 建议实施顺序

### 第一批：可直接发布的内部修复

1. 日志和 DSN 脱敏。
2. 修复任务关闭 panic、房间并发关闭和监听器去重。
3. 修复加入失败监听器泄漏及 WebSocket 取消。
4. 修复 nil 解引用和畸形单播消息 panic。
5. 修复 MySQL 聚合函数。
6. 修复多节点批量删除代理循环。
7. 复用 S3 客户端。

### 第二批：补充测试后发布

1. 全局分页和并行 RPC。
2. 文件/数据库一致性状态机。
3. 依赖升级。
4. SQLite 一致性备份。

### 第三批：需要迁移方案的安全改造

1. 默认认证策略。
2. RPC 认证与 TLS。
3. 上传、WebSocket 和 HTTP 资源限制。
4. 密码存储格式及秘密传输协议。
5. 多节点数据库部署约束。
6. S3 生命周期策略。

## 7. 验证结果

- 除 `bin` 入口外，其余包能够编译并通过 `go vet`。
- `go test ./...`、完整 `go vet ./...` 和 race 构建因为缺少 `bin/dist/*` 失败。
- 初始审查时仓库没有 Go `_test.go` 文件，因此当时的 race 检查没有实际覆盖并发路径。
- `govulncheck` 检出 4 个当前调用路径可达的依赖漏洞。
- 以上结果是开始修复前的基线，最新修复验证结果见下一节。

## 8. 兼容修复实施状态

2026-07-24 已开始实施不改变现有对外协议的修复：

| 状态 | 修复内容 | 对应问题 |
| --- | --- | --- |
| 已完成 | 访问日志敏感查询参数脱敏 | CR-004 |
| 已完成 | MySQL DSN 日志脱敏、配置保存失败显式记录 | CR-005 |
| 已完成 | 密码使用恒定时间比较，JWT 密钥初始化和读取增加并发保护 | CR-015 |
| 已完成 | 加入失败时清理 remote room 和监听器 | CR-006 |
| 已完成 | 房间结束或 context 取消时中断 WebSocket 阻塞读取 | CR-007 |
| 已完成 | 日志组不存在时返回错误，避免 nil panic | CR-012 |
| 已完成 | SQLite/MySQL 月份聚合方言适配 | CR-013 |
| 已完成 | Task 初始化、幂等启动/关闭、ticker 停止和无效任务校验 | CR-014 |
| 已完成 | 房间幂等关闭、状态锁、快照复制和畸形地址防护 | CR-015 |
| 已完成 | 监听器正确去重、读取快照、空集合清理和无效事件校验 | CR-016 |
| 已完成 | S3 session/client 初始化后复用 | CR-019 |
| 已完成 | 增加不依赖前端静态资源的 `cmd/backend` 入口 | CR-021 |
| 已完成 | 增加数据库、事件、房间、鉴权、路由、存储和任务测试 | CR-021 |
| 待后续处理 | 配置文件权限、跨节点批量删除、一致性 SQLite 备份和路由标签优化 | CR-005、CR-008、CR-010、CR-020 |
| 未实施 | 会改变认证、RPC、资源限制、分页、错误响应或部署/存储策略的修复 | CR-001、CR-002、CR-003、CR-009、CR-010（多节点约束）、CR-011、CR-017、CR-018、CR-022 |

本批修复验证结果：

- 兼容范围内全部包 `go test` 通过。
- 兼容范围内全部包 `go test -race` 通过。
- 兼容范围内全部包 `go vet` 通过。
- `go build -o /tmp/page-spy-api-backend ./cmd/backend` 通过。
- `git diff --check` 通过。
- 原 `bin` 入口仍依赖未提交的 `bin/dist/*`；新增的 `cmd/backend` 可以独立构建。
