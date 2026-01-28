#!/bin/bash

# APISIX 配置生成脚本
# 功能：从模板文件生成配置文件，并注入环境变量
# 安全：避免将包含敏感信息的配置文件提交到 Git

set -e

# 获取项目根目录
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"

# 加载 .env 文件
if [[ -f "${PROJECT_ROOT}/.env" ]]; then
    set -a
    source "${PROJECT_ROOT}/.env"
    set +a
fi

# 配置变量
APISIX_ADMIN_KEY="${APISIX_ADMIN_KEY:-edd1c9f034335f136f87ad84b625c8f1}"
CONFIG_TEMPLATE="${PROJECT_ROOT}/apisix/config/config.yaml.tmpl"
CONFIG_FILE="${PROJECT_ROOT}/apisix/config/config.yaml"
GLOBAL_TEMPLATE="${PROJECT_ROOT}/apisix/config/global.yaml.tmpl"
GLOBAL_FILE="${PROJECT_ROOT}/apisix/config/global.yaml"

# 检查模板文件是否存在
if [ ! -f "$CONFIG_TEMPLATE" ]; then
    echo "❌ 错误: 模板文件不存在: $CONFIG_TEMPLATE"
    exit 1
fi

if [ ! -f "$GLOBAL_TEMPLATE" ]; then
    echo "❌ 错误: 模板文件不存在: $GLOBAL_TEMPLATE"
    exit 1
fi

# 从模板生成 config.yaml
echo "📝 生成 APISIX 配置文件..."
cp "$CONFIG_TEMPLATE" "$CONFIG_FILE"

# 替换环境变量
# 处理 ${{VAR:=default}} 语法
sed -i.bak "s|\${{APISIX_ADMIN_KEY:=.*}}|${APISIX_ADMIN_KEY}|g" "$CONFIG_FILE"
rm -f "${CONFIG_FILE}.bak"

# 从模板生成 global.yaml
cp "$GLOBAL_TEMPLATE" "$GLOBAL_FILE"

echo "✅ APISIX 配置文件已生成:"
echo "   • config.yaml (APISIX_ADMIN_KEY 已注入)"
echo "   • global.yaml (JWT_SECRET 将在 make update-routes 时注入)"
