# 快速入门指南

本指南将带你从零开始, 使用 uyou-api-gateway 框架构建、运行和测试一个完整的微服务。

## 目录

1. [环境准备](#1-环境准备)
2. [启动开发环境](#2-启动开发环境)
3. [创建第一个微服务](#3-创建第一个微服务)
4. [理解生成的代码](#4-理解生成的代码)
5. [本地开发与测试](#5-本地开发与测试)
6. [配置 API 路由](#6-配置-api-路由)
7. [通过网关访问服务](#7-通过网关访问服务)

---

## 1. 环境准备

在开始之前, 请确保你的开发环境中安装了以下工具:

- **Docker**: 用于运行容器化服务。
- **Docker Compose**: 用于编排多容器应用。
- **Go**: 1.21 或更高版本。
- **Git**: 用于版本控制。
- **protoc**: Protobuf 编译器。
- **protoc-gen-go** & **protoc-gen-go-grpc**: Go 的 Protobuf 插件。

## 2. 启动开发环境

首先, 克隆 uyou-api-gateway 项目并启动完整的开发环境。

```bash
# 克隆项目
git clone https://github.com/yeegeek/uyou-api-gateway.git
cd uyou-api-gateway

# 启动开发环境 (APISIX, etcd, Redis, PostgreSQL, MongoDB)
make start dev
```

`make start dev` 命令会使用 `docker-compose.dev.yml` 文件启动所有开发所需的服务。你可以通过 `make status dev` 查看它们的状态。

## 3. 创建第一个微服务

框架的核心优势在于能够快速生成标准化的微服务。让我们创建一个名为 `user` 的服务。

```bash
# 在项目根目录运行
make new-service
```

生成器会以交互方式询问你一些配置信息:

- **服务名称**: 输入 `user`。
- **Go 模块路径**: 接受默认值 `github.com/yeegeek/uyou-user-service`。
- **gRPC 端口**: 接受默认值 `50051`。
- **数据库类型**: 选择 `1` (PostgreSQL)。

确认后, 生成器会在 `services/user` 目录下创建所有必要的文件, **并自动将该服务添加到 `docker-compose.dev.yml`**。

## 4. 理解生成的代码

进入新创建的服务目录 `cd services/user`, 你会看到以下结构:

```
.
├── api/proto/user.proto   # API 定义
├── cmd/server/main.go     # 服务入口
├── config/config.yaml     # 服务配置
├── deployments/           # 部署相关文件
├── internal/              # 内部代码
├── docker-compose.yml     # 生产环境部署配置
├── Dockerfile             # 生产级 Dockerfile
├── go.mod                 # Go 模块
├── Makefile               # 常用命令
└── README.md              # 服务专属文档
```

- **`docker-compose.yml`**: 这是服务专属的 **生产环境** 部署文件, 包含了服务本身及其依赖。
- **`Dockerfile`**: 多阶段构建的生产级 Dockerfile。
- **`api/proto/user.proto`**: gRPC 服务定义的地方。

## 5. 本地开发与测试

### 5.1 重启开发环境

由于 `docker-compose.dev.yml` 已经被 `make new-service` 修改, 我们需要重启开发环境来加载新的 `user` 服务。

```bash
# 在项目根目录运行
make restart dev
```

现在, `user` 服务已经作为容器在你的本地环境中运行了。

### 5.2 查看日志

你可以随时查看所有开发服务的日志:

```bash
# 在项目根目录运行
make logs dev

# 或者只看 user 服务的日志
docker compose -f docker-compose.dev.yml logs -f user
```

## 6. 配置 API 路由

服务已经运行, 但 API 网关 (APISIX) 还不知道如何将外部请求转发给它。我们需要配置路由。

在项目根目录, 创建一个新的路由配置文件 `apisix/config/routes/user-routes.yaml`:

```yaml
routes:
  - id: "user_create"
    uri: /api/v1/users
    methods:
      - POST
    plugins:
      grpc-transcode:
        proto_id: "user_service"
        service: "user.UserService"
        method: "Create"
    upstream:
      nodes:
        "user:50051": 1 # 服务名:端口 (与 docker-compose.dev.yml 中的服务名一致)
      type: roundrobin
      scheme: grpc

stream_routes:
  - id: "user_service"
    server_addr: "0.0.0.0"
    server_port: 50051
    upstream:
      nodes:
        "user:50051": 1
      type: roundrobin
      scheme: grpc
```

### 同步路由

运行以下命令将新配置应用到 APISIX:

```bash
# 在项目根目录运行
make update-routes
```

## 7. 通过网关访问服务

现在, 所有环节都已打通。让我们通过 APISIX 网关来测试 `user-service` 的 `Create` 方法。

```bash
curl -i -X POST http://localhost:9080/api/v1/users \
  -H "Content-Type: application/json" \
  -d '{"name": "test-user"}'
```

**预期响应**:

```http
HTTP/1.1 200 OK
...

{"id":"12345","message":"User created successfully"}
```

**恭喜!** 你已经成功创建、运行、配置并测试了你的第一个微服务。

接下来, 你可以:
- 在 `services/user/api/proto/user.proto` 中添加更多 RPC 方法。
- 在 `services/user/internal/handler` 目录中实现这些方法的逻辑。
- 在 `apisix/config/routes/user-routes.yaml` 中添加更多路由规则。
- 每次修改代码后, `docker-compose.dev.yml` 会自动重新构建并启动你的服务。
