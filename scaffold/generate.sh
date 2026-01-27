#!/bin/bash

# uyou API Gateway - 微服务脚手架生成器
# 使用方法: ./generate.sh [服务名称]

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 获取脚本所在目录
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TEMPLATES_DIR="$SCRIPT_DIR/templates"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

# 打印带颜色的消息
print_info() { echo -e "${BLUE}$1${NC}"; }
print_success() { echo -e "${GREEN}$1${NC}"; }
print_warning() { echo -e "${YELLOW}$1${NC}"; }
print_error() { echo -e "${RED}$1${NC}"; }

# 读取用户输入
read_input() {
    local prompt="$1"
    local default="$2"
    local result
    
    if [ -n "$default" ]; then
        read -p "$prompt [$default]: " result
        result="${result:-$default}"
    else
        read -p "$prompt: " result
    fi
    echo "$result"
}

# 首字母大写 (兼容 macOS 和 Linux)
capitalize() {
    local str="$1"
    local first_char=$(echo "${str:0:1}" | tr '[:lower:]' '[:upper:]')
    local rest="${str:1}"
    echo "${first_char}${rest}"
}

# 全部大写
to_upper() {
    echo "$1" | tr '[:lower:]' '[:upper:]'
}

# 替换模板变量 (兼容 macOS 和 Linux)
replace_vars() {
    local file="$1"
    local tmp_file="${file}.tmp"
    
    # 使用 sed 进行变量替换
    sed \
        -e "s|{{SERVICE_NAME}}|$SERVICE_NAME|g" \
        -e "s|{{SERVICE_TITLE}}|$SERVICE_TITLE|g" \
        -e "s|{{SERVICE_NAME_UPPER}}|$SERVICE_NAME_UPPER|g" \
        -e "s|{{MODULE_PATH}}|$MODULE_PATH|g" \
        -e "s|{{GRPC_PORT}}|$GRPC_PORT|g" \
        -e "s|{{HTTP_PORT}}|$HTTP_PORT|g" \
        -e "s|{{DB_NAME}}|$DB_NAME|g" \
        -e "s|{{TABLE_NAME}}|$TABLE_NAME|g" \
        -e "s|{{REDIS_DB}}|$REDIS_DB|g" \
        -e "s|{{CACHE_PREFIX}}|$CACHE_PREFIX|g" \
        -e "s|{{DB_TYPE_DESC}}|$DB_TYPE_DESC|g" \
        -e "s|{{DB_REQUIRE}}|$DB_REQUIRE|g" \
        "$file" > "$tmp_file"
    
    mv "$tmp_file" "$file"
}

# 复制并处理模板文件
copy_template() {
    local src="$1"
    local dest="$2"
    
    mkdir -p "$(dirname "$dest")"
    cp "$src" "$dest"
    replace_vars "$dest"
}

# 显示横幅
show_banner() {
    echo ""
    echo "╔════════════════════════════════════════╗"
    echo "║   uyou API Gateway - 服务生成器        ║"
    echo "╚════════════════════════════════════════╝"
    echo ""
}

# 收集用户输入
collect_input() {
    # 服务名称
    if [ -n "$1" ]; then
        SERVICE_NAME="$1"
    else
        SERVICE_NAME=$(read_input "服务名称 (如 chat, user, order)" "chat")
    fi
    SERVICE_NAME=$(echo "$SERVICE_NAME" | tr '[:upper:]' '[:lower:]')
    SERVICE_TITLE=$(capitalize "$SERVICE_NAME")
    SERVICE_NAME_UPPER=$(to_upper "$SERVICE_NAME")
    
    # Go 模块路径
    local default_module="github.com/yeegeek/uyou-${SERVICE_NAME}-service"
    MODULE_PATH=$(read_input "Go 模块路径" "$default_module")
    
    # gRPC 端口
    GRPC_PORT=$(read_input "gRPC 端口" "50051")
    HTTP_PORT=$((GRPC_PORT + 1000))
    
    # 数据库类型
    echo ""
    echo "选择数据库类型:"
    echo "  1. PostgreSQL (适合强一致性场景: 用户、订单、支付)"
    echo "  2. MongoDB    (适合高吞吐场景: 动态、日志、消息)"
    echo "  3. None       (无数据库, 仅使用 Redis 或外部 API)"
    DB_CHOICE=$(read_input "请选择 [1/2/3]" "1")
    
    case "$DB_CHOICE" in
        1)
            DB_TYPE="postgres"
            DB_TYPE_DESC="PostgreSQL"
            DB_NAME=$(read_input "数据库名称" "${SERVICE_NAME}db")
            TABLE_NAME=$(read_input "主表名称" "${SERVICE_NAME}s")
            DB_REQUIRE='	github.com/lib/pq v1.10.9'
            ;;
        2)
            DB_TYPE="mongodb"
            DB_TYPE_DESC="MongoDB"
            DB_NAME=$(read_input "数据库名称" "${SERVICE_NAME}db")
            TABLE_NAME="${SERVICE_NAME}s"
            DB_REQUIRE='	go.mongodb.org/mongo-driver v1.13.1'
            ;;
        3|*)
            DB_TYPE="none"
            DB_TYPE_DESC="无数据库"
            DB_NAME=""
            TABLE_NAME=""
            DB_REQUIRE=""
            ;;
    esac
    
    # Redis 配置
    REDIS_DB=$(read_input "Redis DB (0-15)" "0")
    CACHE_PREFIX=$(read_input "缓存前缀" "$SERVICE_NAME")
}

# 确认配置
confirm_config() {
    echo ""
    echo "╔════════════════════════════════════════╗"
    echo "║           配置确认                     ║"
    echo "╚════════════════════════════════════════╝"
    echo "服务名称:   $SERVICE_NAME"
    echo "模块路径:   $MODULE_PATH"
    echo "gRPC 端口:  $GRPC_PORT"
    echo "HTTP 端口:  $HTTP_PORT"
    echo "数据库:     $DB_TYPE_DESC"
    
    if [ "$DB_TYPE" != "none" ]; then
        echo "数据库名称: $DB_NAME"
    fi
    if [ "$DB_TYPE" = "postgres" ]; then
        echo "表名称:     $TABLE_NAME"
    fi
    
    echo "Redis DB:   $REDIS_DB"
    echo "缓存前缀:   $CACHE_PREFIX"
    echo ""
    
    local confirm=$(read_input "确认生成? (y/n)" "y")
    [ "$confirm" = "y" ] || [ "$confirm" = "Y" ]
}

# 生成服务
generate_service() {
    local SERVICE_DIR="$PROJECT_ROOT/services/$SERVICE_NAME"
    
    print_info "📁 创建目录: $SERVICE_DIR"
    
    # 创建目录结构
    mkdir -p "$SERVICE_DIR"/{cmd/server,internal/{app,delivery,logic,repository,model},pkg/{conf,logger,errno,middleware,ctxout},api/proto,config}
    
    print_info "📝 生成文件..."
    
    # 复制通用模板
    copy_template "$TEMPLATES_DIR/go.mod.tmpl" "$SERVICE_DIR/go.mod"
    copy_template "$TEMPLATES_DIR/Makefile.tmpl" "$SERVICE_DIR/Makefile"
    copy_template "$TEMPLATES_DIR/Dockerfile.tmpl" "$SERVICE_DIR/Dockerfile"
    copy_template "$TEMPLATES_DIR/.gitignore.tmpl" "$SERVICE_DIR/.gitignore"
    copy_template "$TEMPLATES_DIR/README.md.tmpl" "$SERVICE_DIR/README.md"
    
    # 复制数据库类型特定的 .env 模板
    copy_template "$TEMPLATES_DIR/.env.${DB_TYPE}.tmpl" "$SERVICE_DIR/.env.example"
    
    # 复制根据数据库类型选择的模板
    copy_template "$TEMPLATES_DIR/docker-compose.${DB_TYPE}.yml.tmpl" "$SERVICE_DIR/docker-compose.yml"
    copy_template "$TEMPLATES_DIR/config/config.${DB_TYPE}.yaml.tmpl" "$SERVICE_DIR/config/config.yaml"
    copy_template "$TEMPLATES_DIR/internal/model/model.${DB_TYPE}.go.tmpl" "$SERVICE_DIR/internal/model/${SERVICE_NAME}.go"
    
    # 复制 Proto 模板
    copy_template "$TEMPLATES_DIR/api/proto/service.proto.tmpl" "$SERVICE_DIR/api/proto/${SERVICE_NAME}.proto"
    copy_template "$TEMPLATES_DIR/api/proto/internal.proto.tmpl" "$SERVICE_DIR/api/proto/${SERVICE_NAME}.internal.proto"
    
    # 复制 cmd 模板
    copy_template "$TEMPLATES_DIR/cmd/server/main.go.tmpl" "$SERVICE_DIR/cmd/server/main.go"
    
    # 复制 pkg 模板
    copy_template "$TEMPLATES_DIR/pkg/conf/conf.go.tmpl" "$SERVICE_DIR/pkg/conf/conf.go"
    copy_template "$TEMPLATES_DIR/pkg/logger/logger.go.tmpl" "$SERVICE_DIR/pkg/logger/logger.go"
    copy_template "$TEMPLATES_DIR/pkg/errno/errno.go.tmpl" "$SERVICE_DIR/pkg/errno/errno.go"
    copy_template "$TEMPLATES_DIR/pkg/ctxout/ctxout.go.tmpl" "$SERVICE_DIR/pkg/ctxout/ctxout.go"
    copy_template "$TEMPLATES_DIR/pkg/middleware/interceptors.go.tmpl" "$SERVICE_DIR/pkg/middleware/interceptors.go"
    
    # 复制 internal 模板
    copy_template "$TEMPLATES_DIR/internal/app/app.go.tmpl" "$SERVICE_DIR/internal/app/app.go"
    copy_template "$TEMPLATES_DIR/internal/logic/logic.go.tmpl" "$SERVICE_DIR/internal/logic/logic.go"
    copy_template "$TEMPLATES_DIR/internal/repository/repository.go.tmpl" "$SERVICE_DIR/internal/repository/repository.go"
    copy_template "$TEMPLATES_DIR/internal/delivery/handler.go.tmpl" "$SERVICE_DIR/internal/delivery/handler.go"
    copy_template "$TEMPLATES_DIR/internal/delivery/internal.go.tmpl" "$SERVICE_DIR/internal/delivery/internal.go"
    
    # 复制数据库特定的 repository 实现 (可选使用)
    if [ "$DB_TYPE" = "postgres" ]; then
        copy_template "$TEMPLATES_DIR/internal/repository/postgres.go.tmpl" "$SERVICE_DIR/internal/repository/postgres.go"
        mkdir -p "$SERVICE_DIR/scripts"
        copy_template "$TEMPLATES_DIR/scripts/init-db.postgres.sql.tmpl" "$SERVICE_DIR/scripts/init-db.sql"
    elif [ "$DB_TYPE" = "mongodb" ]; then
        copy_template "$TEMPLATES_DIR/internal/repository/mongodb.go.tmpl" "$SERVICE_DIR/internal/repository/mongodb.go"
    fi
    
    # 生成 Proto 代码
    print_info "🔨 生成 Proto 代码..."
    if command -v protoc &> /dev/null; then
        (cd "$SERVICE_DIR" && make proto 2>/dev/null) || print_warning "⚠️  Proto 代码生成失败，请手动运行 'make proto'"
    else
        print_warning "⚠️  未找到 protoc，请手动运行 'make proto'"
    fi
    
    # 同步依赖
    print_info "📦 同步依赖..."
    if command -v go &> /dev/null; then
        (cd "$SERVICE_DIR" && go mod tidy 2>/dev/null) || print_warning "⚠️  go mod tidy 失败，请手动运行"
    fi
}

# 更新 docker-compose.dev.yml
update_dev_compose() {
    local DEV_COMPOSE="$PROJECT_ROOT/docker-compose.dev.yml"
    
    if [ ! -f "$DEV_COMPOSE" ]; then
        print_warning "⚠️  docker-compose.dev.yml 不存在，跳过更新"
        return
    fi
    
    # 检查服务是否已存在
    if grep -q "  ${SERVICE_NAME}:" "$DEV_COMPOSE"; then
        print_warning "⚠️  服务 $SERVICE_NAME 已存在于 docker-compose.dev.yml"
        return
    fi
    
    # 创建临时服务配置文件
    local TMP_SERVICE=$(mktemp)
    
    # 写入服务配置
    {
        echo ""
        echo "  # ${SERVICE_TITLE} Service"
        echo "  ${SERVICE_NAME}:"
        echo "    build:"
        echo "      context: ./services/${SERVICE_NAME}"
        echo "      dockerfile: Dockerfile"
        echo "    container_name: uyou-${SERVICE_NAME}-dev"
        echo "    environment:"
        
        if [ "$DB_TYPE" = "postgres" ]; then
            echo "      DB_HOST: postgres"
            echo "      DB_PORT: 5432"
            echo "      DB_USER: postgres"
            echo "      DB_PASSWORD: postgres"
            echo "      DB_NAME: ${DB_NAME}"
        elif [ "$DB_TYPE" = "mongodb" ]; then
            echo "      MONGO_HOST: mongodb"
            echo "      MONGO_PORT: 27017"
            echo "      MONGO_USER: root"
            echo "      MONGO_PASSWORD: example"
            echo "      MONGO_DATABASE: ${DB_NAME}"
        fi
        
        echo "      REDIS_HOST: redis"
        echo "      REDIS_PORT: 6379"
        echo "      REDIS_DB: ${REDIS_DB}"
        echo "    ports:"
        echo "      - \"${GRPC_PORT}:${GRPC_PORT}\""
        echo "    depends_on:"
        
        case "$DB_TYPE" in
            postgres)
                echo "      - postgres"
                echo "      - redis"
                ;;
            mongodb)
                echo "      - mongodb"
                echo "      - redis"
                ;;
            *)
                echo "      - redis"
                ;;
        esac
        
        echo "    networks:"
        echo "      - uyou-network"
        echo "    restart: unless-stopped"
    } > "$TMP_SERVICE"

    # 在标记位置插入服务配置
    local MARKER="  # 新生成的微服务将自动添加到此处"
    if grep -q "$MARKER" "$DEV_COMPOSE"; then
        # 使用 sed 在标记前插入内容
        local TMP_COMPOSE=$(mktemp)
        while IFS= read -r line; do
            if [[ "$line" == *"$MARKER"* ]]; then
                cat "$TMP_SERVICE"
            fi
            echo "$line"
        done < "$DEV_COMPOSE" > "$TMP_COMPOSE"
        mv "$TMP_COMPOSE" "$DEV_COMPOSE"
        print_success "✅ 已将服务添加到 docker-compose.dev.yml"
    else
        print_warning "⚠️  未找到标记位置，请手动添加服务到 docker-compose.dev.yml"
    fi
    
    rm -f "$TMP_SERVICE"
}

# 显示后续步骤
print_next_steps() {
    echo ""
    echo "╔════════════════════════════════════════╗"
    echo "║           后续步骤                     ║"
    echo "╚════════════════════════════════════════╝"
    echo "1. 进入服务目录:"
    echo "   cd services/$SERVICE_NAME"
    echo ""
    echo "2. 安装依赖并同步:"
    echo "   go mod tidy"
    echo ""
    echo "3. 生成 gRPC 代码:"
    echo "   make proto"
    echo ""
    echo "4. 实现业务逻辑:"
    echo "   - internal/delivery/  (gRPC 处理器)"
    echo "   - internal/logic/     (业务逻辑)"
    echo "   - internal/repository/ (数据访问 - 默认为内存实现)"
    
    if [ "$DB_TYPE" = "postgres" ]; then
        echo ""
        echo "5. [可选] 使用真实数据库:"
        echo "   - 运行 scripts/init-db.sql 创建表结构"
        echo "   - 修改 internal/repository/repository.go 使用 postgres.go 中的实现"
    elif [ "$DB_TYPE" = "mongodb" ]; then
        echo ""
        echo "5. [可选] 使用真实数据库:"
        echo "   - 修改 internal/repository/repository.go 使用 mongodb.go 中的实现"
    fi
    
    echo ""
    echo "6. 本地开发:"
    echo "   cd ../../"
    echo "   make start dev  # 启动所有开发环境服务"
    echo ""
    echo "7. 同步 APISIX 路由 (自动根据 proto 文件生成):"
    echo "   make update-routes"
    echo ""
    echo "8. 测试 API:"
    echo "   # 创建"
    echo "   curl -X POST http://localhost:9080/api/v1/${SERVICE_NAME}s -H 'Content-Type: application/json' -d '{\"name\":\"test\"}'"
    echo "   # 查询列表"
    echo "   curl http://localhost:9080/api/v1/${SERVICE_NAME}s"
    echo "   # 获取单个"
    echo "   curl http://localhost:9080/api/v1/${SERVICE_NAME}s/1"
    echo ""
    echo "📖 详细文档: services/${SERVICE_NAME}/README.md"
}

# 主函数
main() {
    show_banner
    collect_input "$1"
    
    if ! confirm_config; then
        print_error "❌ 已取消"
        exit 1
    fi
    
    generate_service
    update_dev_compose
    
    echo ""
    print_success "✅ 服务生成成功！"
    print_next_steps
}

# 运行主函数
main "$@"
