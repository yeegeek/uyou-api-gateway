package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

// ServiceConfig 服务配置
type ServiceConfig struct {
	ServiceName   string // 服务名称，如 chat
	ServiceTitle  string // 服务标题，如 Chat
	ModulePath    string // Go 模块路径
	Port          int    // gRPC 端口
	HTTPPort      int    // HTTP 端口(用于健康检查)
	DatabaseType  string // 数据库类型: postgres, mongodb, none
	DatabaseName  string // 数据库名称
	TableName     string // 表名称(PostgreSQL)
	RedisDB       int    // Redis DB
	CachePrefix   string // 缓存前缀
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
		"internal/handler",
		"internal/service",
		"internal/repository",
		"internal/model",
		"pkg/config",
		"pkg/logger",
		"api/proto",
		"config",
		"deployments",
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
	if err := generateReadme(serviceDir, config); err != nil {
		return err
	}
	if err := generateGitignore(serviceDir); err != nil {
		return err
	}

	return nil
}

func generateGoMod(serviceDir string, config *ServiceConfig) error {
	content := fmt.Sprintf(`module %s

go 1.21

require (
	github.com/redis/go-redis/v9 v9.5.1
	github.com/spf13/viper v1.18.2
	go.uber.org/zap v1.27.0
	google.golang.org/grpc v1.60.0
	google.golang.org/protobuf v1.32.0
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
COPY go.mod go.sum ./
RUN go mod download

# 复制源代码
COPY . .

# 构建
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o bin/server cmd/server/main.go

# 运行阶段
FROM alpine:latest

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
	content := fmt.Sprintf(`package main

import (
	"fmt"
	"log"
	"net"
	"net/http"

	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
)

func main() {
	// 启动健康检查 HTTP 服务器
	go startHealthServer()

	// 启动 gRPC 服务器
	lis, err := net.Listen("tcp", ":%d")
	if err != nil {
		log.Fatalf("failed to listen: %%v", err)
	}

	s := grpc.NewServer()
	
	// 注册健康检查服务
	healthServer := health.NewServer()
	grpc_health_v1.RegisterHealthServer(s, healthServer)
	healthServer.SetServingStatus("", grpc_health_v1.HealthCheckResponse_SERVING)
	
	// TODO: 注册业务服务
	// pb.Register%sServiceServer(s, &server{})

	fmt.Printf("🚀 %s Service 启动在端口 %d\n", config.Port)
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %%v", err)
	}
}

func startHealthServer() {
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})
	
	if err := http.ListenAndServe(":%d", nil); err != nil {
		log.Printf("Health server error: %%v", err)
	}
}
`, config.Port, config.ServiceTitle, config.ServiceTitle, config.HTTPPort)
	return os.WriteFile(filepath.Join(serviceDir, "cmd/server/main.go"), []byte(content), 0644)
}

func generateConfig(serviceDir string, config *ServiceConfig) error {
	var content string

	switch config.DatabaseType {
	case "postgres":
		content = fmt.Sprintf(`server:
  grpc_port: %d
  http_port: %d

database:
  host: ${DB_HOST:localhost}
  port: ${DB_PORT:5432}
  user: ${DB_USER:postgres}
  password: ${DB_PASSWORD:postgres}
  dbname: ${DB_NAME:%s}
  sslmode: disable

redis:
  host: ${REDIS_HOST:localhost}
  port: ${REDIS_PORT:6379}
  db: %d
  password: ""

log:
  level: info
  format: json
`, config.Port, config.HTTPPort, config.DatabaseName, config.RedisDB)

	case "mongodb":
		content = fmt.Sprintf(`server:
  grpc_port: %d
  http_port: %d

database:
  host: ${MONGO_HOST:localhost}
  port: ${MONGO_PORT:27017}
  user: ${MONGO_USER:root}
  password: ${MONGO_PASSWORD:example}
  dbname: ${MONGO_DATABASE:%s}

redis:
  host: ${REDIS_HOST:localhost}
  port: ${REDIS_PORT:6379}
  db: %d
  password: ""

log:
  level: info
  format: json
`, config.Port, config.HTTPPort, config.DatabaseName, config.RedisDB)

	case "none":
		content = fmt.Sprintf(`server:
  grpc_port: %d
  http_port: %d

redis:
  host: ${REDIS_HOST:localhost}
  port: ${REDIS_PORT:6379}
  db: %d
  password: ""

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

func printNextSteps(config *ServiceConfig) {
	fmt.Println()
	fmt.Println("╔════════════════════════════════════════╗")
	fmt.Println("║           后续步骤                     ║")
	fmt.Println("╚════════════════════════════════════════╝")
	fmt.Printf("1. 进入服务目录:\n")
	fmt.Printf("   cd services/%s\n", config.ServiceName)
	fmt.Println()
	fmt.Printf("2. 编辑 Proto 文件:\n")
	fmt.Printf("   vim api/proto/%s.proto\n", config.ServiceName)
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
	fmt.Println("📖 详细文档: services/%s/README.md", config.ServiceName)
}
