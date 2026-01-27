#!/bin/bash

# APISIX 配置合并与部署工具
#
# 功能：
# 1. 自动扫描 services/ 目录下的 proto 文件
# 2. 动态分配或保留 proto_id
# 3. 自动生成缺失的路由配置文件
# 4. 同步全局配置 (global.yaml) 到 APISIX
# 5. 合并所有路由并推送到 APISIX
# 6. 导出合并后的配置到 apisix.yaml 供审计

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# 获取项目根目录
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"

# 配置变量
GATEWAY_URL="${APISIX_ADMIN_URL:-http://localhost:9180}"
ADMIN_KEY="${APISIX_ADMIN_KEY:-edd1c9f034335f136f87ad84b625c8f1}"
CONFIG_DIR="${APISIX_CONFIG_DIR:-${PROJECT_ROOT}/apisix/config}"
ROUTES_DIR="${APISIX_ROUTES_DIR:-${CONFIG_DIR}/routes}"
PROTOS_ARCHIVE_DIR="${APISIX_PROTOS_DIR:-${CONFIG_DIR}/protos}"

# 是否仅部署现有的配置（不从 services 重新生成）
DEPLOY_ONLY=false
if [[ "$1" == "--deploy-only" ]]; then
    DEPLOY_ONLY=true
fi

echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}APISIX 配置合并与部署工具${NC}"
echo -e "${BLUE}========================================${NC}"
echo "环境: ${ENV:-dev}"
echo "APISIX Admin: ${GATEWAY_URL}"
echo "配置目录: ${CONFIG_DIR}"
echo ""

# 检查依赖
check_dependencies() {
    if ! command -v curl &> /dev/null; then
        echo -e "${RED}错误: 未找到 curl 工具${NC}"
        exit 1
    fi
    
    if ! command -v yq &> /dev/null; then
        echo -e "${RED}错误: 未找到 yq 工具。请安装 yq (brew install yq)${NC}"
        exit 1
    fi
    echo -e "${GREEN}✓ 找到 yq 工具${NC}"
}

# 等待 APISIX 就绪
wait_for_apisix() {
    echo "等待 APISIX 服务就绪..."
    local max_retries=30
    local count=0
    while ! curl -s "${GATEWAY_URL}/apisix/admin/routes" -H "X-API-KEY: ${ADMIN_KEY}" > /dev/null; do
        sleep 1
        ((count++))
        if [ $count -ge $max_retries ]; then
            echo -e "${RED}错误: APISIX 服务未在 ${max_retries}s 内就绪${NC}"
            exit 1
        fi
    done
    echo -e "${GREEN}✓ APISIX 已就绪${NC}"
}

# 创建 Proto 定义
create_proto() {
    local id=$1
    local proto_path=$2
    
    local content=$(cat "$proto_path")
    # 构造请求体
    local payload=$(python3 -c "import json, sys; print(json.dumps({'content': sys.stdin.read()}))" <<EOF
${content}
EOF
)

    local response=$(curl -s -X PUT "${GATEWAY_URL}/apisix/admin/protos/${id}" \
        -H "X-API-KEY: ${ADMIN_KEY}" \
        -H "Content-Type: application/json" \
        -d "${payload}")
    
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

# 同步全局配置 (通过 global.yaml)
sync_global_config() {
    local global_yaml="${CONFIG_DIR}/global.yaml"
    
    if [ ! -f "$global_yaml" ]; then
        echo -e "${YELLOW}⚠ 未找到 global.yaml，跳过全局配置同步${NC}"
        return 0
    fi
    
    echo "同步全局配置 (${global_yaml})..."
    
    # 同步 Global Rules
    local rule_ids=$(yq eval '.global_rules[].id' "$global_yaml" 2>/dev/null)
    if [ -n "$rule_ids" ] && [ "$rule_ids" != "null" ]; then
        for rid in $rule_ids; do
            local rule_json=$(yq eval ".global_rules[] | select(.id == $rid) | del(.id)" -o=json "$global_yaml")
            curl -s -X PUT "${GATEWAY_URL}/apisix/admin/global_rules/${rid}" \
                -H "X-API-KEY: ${ADMIN_KEY}" -H "Content-Type: application/json" -d "${rule_json}" > /dev/null
            echo -e "  ${GREEN}✓ global_rules/${rid} 同步成功${NC}"
        done
    fi
    
    # 同步 Consumers（以 Consumer.plugins.jwt-auth 的方式配置，适配 APISIX 3.8）
    local consumer_names=$(yq eval '.consumers[].username' "$global_yaml" 2>/dev/null)
    if [ -n "$consumer_names" ] && [ "$consumer_names" != "null" ]; then
        # 从环境变量读取 JWT_SECRET，如果不存在则使用配置文件中的值
        local jwt_secret="${JWT_SECRET:-$(yq eval '.consumers[0].plugins."jwt-auth".secret' "$global_yaml" 2>/dev/null)}"
        if [ -z "$jwt_secret" ] || [ "$jwt_secret" = "null" ]; then
            jwt_secret="uyou_secret_key_2026"  # 默认值
        fi

        for cname in $consumer_names; do
            # 取出 consumer 完整定义（包含 username 与 plugins）
            local consumer_json=$(yq eval ".consumers[] | select(.username == \"$cname\")" -o=json "$global_yaml" 2>/dev/null)
            [ -z "$consumer_json" ] || [ "$consumer_json" = "null" ] && continue

            # 如果 consumer 有 jwt-auth 插件，更新 secret（从环境变量注入）
            if echo "$consumer_json" | grep -q "\"jwt-auth\""; then
                consumer_json=$(echo "$consumer_json" | python3 -c "
import json, sys
data = json.load(sys.stdin)
if 'plugins' in data and 'jwt-auth' in data['plugins']:
    data['plugins']['jwt-auth']['secret'] = sys.argv[1]
print(json.dumps(data))
" "$jwt_secret")
            fi

            curl -s -X PUT "${GATEWAY_URL}/apisix/admin/consumers/${cname}" \
                -H "X-API-KEY: ${ADMIN_KEY}" -H "Content-Type: application/json" -d "${consumer_json}" > /dev/null
            echo -e "  ${GREEN}✓ consumers/${cname} 同步成功${NC}"
        done
    fi
}

# 提取现有配置文件中的 proto_id 映射
extract_existing_proto_ids() {
    if [ ! -d "$ROUTES_DIR" ]; then return; fi
    for f in "$ROUTES_DIR"/*.yaml; do
        [ -e "$f" ] || continue
        local pid=$(yq eval '.routes[0].plugins.grpc-transcode.proto_id' "$f" 2>/dev/null)
        if [ -n "$pid" ] && [ "$pid" != "null" ]; then
            local sname=$(basename "$f" | sed 's/-routes.yaml//; s/.yaml//; s/.yml//')
            echo "$sname:$pid"
        fi
    done
}

# 生成默认路由配置文件
generate_default_routes() {
    local service_name=$1
    local proto_id=$2
    local proto_file=$3
    local output_file=$4
    
    echo "  → 为 ${service_name} 生成全量路由配置: ${output_file}"
    mkdir -p "$(dirname "$output_file")"
    
    local rpc_methods=$(python3 <<PYTHON_SCRIPT
import re
import sys

methods = []
try:
    with open("$proto_file", "r") as f:
        content = f.read()
        pattern = re.compile(r'((?://.*?\n\s*)*)rpc\s+(\w+)\s*\(', re.MULTILINE)
        for match in pattern.finditer(content):
            comment_block = match.group(1)
            method_name = match.group(2)
            is_auth = "true" if "@auth" in comment_block else "false"
            rate, burst = "0", "0"
            limit_match = re.search(r'@limit\(rate=(\d+),\s*burst=(\d+)\)', comment_block)
            if limit_match:
                rate, burst = limit_match.group(1), limit_match.group(2)
            methods.append(f"{method_name}|{is_auth}|{rate}|{burst}")
    print("\n".join(methods))
except Exception as e:
    sys.exit(1)
PYTHON_SCRIPT
)

    if [ -z "$rpc_methods" ]; then rpc_methods="List|false|0|0 Create|false|0|0 Get|false|0|0"; fi

    cat > "$output_file" <<EOF
# 自动生成的路由配置 - $(date)
# 源文件: ${proto_file}
routes:
EOF

    local service_class=$(python3 -c "import sys; print(sys.argv[1].capitalize())" "${service_name}")Service

    for entry in $rpc_methods; do
        local method_name=$(echo "$entry" | cut -d'|' -f1)
        local is_auth=$(echo "$entry" | cut -d'|' -f2)
        local rate=$(echo "$entry" | cut -d'|' -f3)
        local burst=$(echo "$entry" | cut -d'|' -f4)
        
        local http_method="POST"
        local uri_suffix="/${method_name}"
        local uri_base="/api/v1/${service_name}s"
        
        # 是否需要路径参数 (Get/Update/Delete 需要 :id)
        local needs_path_param="false"
        case "$method_name" in
            Get*)    http_method="GET";    uri_suffix="/:id"; needs_path_param="true" ;;
            List*)   http_method="GET";    uri_suffix=""   ;;
            Create*) http_method="POST";   uri_suffix=""   ;;
            Update*) http_method="PUT";    uri_suffix="/:id"; needs_path_param="true" ;;
            Delete*) http_method="DELETE"; uri_suffix="/:id"; needs_path_param="true" ;;
        esac
        
        local full_uri=$(echo "${uri_base}${uri_suffix}" | sed 's/\/$//')
        local method_name_lower=$(echo "$method_name" | tr '[:upper:]' '[:lower:]')
        
        cat >> "$output_file" <<EOF
  - id: "${service_name}_${method_name_lower}"
    name: "${service_name}_${method_name_lower}"
    uri: ${full_uri}
    methods: ["${http_method}"]
    plugins:
EOF
        [ "$is_auth" = "true" ] && echo "      jwt-auth: {}" >> "$output_file"
        if [ "$rate" != "0" ]; then
            cat >> "$output_file" <<EOF
      limit-req:
        rate: ${rate}
        burst: ${burst}
        key: "remote_addr"
        rejected_code: 429
EOF
        fi
        # 为需要路径参数的路由添加 serverless-pre-function
        if [ "$needs_path_param" = "true" ]; then
            cat >> "$output_file" <<'SERVERLESS_EOF'
      serverless-pre-function:
        phase: rewrite
        functions:
          - "return function(conf, ctx) local id = ctx.curr_req_matched and ctx.curr_req_matched.id; if id then ngx.req.set_uri_args({id = id}) end end"
SERVERLESS_EOF
        fi
        cat >> "$output_file" <<EOF
      grpc-transcode:
        proto_id: "${proto_id}"
        service: "${service_name}.${service_class}"
        method: "${method_name}"
    upstream:
      nodes:
        "${service_name}:50051": 1
      type: roundrobin
      scheme: grpc
EOF
    done
}

# 创建路由
create_route() {
    local json_str=$1
    local route_id=$(echo "$json_str" | python3 -c "import json, sys; d=json.load(sys.stdin); print(d.get('id', ''))")
    
    [ -z "$route_id" ] && return 1
    
    local response=$(curl -s -X PUT "${GATEWAY_URL}/apisix/admin/routes/${route_id}" \
        -H "X-API-KEY: ${ADMIN_KEY}" \
        -H "Content-Type: application/json" \
        -d "${json_str}")
    
    if echo "$response" | grep -q '"key"'; then
        echo -e "创建路由: ${route_id}\n  ${GREEN}✓ 成功${NC}"
        return 0
    else
        echo -e "创建路由: ${route_id}\n  ${RED}✗ 失败: $response${NC}"
        return 1
    fi
}

# 主程序逻辑
main() {
    check_dependencies
    wait_for_apisix
    
    echo -e "\n${GREEN}步骤 0: 同步全局配置${NC}"
    sync_global_config
    
    echo -e "\n${GREEN}步骤 1: 创建 Proto 定义${NC}"
    SERVICES_DIR="${PROJECT_ROOT}/services"
    mkdir -p "$PROTOS_ARCHIVE_DIR"
    
    local existing_mappings=$(extract_existing_proto_ids)
    local max_id=0
    for mapping in $existing_mappings; do
        local pid=$(echo "$mapping" | cut -d: -f2)
        if [[ "$pid" =~ ^[0-9]+$ ]] && [ "$pid" -gt "$max_id" ]; then max_id=$pid; fi
    done
    
    get_proto_id() {
        local sname=$1
        for mapping in $existing_mappings; do
            if [[ "$mapping" == "$sname:"* ]]; then echo "$mapping" | cut -d: -f2; return; fi
        done
        max_id=$((max_id + 1)); echo "$max_id"
    }
    
    if [ "$DEPLOY_ONLY" = true ]; then
        for pfile in "$PROTOS_ARCHIVE_DIR"/*.proto; do
            [ -e "$pfile" ] || continue
            local sname=$(basename "$pfile" .proto)
            create_proto "$(get_proto_id "$sname")" "$pfile"
        done
    else
        [ -d "$SERVICES_DIR" ] && for d in "$SERVICES_DIR"/*; do
            [ -d "$d" ] || continue
            local sname=$(basename "$d")
            local pfile=$(find "$d" -name "*.proto" -not -name "*.internal.proto" | grep -E "/${sname}\.proto$" | head -1)
            [ -z "$pfile" ] && pfile=$(find "$d" -name "*.proto" -not -name "*.internal.proto" | head -1)
            if [ -n "$pfile" ]; then
                local pid=$(get_proto_id "$sname")
                echo "找到 ${sname} 的 proto: ${pfile} (ID: ${pid})"
                cp "$pfile" "${PROTOS_ARCHIVE_DIR}/${sname}.proto"
                create_proto "$pid" "$pfile"
                generate_default_routes "$sname" "$pid" "$pfile" "${ROUTES_DIR}/${sname}-routes.yaml"
            fi
        done
    fi

    echo -e "\n${GREEN}步骤 2: 创建路由配置${NC}"
    local total=0 success=0 failed=0
    local temp_json=$(mktemp)
    echo "[]" > "$temp_json"

    for rfile in "${ROUTES_DIR}"/*.yaml; do
        [ -e "$rfile" ] || continue
        echo "处理: $(basename "$rfile")"
        while read -r rjson; do
            [ -z "$rjson" ] || [ "$rjson" = "null" ] && continue
            if create_route "$rjson"; then
                ((success++))
            else
                ((failed++))
            fi
            ((total++))
            # 合并到临时文件用于导出
            local current_all=$(cat "$temp_json")
            python3 -c "import json, sys; a=json.loads(sys.argv[1]); r=json.loads(sys.argv[2]); a.append(r); print(json.dumps(a))" "$current_all" "$rjson" > "${temp_json}.new"
            mv "${temp_json}.new" "$temp_json"
        done < <(yq eval '.routes[]' -o=json --indent 0 "$rfile")
    done

    echo -e "\n${YELLOW}导出合并后的配置至 apisix.yaml...${NC}"
    echo "# 自动生成的 APISIX 配置文件" > "${CONFIG_DIR}/apisix.yaml"
    echo "# generated_at: $(date)" >> "${CONFIG_DIR}/apisix.yaml"
    echo "" >> "${CONFIG_DIR}/apisix.yaml"
    yq eval-all -P '. as $item ireduce ({"routes": []}; .routes += $item.routes)' "${ROUTES_DIR}"/*.yaml >> "${CONFIG_DIR}/apisix.yaml"
    echo -e "  ${GREEN}✓ 导出成功${NC}"
    rm -f "$temp_json"

    echo -e "\n========================================\n部署完成！\n总路由数: $total\n成功: $success\n失败: $failed\n========================================\n"
}

main "$@"
