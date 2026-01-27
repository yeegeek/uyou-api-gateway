package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"
)

// ServiceConfig 服务配置
type ServiceConfig struct {
	ServiceName  string // 服务名称，如 chat
	ServiceTitle string // 服务标题，如 Chat
	ModulePath   string // Go 模块路径
	Port         int    // gRPC 端口
	HTTPPort     int    // HTTP 端口(用于健康检查)
	DatabaseType string // 数据库类型: postgres, mongodb, none
	DatabaseName string // 数据库名称
	TableName    string // 表名称(PostgreSQL)
	RedisDB      int    // Redis DB
	CachePrefix  string // 缓存前缀
}

func main() {
	fmt.Println("╔════════════════════════════════════════╗")
	fmt.Println("║   uyou API Gateway - 服务生成器       ║")
	fmt.Println("╚════════════════════════════════════════╝")
	fmt.Println()

	// 1. 收集用户输入
	config := collectInput()

	// 2. 确认配置
	if !confirmConfig(config) {
		fmt.Println("❌ 已取消")
		return
	}

	// 3. 生成服务
	if err := generateService(config); err != nil {
		fmt.Printf("❌ 生成失败: %v\n", err)
		os.Exit(1)
	}

	// 4. 更新 docker-compose.dev.yml
	if err := updateDevCompose(config); err != nil {
		fmt.Printf("⚠️  更新 docker-compose.dev.yml 失败: %v\n", err)
	} else {
		fmt.Println("✅ 已将服务添加到 docker-compose.dev.yml")
	}

	fmt.Println()
	fmt.Println("✅ 服务生成成功！")
	printNextSteps(config)
}

// collectInput 收集用户输入
func collectInput() *ServiceConfig {
	reader := bufio.NewReader(os.Stdin)
	config := &ServiceConfig{}

	// 服务名称 (不带 -service 后缀)
	config.ServiceName = strings.ToLower(readInput(reader, "服务名称 (如 chat, user, order)", "chat"))
	config.ServiceTitle = strings.Title(config.ServiceName)

	// Go 模块路径
	defaultModule := fmt.Sprintf("github.com/yeegeek/uyou-%s-service", config.ServiceName)
	config.ModulePath = readInput(reader, "Go 模块路径", defaultModule)

	// gRPC 端口
	portStr := readInput(reader, "gRPC 端口", "50051")
	fmt.Sscanf(portStr, "%d", &config.Port)
	config.HTTPPort = config.Port + 1000 // HTTP 端口 = gRPC 端口 + 1000

	// 数据库类型
	fmt.Println()
	fmt.Println("选择数据库类型:")
	fmt.Println("  1. PostgreSQL (适合强一致性场景: 用户、订单、支付)")
	fmt.Println("  2. MongoDB    (适合高吞吐场景: 动态、日志、消息)")
	fmt.Println("  3. None       (无数据库, 仅使用 Redis 或外部 API)")
	dbChoice := readInput(reader, "请选择 [1/2/3]", "1")

	switch dbChoice {
	case "1":
		config.DatabaseType = "postgres"
		config.DatabaseName = readInput(reader, "数据库名称", config.ServiceName+"db")
		config.TableName = readInput(reader, "主表名称", config.ServiceName+"s")
	case "2":
		config.DatabaseType = "mongodb"
		config.DatabaseName = readInput(reader, "数据库名称", config.ServiceName+"db")
	case "3":
		config.DatabaseType = "none"
	default:
		config.DatabaseType = "postgres"
		config.DatabaseName = config.ServiceName + "db"
		config.TableName = config.ServiceName + "s"
	}

	// Redis 配置
	redisDBStr := readInput(reader, "Redis DB (0-15)", "0")
	fmt.Sscanf(redisDBStr, "%d", &config.RedisDB)
	config.CachePrefix = readInput(reader, "缓存前缀", config.ServiceName)

	return config
}

func readInput(reader *bufio.Reader, prompt, defaultValue string) string {
	if defaultValue != "" {
		fmt.Printf("%s [%s]: ", prompt, defaultValue)
	} else {
		fmt.Printf("%s: ", prompt)
	}
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)
	if input == "" {
		return defaultValue
	}
	return input
}

func confirmConfig(config *ServiceConfig) bool {
	fmt.Println()
	fmt.Println("╔════════════════════════════════════════╗")
	fmt.Println("║           配置确认                     ║")
	fmt.Println("╚════════════════════════════════════════╝")
	fmt.Printf("服务名称:   %s\n", config.ServiceName)
	fmt.Printf("模块路径:   %s\n", config.ModulePath)
	fmt.Printf("gRPC 端口:  %d\n", config.Port)
	fmt.Printf("HTTP 端口:  %d\n", config.HTTPPort)

	switch config.DatabaseType {
	case "postgres":
		fmt.Printf("数据库:     PostgreSQL (%s)\n", config.DatabaseName)
		fmt.Printf("表名称:     %s\n", config.TableName)
	case "mongodb":
		fmt.Printf("数据库:     MongoDB (%s)\n", config.DatabaseName)
	case "none":
		fmt.Printf("数据库:     无\n")
	}

	fmt.Printf("Redis DB:   %d\n", config.RedisDB)
	fmt.Printf("缓存前缀:   %s\n", config.CachePrefix)
	fmt.Println()

	reader := bufio.NewReader(os.Stdin)
	confirm := readInput(reader, "确认生成? (y/n)", "y")
	return strings.ToLower(confirm) == "y"
}

func generateService(config *ServiceConfig) error {
	// 确定服务目录 (不带 -service 后缀)
	cwd, _ := os.Getwd()
	var serviceDir string
	if filepath.Base(cwd) == "scaffold" {
		serviceDir = filepath.Join("../services", config.ServiceName)
	} else {
		serviceDir = filepath.Join("services", config.ServiceName)
	}

	fmt.Printf("📁 创建目录: %s\n", serviceDir)

	// 创建目录结构
	dirs := []string{
		"cmd/server",
		"internal/app",
		"internal/delivery",
		"internal/logic",
		"internal/repository",
		"internal/model",
		"pkg/conf",
		"pkg/logger",
		"pkg/errno",
		"pkg/middleware",
		"pkg/ctxout",
		"api/proto",
		"config",
	}

	for _, dir := range dirs {
		fullPath := filepath.Join(serviceDir, dir)
		if err := os.MkdirAll(fullPath, 0755); err != nil {
			return fmt.Errorf("创建目录失败 %s: %w", fullPath, err)
		}
	}

	// 生成文件
	fmt.Println("📝 生成文件...")

	if err := generateGoMod(serviceDir, config); err != nil {
		return err
	}
	if err := generateMakefile(serviceDir, config); err != nil {
		return err
	}
	if err := generateDockerfile(serviceDir, config); err != nil {
		return err
	}
	if err := generateProductionDockerCompose(serviceDir, config); err != nil {
		return err
	}
	if err := generateProto(serviceDir, config); err != nil {
		return err
	}
	if err := generateMain(serviceDir, config); err != nil {
		return err
	}
	if err := generateConfig(serviceDir, config); err != nil {
		return err
	}
	if err := generateEnvExample(serviceDir, config); err != nil {
		return err
	}
	if err := generateReadme(serviceDir, config); err != nil {
		return err
	}
	if err := generateGitignore(serviceDir); err != nil {
		return err
	}

	// 新增 pkg 相关文件
	if err := generatePkgConf(serviceDir, config); err != nil {
		return err
	}
	if err := generatePkgLogger(serviceDir, config); err != nil {
		return err
	}
	if err := generatePkgErrno(serviceDir, config); err != nil {
		return err
	}
	if err := generatePkgMiddleware(serviceDir, config); err != nil {
		return err
	}
	if err := generatePkgContext(serviceDir, config); err != nil {
		return err
	}
	if err := generateInternalApp(serviceDir, config); err != nil {
		return err
	}
	if err := generateInternalLogic(serviceDir, config); err != nil {
		return err
	}
	if err := generateInternalRepository(serviceDir, config); err != nil {
		return err
	}
	if err := generateInternalHandler(serviceDir, config); err != nil {
		return err
	}
	if err := generateInternalProto(serviceDir, config); err != nil {
		return err
	}
	if err := generateInternalModel(serviceDir, config); err != nil {
		return err
	}

	// Generate Proto Go files
	fmt.Println("🔨 生成 Proto 代码 (make proto)...")
	cmdProto := exec.Command("make", "proto")
	cmdProto.Dir = serviceDir
	cmdProto.Stdout = os.Stdout
	cmdProto.Stderr = os.Stderr
	if err := cmdProto.Run(); err != nil {
		fmt.Printf("⚠️  make proto failed: %v\n", err)
	}

	// Run go mod tidy
	fmt.Println("📦 同步依赖 (go mod tidy)...")
	cmd := exec.Command("go", "mod", "tidy")
	cmd.Dir = serviceDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Printf("⚠️  go mod tidy partial fail: %v\n", err)
		// Don't fail the whole generation, let user fix it
	}

	return nil
}

func generateGoMod(serviceDir string, config *ServiceConfig) error {
	content := fmt.Sprintf(`module %s

go 1.21

require (
	github.com/redis/go-redis/v9 v9.7.0
	github.com/spf13/viper v1.19.0
	github.com/joho/godotenv v1.5.1
	github.com/golang-jwt/jwt/v5 v5.2.0
	go.uber.org/zap v1.27.0
	google.golang.org/grpc v1.67.1
	google.golang.org/protobuf v1.35.1
`, config.ModulePath)

	if config.DatabaseType == "postgres" {
		content += `	github.com/lib/pq v1.10.9
`
	} else if config.DatabaseType == "mongodb" {
		content += `	go.mongodb.org/mongo-driver v1.13.1
`
	}

	content += `)
`
	return os.WriteFile(filepath.Join(serviceDir, "go.mod"), []byte(content), 0644)
}

func generateMakefile(serviceDir string, config *ServiceConfig) error {
	content := fmt.Sprintf(`.PHONY: proto build test clean docker-build

# 生成 Proto 代码
proto:
	protoc --go_out=. --go_opt=paths=source_relative \
	       --go-grpc_out=. --go-grpc_opt=paths=source_relative \
	       api/proto/*.proto

# 构建服务
build:
	go build -o bin/%s cmd/server/main.go

# 测试
test:
	go test -v ./...

# 清理
clean:
	rm -rf bin/

# Docker 构建
docker-build:
	docker build -t %s:latest .
`, config.ServiceName, config.ServiceName)
	return os.WriteFile(filepath.Join(serviceDir, "Makefile"), []byte(content), 0644)
}

func generateDockerfile(serviceDir string, config *ServiceConfig) error {
	content := fmt.Sprintf(`# 构建阶段
FROM golang:1.21-alpine AS builder

WORKDIR /app

# 安装构建依赖
RUN apk add --no-cache git

# 复制依赖文件
COPY go.mod ./
COPY go.sum* ./
RUN go mod download || true

# 复制源代码
COPY . .

# 同步依赖
RUN go mod tidy

# 构建
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o bin/server cmd/server/main.go

# 运行阶段
FROM alpine:3.19

WORKDIR /app

# 安装运行时依赖
RUN apk --no-cache add ca-certificates tzdata

# 设置时区
ENV TZ=Asia/Shanghai

# 复制二进制文件和配置
COPY --from=builder /app/bin/server .
COPY --from=builder /app/config ./config

# 暴露端口
EXPOSE %d

# 健康检查
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost:%d/health || exit 1

# 启动服务
CMD ["./server"]
`, config.Port, config.HTTPPort)
	return os.WriteFile(filepath.Join(serviceDir, "Dockerfile"), []byte(content), 0644)
}

func generateProductionDockerCompose(serviceDir string, config *ServiceConfig) error {
	var content string

	switch config.DatabaseType {
	case "postgres":
		content = fmt.Sprintf(`version: '3.8'

services:
  # %s 服务
  %s:
    build:
      context: .
      dockerfile: Dockerfile
    container_name: uyou-%s
    environment:
      DB_HOST: postgres
      DB_PORT: 5432
      DB_USER: postgres
      DB_PASSWORD: ${DB_PASSWORD:-postgres}
      DB_NAME: %s
      REDIS_HOST: redis
      REDIS_PORT: 6379
      REDIS_DB: %d
    ports:
      - "%d:%d"
    depends_on:
      - postgres
      - redis
    networks:
      - uyou-network
    restart: unless-stopped
    healthcheck:
      test: ["CMD", "wget", "--no-verbose", "--tries=1", "--spider", "http://localhost:%d/health"]
      interval: 30s
      timeout: 3s
      retries: 3

  # PostgreSQL 数据库
  postgres:
    image: postgres:15-alpine
    container_name: uyou-%s-postgres
    environment:
      POSTGRES_DB: %s
      POSTGRES_USER: postgres
      POSTGRES_PASSWORD: ${DB_PASSWORD:-postgres}
    volumes:
      - postgres_data:/var/lib/postgresql/data
    networks:
      - uyou-network
    restart: unless-stopped

  # Redis 缓存
  redis:
    image: redis:7-alpine
    container_name: uyou-%s-redis
    volumes:
      - redis_data:/data
    networks:
      - uyou-network
    restart: unless-stopped

networks:
  uyou-network:
    driver: bridge

volumes:
  postgres_data:
  redis_data:
`,
			config.ServiceTitle,
			config.ServiceName,
			config.ServiceName,
			config.DatabaseName,
			config.RedisDB,
			config.Port,
			config.Port,
			config.HTTPPort,
			config.ServiceName,
			config.DatabaseName,
			config.ServiceName,
		)

	case "mongodb":
		content = fmt.Sprintf(`version: '3.8'

services:
  # %s 服务
  %s:
    build:
      context: .
      dockerfile: Dockerfile
    container_name: uyou-%s
    environment:
      MONGO_HOST: mongodb
      MONGO_PORT: 27017
      MONGO_USER: root
      MONGO_PASSWORD: ${MONGO_PASSWORD:-example}
      MONGO_DATABASE: %s
      REDIS_HOST: redis
      REDIS_PORT: 6379
      REDIS_DB: %d
    ports:
      - "%d:%d"
    depends_on:
      - mongodb
      - redis
    networks:
      - uyou-network
    restart: unless-stopped
    healthcheck:
      test: ["CMD", "wget", "--no-verbose", "--tries=1", "--spider", "http://localhost:%d/health"]
      interval: 30s
      timeout: 3s
      retries: 3

  # MongoDB 数据库
  mongodb:
    image: mongo:7
    container_name: uyou-%s-mongodb
    environment:
      MONGO_INITDB_ROOT_USERNAME: root
      MONGO_INITDB_ROOT_PASSWORD: ${MONGO_PASSWORD:-example}
      MONGO_INITDB_DATABASE: %s
    volumes:
      - mongodb_data:/data/db
    networks:
      - uyou-network
    restart: unless-stopped

  # Redis 缓存
  redis:
    image: redis:7-alpine
    container_name: uyou-%s-redis
    volumes:
      - redis_data:/data
    networks:
      - uyou-network
    restart: unless-stopped

networks:
  uyou-network:
    driver: bridge

volumes:
  mongodb_data:
  redis_data:
`,
			config.ServiceTitle,
			config.ServiceName,
			config.ServiceName,
			config.DatabaseName,
			config.RedisDB,
			config.Port,
			config.Port,
			config.HTTPPort,
			config.ServiceName,
			config.DatabaseName,
			config.ServiceName,
		)

	case "none":
		content = fmt.Sprintf(`version: '3.8'

services:
  # %s 服务
  %s:
    build:
      context: .
      dockerfile: Dockerfile
    container_name: uyou-%s
    environment:
      REDIS_HOST: redis
      REDIS_PORT: 6379
      REDIS_DB: %d
    ports:
      - "%d:%d"
    depends_on:
      - redis
    networks:
      - uyou-network
    restart: unless-stopped
    healthcheck:
      test: ["CMD", "wget", "--no-verbose", "--tries=1", "--spider", "http://localhost:%d/health"]
      interval: 30s
      timeout: 3s
      retries: 3

  # Redis 缓存
  redis:
    image: redis:7-alpine
    container_name: uyou-%s-redis
    volumes:
      - redis_data:/data
    networks:
      - uyou-network
    restart: unless-stopped

networks:
  uyou-network:
    driver: bridge

volumes:
  redis_data:
`,
			config.ServiceTitle,
			config.ServiceName,
			config.ServiceName,
			config.RedisDB,
			config.Port,
			config.Port,
			config.HTTPPort,
			config.ServiceName,
		)
	}

	return os.WriteFile(filepath.Join(serviceDir, "docker-compose.yml"), []byte(content), 0644)
}

func generateProto(serviceDir string, config *ServiceConfig) error {
	funcMap := template.FuncMap{
		"lower": strings.ToLower,
		"title": strings.Title,
	}

	tmplText := `syntax = "proto3";

package {{ .ServiceName | lower }};

option go_package = "{{ .ModulePath }}/api/proto";

// {{ .ServiceTitle }} Service
service {{ .ServiceTitle }}Service {
  // 创建{{ .ServiceTitle }}
  // @auth
  // @limit(rate=5, burst=10)
  rpc Create(Create{{ .ServiceTitle }}Request) returns (Create{{ .ServiceTitle }}Response);
  
  // 获取{{ .ServiceTitle }}
  rpc Get(Get{{ .ServiceTitle }}Request) returns (Get{{ .ServiceTitle }}Response);
  
  // 更新{{ .ServiceTitle }}
  rpc Update(Update{{ .ServiceTitle }}Request) returns (Update{{ .ServiceTitle }}Response);
  
  // 删除{{ .ServiceTitle }}
  rpc Delete(Delete{{ .ServiceTitle }}Request) returns (Delete{{ .ServiceTitle }}Response);
  
  // 列表{{ .ServiceTitle }}
  rpc List(List{{ .ServiceTitle }}Request) returns (List{{ .ServiceTitle }}Response);
}

// {{ .ServiceTitle }} 实体
message {{ .ServiceTitle }} {
  int64 id = 1;
  string name = 2;
  int64 created_at = 3;
  int64 updated_at = 4;
}

// 创建请求
message Create{{ .ServiceTitle }}Request {
  string name = 1;
}

message Create{{ .ServiceTitle }}Response {
  int64 id = 1;
  string message = 2;
}

// 获取请求
message Get{{ .ServiceTitle }}Request {
  int64 id = 1;
}

message Get{{ .ServiceTitle }}Response {
  {{ .ServiceTitle }} data = 1;
}

// 更新请求
message Update{{ .ServiceTitle }}Request {
  int64 id = 1;
  string name = 2;
}

message Update{{ .ServiceTitle }}Response {
  bool success = 1;
  string message = 2;
}

// 删除请求
message Delete{{ .ServiceTitle }}Request {
  int64 id = 1;
}

message Delete{{ .ServiceTitle }}Response {
  bool success = 1;
  string message = 2;
}

// 列表请求
message List{{ .ServiceTitle }}Request {
  int32 page = 1;
  int32 page_size = 2;
}

message List{{ .ServiceTitle }}Response {
  repeated {{ .ServiceTitle }} items = 1;
  int64 total = 2;
}
`

	tmpl, err := template.New("proto").Funcs(funcMap).Parse(tmplText)
	if err != nil {
		return err
	}

	protoFile := filepath.Join(serviceDir, "api/proto", config.ServiceName+".proto")
	file, err := os.Create(protoFile)
	if err != nil {
		return err
	}
	defer file.Close()

	return tmpl.Execute(file, config)
}

func generateMain(serviceDir string, config *ServiceConfig) error {
	tmplText := `package main

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"

	"{{ .ModulePath }}/internal/app"
	"{{ .ModulePath }}/pkg/conf"
	"{{ .ModulePath }}/pkg/logger"
)

func main() {
	// 1. 初始化配置
	c := conf.Load()

	// 2. 初始化日志
	log := logger.New(c.Log.Level, c.Log.Format)

	// 3. 构造应用实例
	application, err := app.New(c, log)
	if err != nil {
		log.Fatal("failed to initialize application", logger.Error(err))
	}

	// 4. 启动健康检查 HTTP 服务器
	go startHealthServer(c.Server.HTTPPort, log)

	// 5. 启动 gRPC 服务器
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", c.Server.GRPCPort))
	if err != nil {
		log.Fatal("failed to listen", logger.Int("port", c.Server.GRPCPort), logger.Error(err))
	}

	s := grpc.NewServer(
		app.NewGRPCInterceptors(log, c)...,
	)
	
	// 注册健康检查服务
	healthServer := health.NewServer()
	grpc_health_v1.RegisterHealthServer(s, healthServer)
	healthServer.SetServingStatus("", grpc_health_v1.HealthCheckResponse_SERVING)
	
	// 注册业务服务
	application.RegisterServers(s)

	// 优雅中止处理
	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
		<-sigCh
		log.Info("shutting down gRPC server...")
		s.GracefulStop()
	}()

	log.Info("🚀 {{ .ServiceTitle }} Service started", logger.Int("port", c.Server.GRPCPort))
	if err := s.Serve(lis); err != nil && err != grpc.ErrServerStopped {
		log.Fatal("failed to serve", logger.Error(err))
	}
}

func startHealthServer(port int, log *logger.Logger) {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})
	
	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: mux,
	}

	log.Info("🏥 Health server started", logger.Int("port", port))
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Error("health server error", logger.Error(err))
	}
}
`
	tmpl, err := template.New("main").Parse(tmplText)
	if err != nil {
		return err
	}

	file, err := os.Create(filepath.Join(serviceDir, "cmd/server/main.go"))
	if err != nil {
		return err
	}
	defer file.Close()

	return tmpl.Execute(file, config)
}

func generateConfig(serviceDir string, config *ServiceConfig) error {
	var content string

	switch config.DatabaseType {
	case "postgres":
		content = fmt.Sprintf(`server:
  grpc_port: %d
  http_port: %d

database:
  host: postgres
  port: 5432
  user: postgres
  password: postgres
  dbname: %s
  sslmode: disable

redis:
  host: redis
  port: 6379
  db: %d
  password: ""

jwt:
  secret: your-jwt-secret-key-change-in-production

log:
  level: info
  format: json
`, config.Port, config.HTTPPort, config.DatabaseName, config.RedisDB)

	case "mongodb":
		content = fmt.Sprintf(`server:
  grpc_port: %d
  http_port: %d

database:
  host: mongodb
  port: 27017
  user: root
  password: example
  dbname: %s

redis:
  host: redis
  port: 6379
  db: %d
  password: ""

jwt:
  secret: your-jwt-secret-key-change-in-production

log:
  level: info
  format: json
`, config.Port, config.HTTPPort, config.DatabaseName, config.RedisDB)

	case "none":
		content = fmt.Sprintf(`server:
  grpc_port: %d
  http_port: %d

redis:
  host: redis
  port: 6379
  db: %d
  password: ""

jwt:
  secret: your-jwt-secret-key-change-in-production

log:
  level: info
  format: json
`, config.Port, config.HTTPPort, config.RedisDB)
	}

	return os.WriteFile(filepath.Join(serviceDir, "config/config.yaml"), []byte(content), 0644)
}

func generateReadme(serviceDir string, config *ServiceConfig) error {
	var dbSection string
	switch config.DatabaseType {
	case "postgres":
		dbSection = "PostgreSQL"
	case "mongodb":
		dbSection = "MongoDB"
	case "none":
		dbSection = "无数据库"
	}

	content := fmt.Sprintf(`# %s Service

%s 微服务 - 基于 gRPC 的高性能服务

## 技术栈

- **语言**: Go 1.21+
- **RPC**: gRPC
- **数据库**: %s
- **缓存**: Redis
- **配置**: Viper

## 快速开始

### 1. 安装依赖

`+"```bash"+`
go mod download
`+"```"+`

### 2. 生成 Proto 代码

`+"```bash"+`
make proto
`+"```"+`

### 3. 本地开发

在项目根目录启动开发环境:

`+"```bash"+`
cd ../../
make start dev
`+"```"+`

### 4. 构建

`+"```bash"+`
make build
`+"```"+`

### 5. Docker 部署

`+"```bash"+`
# 构建镜像
make docker-build

# 启动服务(包含依赖)
docker compose up -d
`+"```"+`

## 项目结构

`+"```"+`
.
├── cmd/
│   └── server/          # 服务入口
├── internal/
│   ├── handler/         # gRPC 处理器
│   ├── service/         # 业务逻辑
│   ├── repository/      # 数据访问
│   └── model/           # 数据模型
├── pkg/
│   ├── config/          # 配置管理
│   └── logger/          # 日志
├── api/
│   └── proto/           # Proto 定义
├── config/              # 配置文件
├── docker-compose.yml   # 生产环境配置
├── Dockerfile           # 容器镜像
└── Makefile            # 构建命令
`+"```"+`

## 环境变量

| 变量 | 说明 | 默认值 |
|------|------|--------|
| GRPC_PORT | gRPC 端口 | %d |
| HTTP_PORT | HTTP 端口 | %d |
`,
		config.ServiceTitle,
		config.ServiceTitle,
		dbSection,
		config.Port,
		config.HTTPPort,
	)

	if config.DatabaseType == "postgres" {
		content += fmt.Sprintf(`| DB_HOST | 数据库主机 | localhost |
| DB_PORT | 数据库端口 | 5432 |
| DB_NAME | 数据库名称 | %s |
| DB_USER | 数据库用户 | postgres |
| DB_PASSWORD | 数据库密码 | postgres |
`, config.DatabaseName)
	} else if config.DatabaseType == "mongodb" {
		content += fmt.Sprintf(`| MONGO_HOST | 数据库主机 | localhost |
| MONGO_PORT | 数据库端口 | 27017 |
| MONGO_DATABASE | 数据库名称 | %s |
| MONGO_USER | 数据库用户 | root |
| MONGO_PASSWORD | 数据库密码 | example |
`, config.DatabaseName)
	}

	content += `| REDIS_HOST | Redis 主机 | localhost |
| REDIS_PORT | Redis 端口 | 6379 |
| REDIS_DB | Redis DB | 0 |

## License

MIT
`

	return os.WriteFile(filepath.Join(serviceDir, "README.md"), []byte(content), 0644)
}

func generateGitignore(serviceDir string) error {
	content := `# Binaries
bin/
*.exe
*.dll
*.so
*.dylib

# Test
*.test
*.out

# Go
go.work

# IDE
.vscode/
.idea/
*.swp

# OS
.DS_Store

# Generated
*.pb.go

# Env
.env
.env.local

# Logs
*.log
`
	return os.WriteFile(filepath.Join(serviceDir, ".gitignore"), []byte(content), 0644)
}

func updateDevCompose(config *ServiceConfig) error {
	// 读取现有的 docker-compose.dev.yml
	cwd, _ := os.Getwd()
	var devComposeFile string
	if filepath.Base(cwd) == "scaffold" {
		devComposeFile = "../docker-compose.dev.yml"
	} else {
		devComposeFile = "docker-compose.dev.yml"
	}

	content, err := os.ReadFile(devComposeFile)
	if err != nil {
		return err
	}

	contentStr := string(content)

	// 检查服务是否已存在
	if strings.Contains(contentStr, config.ServiceName+":") {
		return nil // 服务已存在, 不重复添加
	}

	// 生成新服务的配置
	var serviceConfig string
	var dependsOn string

	switch config.DatabaseType {
	case "postgres":
		dependsOn = `    depends_on:
      - postgres
      - redis`
	case "mongodb":
		dependsOn = `    depends_on:
      - mongodb
      - redis`
	case "none":
		dependsOn = `    depends_on:
      - redis`
	}

	serviceConfig = fmt.Sprintf(`
  # %s Service
  %s:
    build:
      context: ./services/%s
      dockerfile: Dockerfile
    container_name: uyou-%s-dev
    environment:`, config.ServiceTitle, config.ServiceName, config.ServiceName, config.ServiceName)

	if config.DatabaseType == "postgres" {
		serviceConfig += fmt.Sprintf(`
      DB_HOST: postgres
      DB_PORT: 5432
      DB_USER: postgres
      DB_PASSWORD: postgres
      DB_NAME: %s`, config.DatabaseName)
	} else if config.DatabaseType == "mongodb" {
		serviceConfig += fmt.Sprintf(`
      MONGO_HOST: mongodb
      MONGO_PORT: 27017
      MONGO_USER: root
      MONGO_PASSWORD: example
      MONGO_DATABASE: %s`, config.DatabaseName)
	}

	serviceConfig += fmt.Sprintf(`
      REDIS_HOST: redis
      REDIS_PORT: 6379
      REDIS_DB: %d
    ports:
      - "%d:%d"
%s
    networks:
      - uyou-network
    restart: unless-stopped
`, config.RedisDB, config.Port, config.Port, dependsOn)

	// 在 "# 新生成的微服务将自动添加到此处" 之前插入
	marker := "  # 新生成的微服务将自动添加到此处"
	contentStr = strings.Replace(contentStr, marker, serviceConfig+"\n"+marker, 1)

	// 写回文件
	return os.WriteFile(devComposeFile, []byte(contentStr), 0644)
}

func generatePkgConf(serviceDir string, config *ServiceConfig) error {
	content := fmt.Sprintf(`package conf

import (
	"strings"

	"github.com/joho/godotenv"
	"github.com/spf13/viper"
)

type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	Redis    RedisConfig
	JWT      JWTConfig
	Log      LogConfig
}

type ServerConfig struct {
	GRPCPort int `+"`mapstructure:\"grpc_port\"`"+`
	HTTPPort int `+"`mapstructure:\"http_port\"`"+`
}

type DatabaseConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	DBName   string `+"`mapstructure:\"dbname\"`"+`
	SSLMode  string `+"`mapstructure:\"sslmode\"`"+`
}

type RedisConfig struct {
	Host     string
	Port     int
	DB       int
	Password string
}

type JWTConfig struct {
	Secret string
}

type LogConfig struct {
	Level  string
	Format string
}

func Load() *Config {
	v := viper.New()
	v.SetConfigName("config")
	v.SetConfigType("yaml")
	v.AddConfigPath("./config")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.SetEnvPrefix("%s")
	v.AutomaticEnv()

	// Load .env file if exists
	_ = godotenv.Load()

	if err := v.ReadInConfig(); err != nil {
		// 容忍配置文件不存在，可能全靠环境变量
	}

	conf := &Config{}
	if err := v.Unmarshal(conf); err != nil {
		panic(err)
	}
	return conf
}
`, strings.ToUpper(config.ServiceName))
	return os.WriteFile(filepath.Join(serviceDir, "pkg/conf/conf.go"), []byte(content), 0644)
}

func generatePkgLogger(serviceDir string, config *ServiceConfig) error {
	content := `package logger

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type Logger = zap.Logger

var (
	String = zap.String
	Int    = zap.Int
	Error  = zap.Error
	Any    = zap.Any
)

func New(level, format string) *Logger {
	var l zapcore.Level
	if err := l.UnmarshalText([]byte(level)); err != nil {
		l = zapcore.InfoLevel
	}

	var zapConfig zap.Config
	if format == "json" {
		zapConfig = zap.NewProductionConfig()
	} else {
		zapConfig = zap.NewDevelopmentConfig()
	}
	
	zapConfig.Level = zap.NewAtomicLevelAt(l)
	logger, _ := zapConfig.Build()
	return logger
}
`
	return os.WriteFile(filepath.Join(serviceDir, "pkg/logger/logger.go"), []byte(content), 0644)
}

func generatePkgErrno(serviceDir string, config *ServiceConfig) error {
	content := `package errno

import (
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Errno struct {
	Code    int32
	Message string
}

func (e Errno) Error() string {
	return e.Message
}

func (e Errno) GRPCStatus() *status.Status {
	return status.New(codes.Code(e.Code), e.Message)
}

func (e Errno) WithMessage(msg string) Errno {
	e.Message = msg
	return e
}

var (
	Success          = Errno{Code: 0, Message: "Success"}
	InternalError    = Errno{Code: 13, Message: "Internal Server Error"}
	InvalidArgument = Errno{Code: 3, Message: "Invalid Argument"}
	NotFound        = Errno{Code: 5, Message: "Not Found"}
	AlreadyExists   = Errno{Code: 6, Message: "Already Exists"}
	PermissionDenied = Errno{Code: 7, Message: "Permission Denied"}
	Unauthenticated  = Errno{Code: 16, Message: "Unauthenticated"}
)

func FromError(err error) Errno {
	if err == nil {
		return Success
	}
	if e, ok := err.(Errno); ok {
		return e
	}
	s, _ := status.FromError(err)
	return Errno{Code: int32(s.Code()), Message: s.Message()}
}
`
	return os.WriteFile(filepath.Join(serviceDir, "pkg/errno/errno.go"), []byte(content), 0644)
}

func generatePkgMiddleware(serviceDir string, config *ServiceConfig) error {
	content := fmt.Sprintf(`package middleware

import (
	"context"
	"runtime/debug"
	"time"

	"%s/pkg/logger"
	"%s/pkg/ctxout"
	"github.com/golang-jwt/jwt/v5"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"go.uber.org/zap"
)

// UnaryRecoveryInterceptor handles panic recovery
func UnaryRecoveryInterceptor(log *logger.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
		defer func() {
			if r := recover(); r != nil {
				log.Error("panic recovered",
					logger.Any("panic", r),
					logger.String("stack", string(debug.Stack())),
				)
				err = status.Errorf(codes.Internal, "Internal server error")
			}
		}()
		return handler(ctx, req)
	}
}

// UnaryLoggingInterceptor logs method calls
func UnaryLoggingInterceptor(log *logger.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		start := time.Now()
		resp, err := handler(ctx, req)
		duration := time.Since(start)

		fields := []zap.Field{
			logger.String("method", info.FullMethod),
			zap.Duration("duration", duration),
			logger.String("user_id", ctxout.GetUserID(ctx)),
		}

		if err != nil {
			fields = append(fields, logger.Error(err))
			log.Error("gRPC request failed", fields...)
		} else {
			log.Info("gRPC request success", fields...)
		}

		return resp, err
	}
}

// UnaryAuthInterceptor extracts user ID from JWT token in metadata.
func UnaryAuthInterceptor(secret string) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		md, ok := metadata.FromIncomingContext(ctx)
		if ok {
			var tokenString string
			
			// 1. Try standard Authorization header
			authHeader := md.Get("authorization")
			if len(authHeader) > 0 {
				tokenString = authHeader[0]
				if len(tokenString) > 7 && tokenString[:7] == "Bearer " {
					tokenString = tokenString[7:]
				}
			}

			// 2. Try x-user-id (if acting as token carrier per user feedback)
			if tokenString == "" {
				vals := md.Get("x-user-id")
				if len(vals) > 0 {
					tokenString = vals[0]
				}
			}

			if tokenString != "" {
				token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
					return []byte(secret), nil
				})

				if err == nil && token.Valid {
					if claims, ok := token.Claims.(jwt.MapClaims); ok {
						if userID, ok := claims["user_id"].(string); ok {
							ctx = context.WithValue(ctx, ctxout.KeyUserID, userID)
						} else if sub, ok := claims["sub"].(string); ok {
							// Fallback to sub as user ID
							ctx = context.WithValue(ctx, ctxout.KeyUserID, sub)
						}
					}
				}
			}
		}
		return handler(ctx, req)
	}
}
`, config.ModulePath, config.ModulePath)
	return os.WriteFile(filepath.Join(serviceDir, "pkg/middleware/interceptors.go"), []byte(content), 0644)
}

func generateInternalApp(serviceDir string, config *ServiceConfig) error {
	tmplText := `package app

import (
	"google.golang.org/grpc"
	"{{ .ModulePath }}/pkg/conf"
	"{{ .ModulePath }}/pkg/logger"
	"{{ .ModulePath }}/pkg/middleware"
	"{{ .ModulePath }}/internal/delivery"
	"{{ .ModulePath }}/internal/logic"
	"{{ .ModulePath }}/internal/repository"
	pb "{{ .ModulePath }}/api/proto"
)

type App struct {
	conf   *conf.Config
	log    *logger.Logger
	logic  *logic.Logic
}

func New(c *conf.Config, l *logger.Logger) (*App, error) {
	// 1. 初始化 Repository
	repo, err := repository.New(c, l)
	if err != nil {
		return nil, err
	}

	// 2. 初始化 Logic
	lgc := logic.New(c, l, repo)

	return &App{
		conf:  c,
		log:   l,
		logic: lgc,
	}, nil
}

func (a *App) RegisterServers(s *grpc.Server) {
	// 注册公开接口 (由网关同步)
	hdl := delivery.New(a.logic, a.log)
	pb.Register{{ .ServiceTitle }}ServiceServer(s, hdl)
	
	// 注册内部接口 (网关忽略)
	intHdl := delivery.NewInternal(a.logic, a.log)
	pb.RegisterInternal{{ .ServiceTitle }}ServiceServer(s, intHdl)
}

	func NewGRPCInterceptors(l *logger.Logger, cfg *conf.Config) []grpc.ServerOption {
	return []grpc.ServerOption{
		grpc.ChainUnaryInterceptor(
			middleware.UnaryRecoveryInterceptor(l),
			middleware.UnaryLoggingInterceptor(l),
			middleware.UnaryAuthInterceptor(cfg.JWT.Secret),
		),
	}
}
`
	tmpl, err := template.New("app").Parse(tmplText)
	if err != nil {
		return err
	}

	file, err := os.Create(filepath.Join(serviceDir, "internal/app/app.go"))
	if err != nil {
		return err
	}
	defer file.Close()

	return tmpl.Execute(file, config)
}

func generateInternalLogic(serviceDir string, config *ServiceConfig) error {
	content := fmt.Sprintf(`package logic

import (
	"context"
	"time"

	"%s/internal/model"
	"%s/internal/repository"
	"%s/pkg/conf"
	"%s/pkg/errno"
	"%s/pkg/logger"
)

type Logic struct {
	conf *conf.Config
	log  *logger.Logger
	repo *repository.Repository
}

func New(c *conf.Config, l *logger.Logger, r *repository.Repository) *Logic {
	return &Logic{
		conf: c,
		log:  l,
		repo: r,
	}
}

// Create%s 创建%s
func (l *Logic) Create%s(ctx context.Context, name string) (int64, error) {
	l.log.Info("Creating %s", logger.String("name", name))
	
	item := &model.%s{
		Name:      name,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	
	if err := l.repo.Create%s(ctx, item); err != nil {
		l.log.Error("Failed to create %s", logger.Error(err))
		return 0, errno.InternalError
	}
	
	return item.ID, nil
}

// Get%s 获取%s
func (l *Logic) Get%s(ctx context.Context, id int64) (*model.%s, error) {
	item, err := l.repo.Get%sByID(ctx, id)
	if err != nil {
		return nil, errno.InternalError
	}
	if item == nil {
		return nil, errno.NotFound.WithMessage("%s not found")
	}
	return item, nil
}

// Update%s 更新%s
func (l *Logic) Update%s(ctx context.Context, id int64, name string) error {
	item, err := l.repo.Get%sByID(ctx, id)
	if err != nil {
		return errno.InternalError
	}
	if item == nil {
		return errno.NotFound.WithMessage("%s not found")
	}
	
	item.Name = name
	item.UpdatedAt = time.Now()
	
	if err := l.repo.Update%s(ctx, item); err != nil {
		return errno.InternalError
	}
	return nil
}

// Delete%s 删除%s
func (l *Logic) Delete%s(ctx context.Context, id int64) error {
	if err := l.repo.Delete%s(ctx, id); err != nil {
		return errno.InternalError
	}
	return nil
}

// List%s 列表%s (简单实现)
func (l *Logic) List%s(ctx context.Context, page, pageSize int) ([]*model.%s, int64, error) {
	return l.repo.List%s(ctx, page, pageSize)
}
`, config.ModulePath, config.ModulePath, config.ModulePath, config.ModulePath, config.ModulePath,
		config.ServiceTitle, config.ServiceName, config.ServiceTitle, strings.ToLower(config.ServiceName), config.ServiceTitle, config.ServiceTitle, strings.ToLower(config.ServiceName),
		config.ServiceTitle, config.ServiceName, config.ServiceTitle, config.ServiceTitle, config.ServiceTitle, config.ServiceTitle,
		config.ServiceTitle, config.ServiceName, config.ServiceTitle, config.ServiceTitle, config.ServiceTitle, config.ServiceTitle,
		config.ServiceTitle, config.ServiceName, config.ServiceTitle, config.ServiceTitle,
		config.ServiceTitle+"s", config.ServiceName, config.ServiceTitle+"s", config.ServiceTitle, config.ServiceTitle+"s")
	return os.WriteFile(filepath.Join(serviceDir, "internal/logic/logic.go"), []byte(content), 0644)
}

func generateInternalRepository(serviceDir string, config *ServiceConfig) error {
	content := fmt.Sprintf(`package repository

import (
	"context"

	"%s/internal/model"
	"%s/pkg/conf"
	"%s/pkg/logger"
)

type Repository struct {
	conf   *conf.Config
	log    *logger.Logger
	users  map[int64]*model.%s // Simple in-memory storage for demo
	nextID int64
}

func New(c *conf.Config, l *logger.Logger) (*Repository, error) {
	return &Repository{
		conf:   c,
		log:    l,
		users:  make(map[int64]*model.%s),
		nextID: 1,
	}, nil
}

// Create%s creates a new %s
func (r *Repository) Create%s(ctx context.Context, item *model.%s) error {
	item.ID = r.nextID
	r.nextID++
	r.users[item.ID] = item
	return nil
}

// Get%sByID gets a %s by ID
func (r *Repository) Get%sByID(ctx context.Context, id int64) (*model.%s, error) {
	item, ok := r.users[id]
	if !ok {
		return nil, nil
	}
	return item, nil
}

// Update%s updates a %s
func (r *Repository) Update%s(ctx context.Context, item *model.%s) error {
	r.users[item.ID] = item
	return nil
}

// Delete%s deletes a %s
func (r *Repository) Delete%s(ctx context.Context, id int64) error {
	delete(r.users, id)
	return nil
}

// List%s lists %s with pagination
func (r *Repository) List%s(ctx context.Context, page, pageSize int) ([]*model.%s, int64, error) {
	var items []*model.%s
	for _, item := range r.users {
		items = append(items, item)
	}
	// In a real implementation, you would apply pagination here
	return items, int64(len(items)), nil
}
`, config.ModulePath, config.ModulePath, config.ModulePath,
		config.ServiceTitle, config.ServiceTitle,
		config.ServiceTitle, strings.ToLower(config.ServiceTitle), config.ServiceTitle, config.ServiceTitle,
		config.ServiceTitle, strings.ToLower(config.ServiceTitle), config.ServiceTitle, config.ServiceTitle,
		config.ServiceTitle, strings.ToLower(config.ServiceTitle), config.ServiceTitle, config.ServiceTitle,
		config.ServiceTitle, strings.ToLower(config.ServiceTitle), config.ServiceTitle,
		config.ServiceTitle, strings.ToLower(config.ServiceTitle)+"s", config.ServiceTitle+"s", config.ServiceTitle, config.ServiceTitle)
	return os.WriteFile(filepath.Join(serviceDir, "internal/repository/repository.go"), []byte(content), 0644)
}

func generateInternalHandler(serviceDir string, config *ServiceConfig) error {
	content := fmt.Sprintf(`package delivery

import (
	"context"
	"%s/internal/logic"
	"%s/pkg/errno"
	"%s/pkg/logger"
	pb "%s/api/proto"
)

type Handler struct {
	pb.Unimplemented%sServiceServer
	logic *logic.Logic
	log   *logger.Logger
}

func New(l *logic.Logic, log *logger.Logger) *Handler {
	return &Handler{
		logic: l,
		log:   log,
	}
}

// Create 创建%s
func (h *Handler) Create(ctx context.Context, req *pb.Create%sRequest) (*pb.Create%sResponse, error) {
	id, err := h.logic.Create%s(ctx, req.Name)
	if err != nil {
		return nil, errno.FromError(err).GRPCStatus().Err()
	}

	return &pb.Create%sResponse{
		Id:      id,
		Message: "%s created successfully",
	}, nil
}

// Get 获取%s
func (h *Handler) Get(ctx context.Context, req *pb.Get%sRequest) (*pb.Get%sResponse, error) {
	item, err := h.logic.Get%s(ctx, req.Id)
	if err != nil {
		return nil, errno.FromError(err).GRPCStatus().Err()
	}

	return &pb.Get%sResponse{
		Data: item.ToProto(),
	}, nil
}

// Update 更新%s
func (h *Handler) Update(ctx context.Context, req *pb.Update%sRequest) (*pb.Update%sResponse, error) {
	err := h.logic.Update%s(ctx, req.Id, req.Name)
	if err != nil {
		return nil, errno.FromError(err).GRPCStatus().Err()
	}

	return &pb.Update%sResponse{
		Success: true,
		Message: "%s updated successfully",
	}, nil
}

// Delete 删除%s
func (h *Handler) Delete(ctx context.Context, req *pb.Delete%sRequest) (*pb.Delete%sResponse, error) {
	err := h.logic.Delete%s(ctx, req.Id)
	if err != nil {
		return nil, errno.FromError(err).GRPCStatus().Err()
	}

	return &pb.Delete%sResponse{
		Success: true,
		Message: "%s deleted successfully",
	}, nil
}

// List 列表%s
func (h *Handler) List(ctx context.Context, req *pb.List%sRequest) (*pb.List%sResponse, error) {
	items, total, err := h.logic.List%s(ctx, int(req.Page), int(req.PageSize))
	if err != nil {
		return nil, errno.FromError(err).GRPCStatus().Err()
	}

	var pbItems []*pb.%s
	for _, item := range items {
		pbItems = append(pbItems, item.ToProto())
	}

	return &pb.List%sResponse{
		Items: pbItems,
		Total: total,
	}, nil
}
`, config.ModulePath, config.ModulePath, config.ModulePath, config.ModulePath, config.ServiceTitle,
		config.ServiceName, config.ServiceTitle, config.ServiceTitle, config.ServiceTitle, config.ServiceTitle, config.ServiceName,
		config.ServiceName, config.ServiceTitle, config.ServiceTitle, config.ServiceTitle, config.ServiceTitle,
		config.ServiceName, config.ServiceTitle, config.ServiceTitle, config.ServiceTitle, config.ServiceTitle, config.ServiceName,
		config.ServiceName, config.ServiceTitle, config.ServiceTitle, config.ServiceTitle, config.ServiceTitle, config.ServiceName,
		config.ServiceName, config.ServiceTitle, config.ServiceTitle, config.ServiceTitle+"s", config.ServiceTitle, config.ServiceTitle)

	// 同步生成 internal.go
	internalContent := fmt.Sprintf(`package delivery

import (
	"context"

	"%s/internal/logic"
	"%s/pkg/logger"
	pb "%s/api/proto"
)

type InternalHandler struct {
	pb.UnimplementedInternal%sServiceServer
	logic *logic.Logic
	log   *logger.Logger
}

func NewInternal(l *logic.Logic, log *logger.Logger) *InternalHandler {
	return &InternalHandler{
		logic: l,
		log:   log,
	}
}

// InternalSync 实例内部通信方法
func (h *InternalHandler) InternalSync(ctx context.Context, req *pb.InternalSyncRequest) (*pb.InternalSyncResponse, error) {
	h.log.Info("received internal sync request", logger.String("msg", req.Msg))
	return &pb.InternalSyncResponse{Success: true}, nil
}
`, config.ModulePath, config.ModulePath, config.ModulePath, config.ServiceTitle)

	if err := os.WriteFile(filepath.Join(serviceDir, "internal/delivery/handler.go"), []byte(content), 0644); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(serviceDir, "internal/delivery/internal.go"), []byte(internalContent), 0644)
}

func generateInternalProto(serviceDir string, config *ServiceConfig) error {
	tmplText := `syntax = "proto3";

package {{ .ServiceName | lower }};

option go_package = "{{ .ModulePath }}/api/proto";

// Internal{{ .ServiceTitle }}Service 仅限微服务间内部调用的接口 (网关已忽略)
service Internal{{ .ServiceTitle }}Service {
  // 内部同步示例
  rpc InternalSync(InternalSyncRequest) returns (InternalSyncResponse);
}

message InternalSyncRequest {
  string msg = 1;
}

message InternalSyncResponse {
  bool success = 1;
}
`
	tmpl, err := template.New("internal_proto").Funcs(template.FuncMap{"lower": strings.ToLower}).Parse(tmplText)
	if err != nil {
		return err
	}

	protoFile := filepath.Join(serviceDir, "api/proto", config.ServiceName+".internal.proto")
	file, err := os.Create(protoFile)
	if err != nil {
		return err
	}
	defer file.Close()

	return tmpl.Execute(file, config)
}

func generateInternalModel(serviceDir string, config *ServiceConfig) error {
	var modelTmpl string
	if config.DatabaseType == "postgres" {
		modelTmpl = fmt.Sprintf(`package model

import (
	"time"

	pb "%s/api/proto"
)

// %s 数据库模型示例 (PostgreSQL)
type %s struct {
	ID        int64     `+"`gorm:\"primaryKey;autoIncrement\" json:\"id\"`"+`
	Name      string    `+"`gorm:\"size:255;not null\" json:\"name\"`"+`
	CreatedAt time.Time `+"`json:\"created_at\"`"+`
	UpdatedAt time.Time `+"`json:\"updated_at\"`"+`
}

func (m *%s) TableName() string {
	return "%s"
}

// ToProto 转换为 Protobuf 消息
func (m *%s) ToProto() *pb.%s {
	return &pb.%s{
		Id:        m.ID,
		Name:      m.Name,
		CreatedAt: m.CreatedAt.Unix(),
		UpdatedAt: m.UpdatedAt.Unix(),
	}
}
`, config.ModulePath, config.ServiceTitle, config.ServiceTitle, config.ServiceTitle, config.TableName, config.ServiceTitle, config.ServiceTitle, config.ServiceTitle)
	} else if config.DatabaseType == "mongodb" {
		modelTmpl = fmt.Sprintf(`package model

import (
	"time"

	pb "%s/api/proto"
)

// %s 数据库模型示例 (MongoDB)
type %s struct {
	ID        string    `+"`bson:\"_id,omitempty\" json:\"id\"`"+`
	Name      string    `+"`bson:\"name\" json:\"name\"`"+`
	CreatedAt time.Time `+"`bson:\"created_at\" json:\"created_at\"`"+`
	UpdatedAt time.Time `+"`bson:\"updated_at\" json:\"updated_at\"`"+`
}

func (m *%s) ToProto() *pb.%s {
	// 简单示例: 将 ID 哈希为 int64 或调整 proto 定义使用 string id
	return &pb.%s{
		// Id: ... 
		Name:      m.Name,
		CreatedAt: m.CreatedAt.Unix(),
		UpdatedAt: m.UpdatedAt.Unix(),
	}
}
`, config.ModulePath, config.ServiceTitle, config.ServiceTitle, config.ServiceTitle, config.ServiceTitle, config.ServiceTitle)
	} else {
		modelTmpl = fmt.Sprintf(`package model

// %s 模型示例
type %s struct {
	ID   int64  `+"`json:\"id\"`"+`
	Name string `+"`json:\"name\"`"+`
}
`, config.ServiceTitle, config.ServiceTitle)
	}

	return os.WriteFile(filepath.Join(serviceDir, "internal/model", config.ServiceName+".go"), []byte(modelTmpl), 0644)
}

func printNextSteps(config *ServiceConfig) {
	fmt.Println()
	fmt.Println("╔════════════════════════════════════════╗")
	fmt.Println("║           后续步骤                     ║")
	fmt.Println("╚════════════════════════════════════════╝")
	fmt.Printf("1. 进入服务目录:\n")
	fmt.Printf("   cd services/%s\n", config.ServiceName)
	fmt.Println()
	fmt.Printf("2. 安装依赖并同步:\n")
	fmt.Printf("   go mod tidy\n")
	fmt.Println()
	fmt.Printf("3. 生成 gRPC 代码:\n")
	fmt.Printf("   make proto\n")
	fmt.Println()
	fmt.Printf("4. 实现业务逻辑:\n")
	fmt.Printf("   - internal/handler/  (gRPC 处理器)\n")
	fmt.Printf("   - internal/service/  (业务逻辑)\n")
	fmt.Printf("   - internal/repository/ (数据访问)\n")
	fmt.Println()
	fmt.Printf("5. 本地开发:\n")
	fmt.Printf("   cd ../../\n")
	fmt.Printf("   make start dev  # 启动所有开发环境服务\n")
	fmt.Println()
	fmt.Printf("6. 配置 APISIX 路由:\n")
	fmt.Printf("   - 在 apisix/config/routes/ 创建 %s-routes.yaml\n", config.ServiceName)
	fmt.Printf("   - make update-routes\n")
	fmt.Println()
	fmt.Printf("7. 测试 API:\n")
	fmt.Printf("   curl http://localhost:9080/api/v1/%s/...\n", config.ServiceName)
	fmt.Println()
	fmt.Printf("📖 详细文档: services/%s/README.md\n", config.ServiceName)
}

func generateEnvExample(serviceDir string, config *ServiceConfig) error {
	var content string

	common := fmt.Sprintf(`# Server
%s_SERVER_GRPC_PORT=%d
%s_SERVER_HTTP_PORT=%d

# Redis
%s_REDIS_HOST=localhost
%s_REDIS_PORT=6379
%s_REDIS_DB=%d
%s_REDIS_PASSWORD=

# JWT
%s_JWT_SECRET=your-jwt-secret-key-change-in-production

# Log
%s_LOG_LEVEL=info
%s_LOG_FORMAT=json
`,
		strings.ToUpper(config.ServiceName), config.Port,
		strings.ToUpper(config.ServiceName), config.HTTPPort,
		strings.ToUpper(config.ServiceName),
		strings.ToUpper(config.ServiceName),
		strings.ToUpper(config.ServiceName), config.RedisDB,
		strings.ToUpper(config.ServiceName),
		strings.ToUpper(config.ServiceName),
		strings.ToUpper(config.ServiceName),
		strings.ToUpper(config.ServiceName),
	)

	switch config.DatabaseType {
	case "postgres":
		content = fmt.Sprintf(`# Database (PostgreSQL)
%s_DATABASE_HOST=localhost
%s_DATABASE_PORT=5432
%s_DATABASE_USER=postgres
%s_DATABASE_PASSWORD=postgres
%s_DATABASE_DBNAME=%s
%s_DATABASE_SSLMODE=disable

%s`,
			strings.ToUpper(config.ServiceName),
			strings.ToUpper(config.ServiceName),
			strings.ToUpper(config.ServiceName),
			strings.ToUpper(config.ServiceName),
			strings.ToUpper(config.ServiceName), config.DatabaseName,
			strings.ToUpper(config.ServiceName),
			common,
		)
	case "mongodb":
		content = fmt.Sprintf(`# Database (MongoDB)
%s_DATABASE_HOST=localhost
%s_DATABASE_PORT=27017
%s_DATABASE_USER=root
%s_DATABASE_PASSWORD=example
%s_DATABASE_DBNAME=%s

%s`,
			strings.ToUpper(config.ServiceName),
			strings.ToUpper(config.ServiceName),
			strings.ToUpper(config.ServiceName),
			strings.ToUpper(config.ServiceName),
			strings.ToUpper(config.ServiceName), config.DatabaseName,
			common,
		)
	default:
		content = common
	}

	return os.WriteFile(filepath.Join(serviceDir, ".env.example"), []byte(content), 0644)
}

func generatePkgContext(serviceDir string, config *ServiceConfig) error {
	content := `package ctxout

import (
	"context"

	"google.golang.org/grpc/metadata"
)

const (
	// KeyUserID is the key for user ID in context
	KeyUserID = "x-user-id"
)

// GetUserID extracts the user ID from the context.
// It checks both the incoming context metadata (gRPC headers) and values attached to the context.
func GetUserID(ctx context.Context) string {
	// 1. Try to get from metadata (incoming gRPC header)
	md, ok := metadata.FromIncomingContext(ctx)
	if ok {
		vals := md.Get(KeyUserID)
		if len(vals) > 0 && vals[0] != "" {
			return vals[0]
		}
	}

	// 2. Try to get from context value (set by middleware)
	if v := ctx.Value(KeyUserID); v != nil {
		if s, ok := v.(string); ok {
			return s
		}
	}

	return ""
}
`
	return os.WriteFile(filepath.Join(serviceDir, "pkg/ctxout/ctxout.go"), []byte(content), 0644)
}
