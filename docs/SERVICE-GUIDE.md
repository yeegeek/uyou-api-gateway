# 服务开发指南

本指南详细介绍了在 uyou-api-gateway 框架内从零开始开发一个完整微服务的全过程。

## 目录

1. [核心理念：服务即单元](#1-核心理念服务即单元)
2. [第一步：定义 API (Protobuf)](#2-第一步定义-api-protobuf)
3. [第二步：实现业务逻辑](#3-第二步实现业务逻辑)
    - [项目结构概览](#项目结构概览)
    - [Handler (处理器)](#handler-处理器)
    - [Service (服务层)](#service-服务层)
    - [Repository (数据访问层)](#repository-数据访问层)
4. [第三步：配置服务](#4-第三步配置服务)
5. [第四步：本地运行与调试](#5-第四步本地运行与调试)
6. [第五步：集成到 API 网关](#6-第五步集成到-api-网关)
7. [第六步：构建与部署](#7-第六步构建与部署)

---

## 1. 核心理念：服务即单元

在 uyou-api-gateway 框架中, 每个微服务都是一个独立的、自包含的开发单元。这意味着:

- **独立的环境**: 每个服务都有自己的 `docker-compose.yml`, 用于定义其依赖的数据库、缓存或其他服务。
- **独立的构建**: 每个服务都有自己的 `Makefile` 和 `Dockerfile`, 可以独立构建和打包。
- **独立的代码库**: 每个服务都应该有自己的 Git 仓库, `services/` 目录仅用于本地开发时的统一管理。

这种设计使得团队可以并行开发, 互不干扰, 同时也保证了生产环境的隔离性。

## 2. 第一步：定义 API (Protobuf)

一切从 API 定义开始。微服务之间的通信以及网关到服务的通信都使用 gRPC, 其接口由 Protobuf (Protocol Buffers) 定义。

进入你的服务目录 (例如 `services/user-service`), 打开 `api/proto/user.proto`。

```protobuf
syntax = "proto3";

package user;

option go_package = "github.com/yeegeek/uyou-user-service/api/proto";

// UserService 服务
service UserService {
  // 创建用户
  rpc Create(CreateUserRequest) returns (CreateUserResponse);
  
  // 获取用户详情
  rpc Get(GetUserRequest) returns (GetUserResponse);
}

// 消息体定义
message User {
  int64 id = 1;
  string name = 2;
  int64 created_at = 3;
}

message CreateUserRequest {
  string name = 1;
}

message CreateUserResponse {
  int64 id = 1;
  string message = 2;
}

message GetUserRequest {
  int64 id = 1;
}

message GetUserResponse {
  User data = 1;
}
```

**最佳实践与约定**:
- **明确的包名**: `package user;`
- **正确的 `go_package`**: 指向你的 Go 模块路径。
- **权限与频率控制 (基于魔术注释)**:
    - **认证接口**: 在 RPC 定义上方添加 `// @auth` 注释。
    - **限流控制**: 添加 `// @limit(rate=10, burst=20)`，其中 `rate` 为每秒请求数，`burst` 为最大突发请求数。
    - **注意**: 方法名应始终保持 **PascalCase** (首字母大写)，以符合 Go 语言的导出规则。
- **内部接口隔离**: 
    - 如果某些接口仅供微服务间内部调用，不希望暴露给外部网关，请将其定义在以 `.internal.proto` 结尾的文件中 (例如 `user.internal.proto`)。网关部署脚本会自动忽略此类文件。
- **清晰的注释**: 为每个服务和方法添加注释。
- **版本兼容**: 只在末尾添加新字段, 不要修改现有字段的编号。

定义好 API 后, 运行 `make proto` 生成 Go 代码。

```bash
# 在服务目录下运行
make proto
```

## 3. 第二步：实现业务逻辑

代码生成后, 就需要填充业务逻辑了。框架推荐分层架构来组织代码。

### 项目结构概览

```
internal/
├── delivery/        # gRPC 处理器 (接收请求, 调用 Logic)
├── logic/           # 业务逻辑核心 (处理业务, 调用 Repository)
└── repository/      # 数据访问层 (与数据库、缓存交互)
```

### Handler (处理器)

`delivery` 层负责处理 gRPC 请求, 验证输入, 并调用 `logic` 层。

在 `internal/delivery/handler.go` 中:

```go
package handler

import (
	"context"
	pb "github.com/yeegeek/uyou-user-service/api/proto"
)

// Server 结构体实现了 proto 中定义的 UserServiceServer 接口
type Server struct {
	pb.UnimplementedUserServiceServer
}

func New() *Server {
	return &Server{}
}

// Create 方法实现了 Create RPC
func (s *Server) Create(ctx context.Context, req *pb.CreateUserRequest) (*pb.CreateUserResponse, error) {
	// 1. 验证输入
	if req.Name == "" {
		return nil, status.Error(codes.InvalidArgument, "Name is required")
	}

	// 2. 调用 Service 层
	// userId, err := s.userService.Create(ctx, req.Name)

	// 3. 返回响应
	return &pb.CreateUserResponse{Id: 1, Message: "Success"}, nil
}
```

### Service (服务层)

`service` 层是业务逻辑的核心, 它不关心 gRPC 或 HTTP, 只处理纯粹的业务规则。

在 `internal/service/user_service.go` 中:

```go
package service

import "context"

// UserService 定义了业务接口
type UserService interface {
	Create(ctx context.Context, name string) (int64, error)
}

type userService struct {
	// repo repository.UserRepository
}

func NewUserService() UserService {
	return &userService{}
}

func (s *userService) Create(ctx context.Context, name string) (int64, error) {
	// 1. 业务规则检查 (例如: 用户名是否唯一)
	
	// 2. 调用 Repository 持久化数据
	// return s.repo.Create(ctx, name)
	
	return 1, nil
}
```

### Repository (数据访问层)

`repository` 层负责所有与数据存储相关的操作, 将业务逻辑与数据源 (数据库、缓存、文件等) 解耦。

在 `internal/repository/user_repository.go` 中:

```go
package repository

import (
	"context"
	"database/sql"
)

// UserRepository 定义了数据访问接口
type UserRepository interface {
	Create(ctx context.Context, name string) (int64, error)
}

type userRepository struct {
	db *sql.DB
}

func NewUserRepository(db *sql.DB) UserRepository {
	return &userRepository{db: db}
}

func (r *userRepository) Create(ctx context.Context, name string) (int64, error) {
	query := "INSERT INTO users (name) VALUES ($1) RETURNING id"
	var id int64
	err := r.db.QueryRowContext(ctx, query, name).Scan(&id)
	if err != nil {
		return 0, err
	}
	return id, nil
}
```

## 4. 第三步：配置服务

所有配置都应通过 `config/config.yaml` 文件管理, 并使用 Viper 库加载。配置系统支持环境变量覆盖，优先从环境变量读取。

### 4.1 配置文件

```yaml
# config/config.yaml
server:
  grpc_port: 50051
  http_port: 51051

database:
  host: localhost
  port: 5432
  user: postgres
  password: postgres
  dbname: userdb

redis:
  host: localhost
  port: 6379

jwt:
  secret: uyou_secret_key_2026  # 优先从环境变量 JWT_SECRET 读取

log:
  level: info
  format: json
```

### 4.2 环境变量

所有配置项都可以通过环境变量覆盖。环境变量命名规则：
- 使用 `{{SERVICE_NAME_UPPER}}_` 作为前缀（例如：`USER_`）
- 将配置路径中的 `.` 替换为 `_`
- 转换为大写

**完整环境变量列表**:

| 环境变量 | 说明 | 默认值 | 示例 |
|---------|------|--------|------|
| `{{SERVICE_NAME_UPPER}}_SERVER_GRPC_PORT` | gRPC 服务端口 | 50051 | 50051 |
| `{{SERVICE_NAME_UPPER}}_SERVER_HTTP_PORT` | HTTP 健康检查端口 | 51051 | 51051 |
| `{{SERVICE_NAME_UPPER}}_DATABASE_HOST` | 数据库主机 | localhost | postgres |
| `{{SERVICE_NAME_UPPER}}_DATABASE_PORT` | 数据库端口 | 5432 | 5432 |
| `{{SERVICE_NAME_UPPER}}_DATABASE_USER` | 数据库用户名 | postgres | postgres |
| `{{SERVICE_NAME_UPPER}}_DATABASE_PASSWORD` | 数据库密码 | - | postgres |
| `{{SERVICE_NAME_UPPER}}_DATABASE_DBNAME` | 数据库名称 | - | userdb |
| `{{SERVICE_NAME_UPPER}}_DATABASE_SSLMODE` | SSL 模式 | disable | disable |
| `{{SERVICE_NAME_UPPER}}_REDIS_HOST` | Redis 主机 | localhost | redis |
| `{{SERVICE_NAME_UPPER}}_REDIS_PORT` | Redis 端口 | 6379 | 6379 |
| `{{SERVICE_NAME_UPPER}}_REDIS_DB` | Redis 数据库编号 | 0 | 0 |
| `{{SERVICE_NAME_UPPER}}_REDIS_PASSWORD` | Redis 密码 | - | - |
| `JWT_SECRET` | JWT 密钥（全局，所有服务共享） | uyou_secret_key_2026 | your-secret-key |
| `{{SERVICE_NAME_UPPER}}_LOG_LEVEL` | 日志级别 | info | debug, info, warn, error |
| `{{SERVICE_NAME_UPPER}}_LOG_FORMAT` | 日志格式 | json | json, console |

**特殊说明**:
- `JWT_SECRET`: 这是一个全局环境变量，不需要服务名前缀。APISIX 和所有微服务都使用此变量来同步 JWT 密钥。
- 在 `docker-compose.dev.yml` 中，所有服务都会自动从 `.env` 文件读取 `JWT_SECRET`。

在 `cmd/server/main.go` 中加载配置, 并通过依赖注入将数据库连接、服务等传递给 `delivery`。

## 5. 第四步：本地运行与调试

服务生成器已经为你创建了完美的本地开发环境。

```bash
# 在服务目录下运行
make run
```

此命令会:
1. **启动依赖**: 通过 `docker-compose.yml` 启动 PostgreSQL 和 Redis。
2. **运行代码**: 使用 `go run` 启动你的服务。

你可以直接在 VSCode 或 Goland 中设置断点进行调试。

## 6. 第五步：集成到 API 网关

服务在本地运行后, 需要通过 APISIX 网关暴露给外部。

1. **回到项目根目录** (`cd ../../`)。
2. **创建路由文件**: 在 `apisix/config/routes/` 目录下创建一个 YAML 文件, 例如 `user-routes.yaml`。
3. **编写路由规则**: 使用 `grpc-transcode` 插件将 HTTP REST 请求转换为 gRPC 请求。

   ```yaml
   routes:
     - id: "user_get"
       uri: /api/v1/users/*
       plugins:
         grpc-transcode:
           proto_id: "user_service" # 对应 stream_routes 的 ID
           service: "user.UserService" # proto 中定义的服务名
           method: "Get" # proto 中定义的方法名
       upstream:
         nodes:
           "host.docker.internal:50051": 1 # 指向本地运行的服务
         scheme: grpc
   
   stream_routes:
     - id: "user_service"
       server_addr: "0.0.0.0"
       server_port: 50051 # 服务监听的 gRPC 端口
       upstream:
         nodes:
           "host.docker.internal:50051": 1
         scheme: grpc
   ```

   **关键点**:
   - `host.docker.internal`: 允许 Docker 容器 (APISIX) 访问宿主机上运行的服务。
   - `grpc-transcode`: APISIX 的核心插件, 负责协议转换。

### 全局配置管理 (`global.yaml`)

除了单个服务的路由, 网关的全局行为 (如 CORS、监控、消费者) 均在 `apisix/config/global.yaml` 中声明式管理:

- **Global Rules**: 开启全局插件, 如 `cors` (跨域)、`prometheus` (监控)、`client-control` (负载限制)。
- **Consumers**: 管理认证用户及其密钥 (如 JWT Secret)。

修改该文件后, 运行 `make update-routes` 即可生效。

4. **更新路由**: `make update-routes`

## 7. 第六步：构建与部署

### Docker 构建

每个服务都包含一个 `Dockerfile`, 可以轻松构建独立的 Docker 镜像。

```bash
# 在服务目录下运行
make docker-build
```

### 生产部署

生产部署涉及将服务镜像推送到镜像仓库 (如 Docker Hub, Harbor), 并使用 Kubernetes 或其他容器编排工具进行部署。

详细信息请参考 **[生产部署指南](./DEPLOYMENT.md)**。
