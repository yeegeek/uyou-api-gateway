# 快速入门指南

本指南将带你从零开始, 使用 uyou-api-gateway 框架构建、运行和测试一个完整的微服务。

## 目录

1. [环境准备](#1-环境准备)
2. [启动核心基础设施](#2-启动核心基础设施)
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

## 2. 启动核心基础设施

首先, 克隆 uyou-api-gateway 项目并启动其核心组件。

```bash
# 克隆项目
git clone https://github.com/yeegeek/uyou-api-gateway.git
cd uyou-api-gateway

# 启动 APISIX, etcd, Redis
make start
```

`make start` 命令会启动 `docker-compose.yml` 文件中定义的所有服务。你可以通过 `make status` 或 `docker compose ps` 查看它们的状态。

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
- **数据库名称**: 接受默认值 `userdb`。
- **主表名称**: 接受默认值 `users`。

确认后, 生成器会在 `services/user-service` 目录下创建所有必要的文件。

## 4. 理解生成的代码

进入新创建的服务目录 `cd services/user-service`, 你会看到以下结构:

```
.
├── api/proto/user.proto   # API 定义
├── cmd/server/main.go     # 服务入口
├── config/config.yaml     # 服务配置
├── internal/              # 内部代码
├── docker-compose.yml     # 本地开发环境
├── Dockerfile             # 容器镜像定义
├── go.mod                 # Go 模块
├── Makefile               # 常用命令
└── README.md              # 服务专属文档
```

- **`docker-compose.yml`**: 这是服务专属的开发环境, 只包含 `user-service` 和它依赖的 `postgres` 数据库。
- **`api/proto/user.proto`**: gRPC 服务定义的地方。
- **`Makefile`**: 提供了 `proto`, `build`, `run` 等便捷命令。

## 5. 本地开发与测试

### 5.1 生成 gRPC 代码

`user.proto` 文件定义了 API, 但我们需要生成 Go 代码才能使用它。

```bash
# 在 services/user-service 目录运行
make proto
```

此命令会根据 `.proto` 文件生成 `*.pb.go` 和 `*_grpc.pb.go` 文件。

### 5.2 启动服务

现在, 启动 `user-service` 和它的数据库。

```bash
# 在 services/user-service 目录运行
make run
```

`make run` 会:
1. 使用 `docker-compose up -d` 启动 PostgreSQL 数据库。
2. 使用 `go run` 编译并运行你的 Go 服务代码。

服务将在端口 `50051` 上监听 gRPC 请求。

## 6. 配置 API 路由

服务已经运行, 但 API 网关 (APISIX) 还不知道如何将外部请求转发给它。我们需要配置路由。

回到项目根目录 `cd ../../`。

创建一个新的路由配置文件 `apisix/config/routes/user-routes.yaml`:

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
        "host.docker.internal:50051": 1
      type: roundrobin
      scheme: grpc

stream_routes:
  - id: "user_service"
    server_addr: "0.0.0.0"
    server_port: 50051
    upstream:
      nodes:
        "host.docker.internal:50051": 1
      type: roundrobin
      scheme: grpc
```

**注意**: 我们使用 `host.docker.internal` 来让 APISIX 容器访问在主机上运行的 `user-service`。这是为了方便本地开发。

### 同步路由

运行以下命令将新配置应用到 APISIX:

```bash
# 在项目根目录运行
make update-routes
```

## 7. 通过网关访问服务

现在, 所有环节都已打通。让我们通过 APISIX 网关来测试 `user-service` 的 `Create` 方法。

```bash
curl -i -X POST http://localhost:9080/api/v1/users/register \
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
- 在 `user.proto` 中添加更多 RPC 方法。
- 在 `internal/handler` 目录中实现这些方法的逻辑。
- 在 `user-routes.yaml` 中添加更多路由规则。
