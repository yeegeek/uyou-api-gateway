#!/bin/bash

# 合并多个微服务的 APISIX 配置并推送到 etcd
# 适用于微服务分开开发的场景
#
# 工作流程:
# 1. 从 services/<service-name>/proto/ 目录读取 proto 文件并创建 Proto 定义
# 2. 从 apisix/config/routes/ 目录读取各微服务的路由配置
# 3. 将所有路由配置合并并部署到 APISIX (etcd)

set -e

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 配置
GATEWAY_URL="${APISIX_ADMIN_URL:-http://localhost:9180}"
ADMIN_KEY="${APISIX_ADMIN_KEY:-edd1c9f034335f136f87ad84b625c8f1}"
ENV="${APISIX_ENV:-dev}"
JWT_SECRET="${APISIX_JWT_SECRET:-your-secret-key-change-in-production}"

# 获取脚本所在目录
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"

# 配置目录（支持从环境变量指定）
CONFIG_DIR="${APISIX_CONFIG_DIR:-${PROJECT_ROOT}/apisix/config}"
ROUTES_DIR="${APISIX_ROUTES_DIR:-${CONFIG_DIR}/routes}"

echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}APISIX 配置合并与部署工具${NC}"
echo -e "${GREEN}========================================${NC}"
echo "环境: ${ENV}"
echo "APISIX Admin: ${GATEWAY_URL}"
echo "配置目录: ${CONFIG_DIR}"
echo ""

# 检查依赖
check_dependencies() {
    local missing=()
    local yaml_parser_available=false
    
    if ! command -v curl &> /dev/null; then
        missing+=("curl")
    fi
    
    # 检查 YAML 解析工具
    if command -v yq &> /dev/null; then
        yaml_parser_available=true
        echo -e "${GREEN}✓ 找到 yq 工具${NC}"
    elif command -v python3 &> /dev/null; then
        # 检查 Python 是否有 yaml 模块
        if python3 -c "import yaml" 2>/dev/null; then
            yaml_parser_available=true
            echo -e "${GREEN}✓ 找到 Python3 和 PyYAML 模块${NC}"
        else
            echo -e "${YELLOW}⚠ Python3 已安装，但缺少 PyYAML 模块${NC}"
            echo -e "${YELLOW}   安装方法: pip3 install pyyaml${NC}"
        fi
    fi
    
    if [ "$yaml_parser_available" = false ]; then
        echo -e "${RED}错误: 缺少 YAML 解析工具${NC}"
        echo ""
        echo "请安装以下工具之一："
        echo "  1. yq (推荐):"
        echo "     macOS:   brew install yq"
        echo "     Linux:   见 https://github.com/mikefarah/yq#install"
        echo ""
        echo "  2. Python3 + PyYAML:"
        echo "     pip3 install pyyaml"
        echo ""
        exit 1
    fi
    
    if [ ${#missing[@]} -ne 0 ]; then
        echo -e "${RED}错误: 缺少依赖: ${missing[*]}${NC}"
        exit 1
    fi
}

# 等待 APISIX 就绪
wait_for_apisix() {
    echo -e "${YELLOW}等待 APISIX 服务就绪...${NC}"
    for i in {1..60}; do
        if curl -s -f "${GATEWAY_URL}/apisix/admin/routes" \
           -H "X-API-KEY: ${ADMIN_KEY}" > /dev/null 2>&1; then
            echo -e "${GREEN}✓ APISIX 已就绪${NC}"
            return 0
        fi
        if [ $i -eq 60 ]; then
            echo -e "${RED}错误: APISIX 服务未就绪${NC}"
            exit 1
        fi
        sleep 2
    done
}

# 创建 Proto 定义（在 APISIX 中注册）
# 
# 注意：这与 make proto 不同！
# - make proto: 编译时生成 Go 代码（.pb.go 文件）
# - create_proto: 运行时将 proto 文件内容上传到 APISIX，用于 REST to gRPC 转码
#
# APISIX 需要 proto 文件来：
# 1. 将客户端的 JSON 请求转换为 gRPC 格式
# 2. 将微服务的 gRPC 响应转换为 JSON 格式
#
# 参数:
#   $1: proto_id - APISIX 中 proto 定义的唯一 ID（如 "1", "2", "3"）
#   $2: proto_file - proto 文件的路径
create_proto() {
    local id=$1
    local proto_file=$2
    
    if [ ! -f "$proto_file" ]; then
        echo -e "${YELLOW}⚠ 跳过不存在的 proto 文件: ${proto_file}${NC}"
        return 0
    fi
    
    echo "创建 Proto ${id}..."
    
    local temp_json=$(mktemp)
    # 使用 python3 构建 JSON（不依赖 jq）
    python3 <<PYTHON_SCRIPT > "$temp_json"
import json
with open("$proto_file", "r") as f:
    content = f.read()
print(json.dumps({"id": "$id", "content": content}))
PYTHON_SCRIPT
    
    local response=$(curl -s -X PUT "${GATEWAY_URL}/apisix/admin/protos/${id}" \
        -H "X-API-KEY: ${ADMIN_KEY}" \
        -H "Content-Type: application/json" \
        -d "@${temp_json}")
    
    rm -f "$temp_json"
    
    if echo "$response" | grep -q '"key"'; then
        echo -e "  ${GREEN}✓ Proto ${id} 创建成功${NC}"
    else
        if echo "$response" | grep -q "already exists\|duplicate"; then
            echo -e "  ${GREEN}✓ Proto ${id} 已存在${NC}"
        else
            echo -e "  ${YELLOW}⚠ Proto ${id} 创建失败（非致命）: $response${NC}"
        fi
    fi
}

# 从 JSON 提取字段值（不依赖 jq）
extract_json_field() {
    local json=$1
    local field=$2
    # 使用 yq 解析 JSON（yq 支持 JSON 输入）
    if command -v yq &> /dev/null; then
        echo "$json" | yq eval -p=json ".$field // \"\"" -
    elif command -v python3 &> /dev/null; then
        echo "$json" | python3 -c "import json,sys; d=json.load(sys.stdin); print(d.get('$field',''))"
    else
        # 简单的 grep 提取（fallback）
        echo "$json" | grep -oP "\"$field\"\\s*:\\s*\"\\K[^\"]*" | head -1
    fi
}

# 创建 JWT Consumer
# 在 APISIX 中创建 JWT Consumer，用于 JWT 认证
#
# consumer_key 说明：
# - 这是 APISIX Consumer 的标识符，用于匹配 JWT token payload 中的 "key" 字段
# - 不是传递给微服务的 key，而是 APISIX 内部用于验证 token 的标识符
# - 微服务生成 JWT token 时，payload 必须包含 "key": "user_key" 字段
# - 微服务从 gRPC metadata 中获取用户信息（X-Consumer-Username 或解析 JWT token）
#
# 详细说明：参见 docs/JWT-AUTH-FLOW.md
create_jwt_consumer() {
    local consumer_key="user_key"
    local secret="${JWT_SECRET}"
    
    echo "创建 JWT Consumer: ${consumer_key}"
    
    local consumer_config=$(cat <<EOF
{
  "username": "${consumer_key}",
  "plugins": {
    "jwt-auth": {
      "key": "${consumer_key}",
      "secret": "${secret}",
      "algorithm": "HS256"
    }
  }
}
EOF
)
    
    local response=$(curl -s -X PUT "${GATEWAY_URL}/apisix/admin/consumers/${consumer_key}" \
        -H "X-API-KEY: ${ADMIN_KEY}" \
        -H "Content-Type: application/json" \
        -d "${consumer_config}")
    
    if echo "$response" | grep -q '"key"'; then
        echo -e "  ${GREEN}✓ JWT Consumer 创建成功${NC}"
        return 0
    else
        if echo "$response" | grep -q "already exists\|duplicate"; then
            echo -e "  ${GREEN}✓ JWT Consumer 已存在${NC}"
            return 0
        else
            echo -e "  ${YELLOW}⚠ JWT Consumer 创建失败（非致命）: $response${NC}"
            return 1
        fi
    fi
}

# 创建路由
create_route() {
    local route_json=$1
    local route_name=$(extract_json_field "$route_json" "name")
    
    if [ -z "$route_name" ] || [ "$route_name" = "null" ]; then
        echo -e "${RED}错误: 路由缺少 name 字段${NC}"
        return 1
    fi
    
    echo "创建路由: ${route_name}"
    
    # 使用 PUT 方法会完全替换路由配置，确保 jwt-auth 插件正确应用
    local response=$(curl -s -X PUT "${GATEWAY_URL}/apisix/admin/routes/${route_name}" \
        -H "X-API-KEY: ${ADMIN_KEY}" \
        -H "Content-Type: application/json" \
        -d "$route_json")
    
    if echo "$response" | grep -q '"key"'; then
        echo -e "  ${GREEN}✓ 成功${NC}"
        return 0
    else
        echo -e "  ${RED}✗ 失败: $response${NC}"
        return 1
    fi
}

# 从 YAML 文件读取路由配置
read_routes_from_yaml() {
    local yaml_file=$1
    
    if [ ! -f "$yaml_file" ]; then
        echo -e "${YELLOW}⚠ 配置文件不存在: ${yaml_file}${NC}"
        return 0
    fi
    
    # 检查是否有 yq 命令（优先使用）
    if command -v yq &> /dev/null; then
        # 获取路由数量
        local route_count=$(yq eval '.routes | length' "$yaml_file" 2>/dev/null)
        if [ -z "$route_count" ] || [ "$route_count" = "0" ] || [ "$route_count" = "null" ]; then
            return 0
        fi
        
        # 逐个输出每条路由的 JSON（紧凑格式，单行）
        for ((i=0; i<route_count; i++)); do
            # 使用 python3 将多行 JSON 压缩为单行
            yq eval -o=json ".routes[$i]" "$yaml_file" | python3 -c "import json,sys; print(json.dumps(json.load(sys.stdin), separators=(',',':')))" 2>/dev/null || \
            yq eval -o=json ".routes[$i]" "$yaml_file" | tr -d '\n' | sed 's/  */ /g'
        done
    elif command -v python3 &> /dev/null; then
        # 检查是否有 yaml 模块
        if python3 -c "import yaml" 2>/dev/null; then
            # 使用 Python 解析 YAML
            python3 <<PYTHON_SCRIPT
import yaml
import json
import sys

try:
    with open("$yaml_file", "r") as f:
        data = yaml.safe_load(f)
        routes = data.get("routes", [])
        for route in routes:
            # 输出紧凑格式（单行）JSON
            print(json.dumps(route, separators=(',',':'), ensure_ascii=False))
except Exception as e:
    print(f"Error: {e}", file=sys.stderr)
    sys.exit(1)
PYTHON_SCRIPT
        else
            echo -e "${RED}错误: Python3 缺少 PyYAML 模块${NC}" >&2
            echo -e "${YELLOW}安装方法: pip3 install pyyaml${NC}" >&2
            return 1
        fi
    else
        echo -e "${RED}错误: 需要 yq 或 python3 (含 PyYAML) 来解析 YAML${NC}" >&2
        return 1
    fi
}

# 主函数
main() {
    check_dependencies
    wait_for_apisix
    
    # 0. 创建 JWT Consumer（如果启用 JWT 认证）
    echo -e "\n${GREEN}步骤 0: 创建 JWT Consumer${NC}"
    create_jwt_consumer
    
    # 1. 创建 Proto 定义
    echo -e "\n${GREEN}步骤 1: 创建 Proto 定义${NC}"
    SERVICES_DIR="${PROJECT_ROOT}/services"
    
    # 获取服务对应的 proto_id（固定映射，不受服务发现顺序影响）
    # 这确保即使某些 proto 文件缺失，proto_id 也始终对应正确的服务
    get_proto_id() {
        case "$1" in
            user) echo "1" ;;
            order) echo "2" ;;
            feed) echo "3" ;;
            *) echo "" ;;
        esac
    }
    
    # 从 services/ 目录下的各个微服务中查找 proto 文件
    if [ -d "$SERVICES_DIR" ]; then
        # 按照固定顺序处理所有服务
        for service_name in user order feed; do
            local proto_id=$(get_proto_id "$service_name")
            if [ -z "$proto_id" ]; then
                continue
            fi
            
            local service_dir="${SERVICES_DIR}/${service_name}"
            local proto_file="${service_dir}/api/proto/${service_name}.proto"
            
            if [ -f "$proto_file" ]; then
                echo "找到 ${service_name} 服务的 proto 文件: ${proto_file}"
                create_proto "${proto_id}" "$proto_file"
            else
                # 兼容旧路径
                local old_proto_file="${service_dir}/proto/${service_name}.proto"
                if [ -f "$old_proto_file" ]; then
                    echo "找到 ${service_name} 服务的 proto 文件 (旧路径): ${old_proto_file}"
                    create_proto "${proto_id}" "$old_proto_file"
                else
                    # 尝试查找该服务目录下的任何 proto 文件
                    if [ -d "${service_dir}/api/proto" ]; then
                        local first_proto=$(find "${service_dir}/api/proto" -maxdepth 1 -name "*.proto" | head -1)
                        if [ -n "$first_proto" ]; then
                            echo "找到 ${service_name} 服务的 proto 文件: ${first_proto}"
                            create_proto "${proto_id}" "$first_proto"
                        fi
                    elif [ -d "${service_dir}/proto" ]; then
                        local first_proto=$(find "${service_dir}/proto" -maxdepth 1 -name "*.proto" | head -1)
                        if [ -n "$first_proto" ]; then
                            echo "找到 ${service_name} 服务的 proto 文件 (旧路径): ${first_proto}"
                            create_proto "${proto_id}" "$first_proto"
                        fi
                    else
                        echo -e "${YELLOW}⚠ ${service_name} 服务的 proto 文件未找到 (期望 proto_id=${proto_id})${NC}"
                    fi
                fi
            fi
        done
    else
        echo -e "${YELLOW}⚠ Services 目录不存在: ${SERVICES_DIR}${NC}"
        echo -e "${YELLOW}   提示: 如果 proto 文件在其他位置，请手动创建 proto 定义${NC}"
    fi
    
    # 2. 读取并创建路由
    echo -e "\n${GREEN}步骤 2: 创建路由配置${NC}"
    
    local route_count=0
    local success_count=0
    
    # 方式 1: 从 routes/ 目录读取多个配置文件
    if [ -d "$ROUTES_DIR" ]; then
        echo "从 ${ROUTES_DIR} 读取路由配置..."
        for route_file in "$ROUTES_DIR"/*.yaml "$ROUTES_DIR"/*.yml; do
            if [ -f "$route_file" ]; then
                echo "处理: $(basename $route_file)"
                while IFS= read -r route_json; do
                    if [ -n "$route_json" ]; then
                        create_route "$route_json" && ((success_count++))
                        ((route_count++))
                    fi
                done < <(read_routes_from_yaml "$route_file")
            fi
        done
    fi
    
    # 方式 2: 从主配置文件读取（已废弃，迁往 routes/ 目录）
    # if [ -f "${CONFIG_DIR}/apisix.yaml" ]; then
    #     echo "从主配置文件读取路由..."
    #     while IFS= read -r route_json; do
    #         if [ -n "$route_json" ]; then
    #             create_route "$route_json" && ((success_count++))
    #             ((route_count++))
    #         fi
    #     done < <(read_routes_from_yaml "${CONFIG_DIR}/apisix.yaml")
    # fi
    
    # 3. 总结
    echo -e "\n${GREEN}========================================${NC}"
    echo -e "${GREEN}部署完成！${NC}"
    echo "总路由数: ${route_count}"
    echo "成功: ${success_count}"
    echo "失败: $((route_count - success_count))"
    echo -e "${GREEN}========================================${NC}"
}

# 运行主函数
main "$@"
