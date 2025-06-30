# MySQL 测试环境

这个目录包含了用于测试 PageSpy API 与 MySQL 数据库连接的 Docker Compose 配置。

## 启动 MySQL 测试环境

```bash
# 进入 test 目录
cd test

# 启动 MySQL 和 Adminer
docker-compose -f docker-compose.yml up -d

# 查看服务状态
docker-compose -f docker-compose.yml ps
```

## 服务信息

- **MySQL 数据库**
  - 端口: 3306
  - 数据库名: pagespy
  - 用户名: pagespy
  - 密码: pagespy123
  - Root 密码: pagespy123

- **Adminer (数据库管理界面)**
  - 访问地址: http://localhost:8080
  - 服务器: mysql
  - 用户名: pagespy
  - 密码: pagespy123
  - 数据库: pagespy

## 测试 PageSpy API

使用提供的 MySQL 配置文件启动 PageSpy API：

```bash
# 回到项目根目录
cd ..

# 使用 MySQL 配置启动应用
./page-spy-api -c test/config-mysql.json
```

## MySQL 连接字符串格式

在配置文件中，MySQL 连接字符串的格式为：

```
用户名:密码@tcp(主机:端口)/数据库名?charset=utf8mb4&parseTime=True&loc=Local
```

示例：
```
pagespy:pagespy123@tcp(localhost:3306)/pagespy?charset=utf8mb4&parseTime=True&loc=Local
```

## 停止测试环境

```bash
# 停止服务
docker-compose -f docker-compose.yml down

# 停止服务并删除数据卷（注意：这会删除所有数据）
docker-compose -f docker-compose.yml down -v
```

## 故障排除

1. **端口冲突**: 如果 3306 或 8080 端口被占用，可以修改 docker-compose.yml 中的端口映射
2. **连接失败**: 确保 MySQL 容器完全启动后再尝试连接（通常需要等待 10-30 秒）
3. **数据持久化**: MySQL 数据存储在 Docker 卷中，重启容器不会丢失数据
