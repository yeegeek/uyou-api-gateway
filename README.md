# uyou-api-gateway

**一个纯净、高效、生产级的 API 网关与微服务开发框架**

基于 Apache APISIX + Go + gRPC, 旨在提供一个开箱即用的微服务基础设施, 让开发者专注于业务逻辑, 而非底层架构。

---

## ✨ 核心特性

- **🚀 纯净的框架**: 移除了所有业务示例代码, 只保留核心框架, 让你从零开始构建自己的应用。
- **⚡️ 自动化服务创建**: 通过 `make new-service` 命令, 交互式生成完整的微服务脚手架, 包括:
  - 标准化的 Go 项目结构
  - gRPC 服务与 API 定义
  - 数据库选择 (PostgreSQL / MongoDB / None)
  - **独立的 Docker Compose 环境**, 实现一键启动开发
- **🔌 动态路由管理**: 基于 APISIX 的动态配置, 通过简单的 YAML 文件管理路由, 无需重启网关。
- **🔧 生产级基础设施**: 包含 APISIX, etcd, Redis, 提供高性能、高可用的网关服务。
- **📚 清晰的文档**: 提供从快速入门到生产部署的完整指南。

---

## 🏁 5 分钟快速开始

### 1. 启动基础设施

```bash
# 克隆项目
git clone https://github.com/yeegeek/uyou-api-gateway.git
cd uyou-api-gateway

# 启动 APISIX, etcd, Redis
make start
```

### 2. 创建你的第一个微服务

```bash
# 运行交互式生成器
make new-service

# --- 按照提示输入 ---
# 服务名称 (如 user, order, product) [user]: user
# Go 模块路径 [github.com/yeegeek/uyou-user-service]: 
# gRPC 端口 [50051]: 
# 选择数据库类型 [1/2/3]: 1
# 数据库名称 [userdb]: 
# 主表名称 [users]: 
# ... 确认生成
```

这将在 `services/user-service` 目录下创建一个全新的微服务。

### 3. 启动新服务进行开发

```bash
# 进入服务目录
cd services/user-service

# 启动服务及其依赖的数据库 (由生成器自动创建的 docker-compose.yml)
make run
```

### 4. 配置 API 路由

回到项目根目录, 创建一个路由配置文件:

```bash
# 编辑 apisix/config/routes/user-routes.yaml
vi apisix/config/routes/user-routes.yaml
```

粘贴以下内容:

```yaml
routes:
  - id: "user_register"
    uri: /api/v1/users/register
    plugins:
      grpc-transcode:
        proto_id: "user_service"
        service: "user.UserService"
        method: "Create"
    upstream:
      nodes:
        "user-service:50051": 1
      type: roundrobin
      scheme: grpc

stream_routes:
  - id: "user_service"
    server_addr: "0.0.0.0"
    server_port: 50051
    upstream:
      nodes:
        "user-service:50051": 1
      type: roundrobin
      scheme: grpc
```

### 5. 同步路由到 APISIX

```bash
# 在项目根目录运行
make update-routes
```

### 6. 测试 API

```bash
curl -i -X POST http://localhost:9080/api/v1/users/register \
  -H "Content-Type: application/json" \
  -d '{"name": "test"}'
```

你将看到来自 `user-service` 的响应!

---

## 📚 文档

- **[快速入门指南](./docs/QUICKSTART.md)**: 更详细的入门教程。
- **[服务开发指南](./docs/SERVICE-GUIDE.md)**: 如何开发和构建你的微服务。
- **[架构设计](./docs/ARCHITECTURE.md)**: 框架的核心设计理念。
- **[生产部署](./docs/DEPLOYMENT.md)**: 如何将你的应用部署到生产环境。

## 🤝 贡献

欢迎提交 PR 和 Issue, 共同完善这个框架。

## License

MIT
