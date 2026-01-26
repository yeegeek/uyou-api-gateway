#!/bin/bash

# 验证 APISIX 路由配置
# 检查项：
# 1. YAML 语法正确性
# 2. 路由名称唯一性
# 3. URI 路径冲突
# 4. 必需字段完整性

set -e

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# 获取脚本所在目录
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"

# 配置目录
ROUTES_DIR="${ROUTES_DIR:-${PROJECT_ROOT}/apisix/config/routes}"
CONFIG_DIR="${CONFIG_DIR:-${PROJECT_ROOT}/apisix/config}"

echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}APISIX 配置验证${NC}"
echo -e "${GREEN}========================================${NC}"
echo "路由目录: ${ROUTES_DIR}"
echo ""

# 错误计数
ERRORS=0
WARNINGS=0

# 检查依赖
check_dependencies() {
    local missing=()
    local yaml_parser_available=false
    
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

# 验证 YAML 语法
validate_yaml_syntax() {
    local file=$1
    local filename=$(basename "$file")
    
    echo -e "${BLUE}验证 YAML 语法: ${filename}${NC}"
    
    # 优先使用 yq
    if command -v yq &> /dev/null; then
        if yq eval '.' "$file" > /dev/null 2>&1; then
            echo -e "${GREEN}  ✓ YAML 语法正确${NC}"
            return 0
        else
            echo -e "${RED}  ✗ YAML 语法错误${NC}"
            yq eval '.' "$file" 2>&1 | head -5
            ((ERRORS++))
            return 1
        fi
    elif command -v python3 &> /dev/null && python3 -c "import yaml" 2>/dev/null; then
        if python3 -c "import yaml; yaml.safe_load(open('$file'))" 2>/dev/null; then
            echo -e "${GREEN}  ✓ YAML 语法正确${NC}"
            return 0
        else
            echo -e "${RED}  ✗ YAML 语法错误${NC}"
            python3 -c "import yaml; yaml.safe_load(open('$file'))" 2>&1 | head -5
            ((ERRORS++))
            return 1
        fi
    else
        echo -e "${YELLOW}  ⚠ 无法验证 YAML 语法（缺少 yq 或 PyYAML）${NC}"
        ((WARNINGS++))
        return 0
    fi
}

# 提取路由信息
extract_routes() {
    local file=$1
    
    # 优先使用 yq
    if command -v yq &> /dev/null; then
        yq eval '.routes[] | [.name // "unnamed", .uri // "", (.methods // [] | join(","))] | join("|")' "$file" 2>/dev/null
    elif command -v python3 &> /dev/null && python3 -c "import yaml" 2>/dev/null; then
        python3 <<PYTHON_SCRIPT
import yaml
import sys

try:
    with open("$file", "r") as f:
        data = yaml.safe_load(f)
    
    routes = data.get("routes", [])
    for route in routes:
        name = route.get("name", "unnamed")
        uri = route.get("uri", "")
        methods = route.get("methods", [])
        print(f"{name}|{uri}|{','.join(methods)}")
except Exception as e:
    print(f"ERROR: {e}", file=sys.stderr)
    sys.exit(1)
PYTHON_SCRIPT
    else
        echo "ERROR: No YAML parser available" >&2
        return 1
    fi
}

# 验证路由唯一性
validate_route_uniqueness() {
    echo -e "${BLUE}验证路由名称唯一性...${NC}"
    
    local route_names=()
    local duplicate_names=()
    
    # 收集所有路由名称
    # 处理 routes 目录下的 yaml 文件
    shopt -s nullglob  # 如果 glob 没有匹配，返回空而不是字面量
    for route_file in "${ROUTES_DIR}"/*.yaml; do
        if [ ! -f "$route_file" ]; then
            continue
        fi
        
        while IFS='|' read -r name uri methods; do
            if [ -n "$name" ] && [ "$name" != "unnamed" ]; then
                if [[ " ${route_names[@]} " =~ " ${name} " ]]; then
                    duplicate_names+=("$name")
                else
                    route_names+=("$name")
                fi
            fi
        done < <(extract_routes "$route_file" 2>/dev/null)
    done
    shopt -u nullglob  # 恢复默认行为
    
    # 处理主配置文件（如果存在）
    if [ -f "${CONFIG_DIR}/apisix.yaml" ]; then
        route_file="${CONFIG_DIR}/apisix.yaml"
        
        while IFS='|' read -r name uri methods; do
            if [ -n "$name" ] && [ "$name" != "unnamed" ]; then
                if [[ " ${route_names[@]} " =~ " ${name} " ]]; then
                    duplicate_names+=("$name")
                else
                    route_names+=("$name")
                fi
            fi
        done < <(extract_routes "$route_file" 2>/dev/null)
    fi
    
    if [ ${#duplicate_names[@]} -gt 0 ]; then
        echo -e "${RED}  ✗ 发现重复的路由名称:${NC}"
        printf "    %s\n" "${duplicate_names[@]}"
        ((ERRORS++))
        return 1
    else
        echo -e "${GREEN}  ✓ 所有路由名称唯一 (共 ${#route_names[@]} 个)${NC}"
        return 0
    fi
}

# 验证 URI 冲突
validate_uri_conflicts() {
    echo -e "${BLUE}验证 URI 路径冲突...${NC}"
    
    local uri_map=()
    local conflicts=()
    
    # 收集所有 URI
    shopt -s nullglob
    for route_file in "${ROUTES_DIR}"/*.yaml; do
        if [ ! -f "$route_file" ]; then
            continue
        fi
        
        while IFS='|' read -r name uri methods; do
            if [ -n "$uri" ]; then
                # 检查冲突（简化版：完全匹配）
                for existing in "${uri_map[@]}"; do
                    if [ "$existing" = "$uri" ]; then
                        conflicts+=("$uri")
                    fi
                done
                uri_map+=("$uri|$name|$methods")
            fi
        done < <(extract_routes "$route_file" 2>/dev/null)
    done
    
    # 处理主配置文件（如果存在）
    if [ -f "${CONFIG_DIR}/apisix.yaml" ]; then
        route_file="${CONFIG_DIR}/apisix.yaml"
        while IFS='|' read -r name uri methods; do
            if [ -n "$uri" ]; then
                for existing in "${uri_map[@]}"; do
                    if [ "$existing" = "$uri" ]; then
                        conflicts+=("$uri")
                    fi
                done
                uri_map+=("$uri|$name|$methods")
            fi
        done < <(extract_routes "$route_file" 2>/dev/null)
    fi
    shopt -u nullglob
    
    if [ ${#conflicts[@]} -gt 0 ]; then
        echo -e "${YELLOW}  ⚠ 发现可能的 URI 冲突:${NC}"
        printf "    %s\n" "${conflicts[@]}"
        ((WARNINGS++))
        return 1
    else
        echo -e "${GREEN}  ✓ 未发现 URI 冲突${NC}"
        return 0
    fi
}

# 验证单个文件的必需字段
validate_file_required_fields() {
    local route_file=$1
    local filename=$(basename "$route_file")
    
    # 优先使用 yq
    if command -v yq &> /dev/null; then
        # 统计缺少必需字段的路由数量
        local missing_count=$(yq eval '[.routes[] | select((has("name") and has("uri") and has("methods") and has("upstream")) | not)] | length' "$route_file" 2>/dev/null)
        if [ "$missing_count" != "0" ] && [ -n "$missing_count" ]; then
            echo -e "  ${RED}✗ ${filename} 存在 ${missing_count} 个缺少必需字段的路由${NC}" >&2
            return 1
        fi
        return 0
    elif command -v python3 &> /dev/null && python3 -c "import yaml" 2>/dev/null; then
        python3 <<PYTHON_SCRIPT
import yaml
import sys

try:
    with open("$route_file", "r") as f:
        data = yaml.safe_load(f)
    
    routes = data.get("routes", [])
    for i, route in enumerate(routes):
        missing = []
        
        if not route.get("name"):
            missing.append("name")
        if not route.get("uri"):
            missing.append("uri")
        if not route.get("methods"):
            missing.append("methods")
        if not route.get("upstream"):
            missing.append("upstream")
        
        if missing:
            print(f"  ✗ 路由 #{i+1} 缺少字段: {', '.join(missing)}", file=sys.stderr)
            sys.exit(1)
except Exception as e:
    print(f"ERROR: {e}", file=sys.stderr)
    sys.exit(1)
PYTHON_SCRIPT
    else
        echo -e "${YELLOW}  ⚠ 无法验证必需字段（缺少 yq 或 PyYAML）${NC}" >&2
        return 0
    fi
}

# 验证必需字段
validate_required_fields() {
    echo -e "${BLUE}验证必需字段...${NC}"
    
    local missing_fields=0
    
    shopt -s nullglob
    for route_file in "${ROUTES_DIR}"/*.yaml; do
        if [ ! -f "$route_file" ]; then
            continue
        fi
        
        if ! validate_file_required_fields "$route_file"; then
            ((ERRORS++))
            missing_fields=1
        fi
    done
    
    # 处理主配置文件（如果存在）
    if [ -f "${CONFIG_DIR}/apisix.yaml" ]; then
        if ! validate_file_required_fields "${CONFIG_DIR}/apisix.yaml"; then
            ((ERRORS++))
            missing_fields=1
        fi
    fi
    shopt -u nullglob
    
    if [ $missing_fields -eq 0 ]; then
        echo -e "${GREEN}  ✓ 所有路由包含必需字段${NC}"
        return 0
    else
        return 1
    fi
}

# 主函数
main() {
    check_dependencies
    
    # 检查路由目录
    if [ ! -d "$ROUTES_DIR" ]; then
        echo -e "${YELLOW}⚠ 路由目录不存在: ${ROUTES_DIR}${NC}"
        echo "创建目录..."
        mkdir -p "$ROUTES_DIR"
    fi
    
    # 检查是否有路由文件
    shopt -s nullglob
    local route_files=("${ROUTES_DIR}"/*.yaml)
    if [ -f "${CONFIG_DIR}/apisix.yaml" ]; then
        route_files+=("${CONFIG_DIR}/apisix.yaml")
    fi
    shopt -u nullglob
    
    if [ ${#route_files[@]} -eq 0 ]; then
        echo -e "${YELLOW}⚠ 未找到路由配置文件${NC}"
        echo "请先运行: make generate-route 或 ./scripts/sync-routes.sh"
        exit 0
    fi
    
    # 执行验证
    echo ""
    for route_file in "${route_files[@]}"; do
        if [ -f "$route_file" ]; then
            validate_yaml_syntax "$route_file"
        fi
    done
    
    echo ""
    validate_route_uniqueness
    
    echo ""
    validate_uri_conflicts
    
    echo ""
    validate_required_fields
    
    # 总结
    echo ""
    echo -e "${GREEN}========================================${NC}"
    if [ $ERRORS -eq 0 ] && [ $WARNINGS -eq 0 ]; then
        echo -e "${GREEN}✓ 配置验证通过！${NC}"
        echo -e "${GREEN}========================================${NC}"
        exit 0
    else
        echo -e "${YELLOW}验证完成，发现问题：${NC}"
        echo "  错误: ${ERRORS}"
        echo "  警告: ${WARNINGS}"
        echo -e "${GREEN}========================================${NC}"
        if [ $ERRORS -gt 0 ]; then
            exit 1
        fi
    fi
}

# 运行主函数
main "$@"
