#!/bin/bash

# APISIX 配置生成脚本
# 功能：从模板文件生成配置文件，并注入环境变量
# 安全：避免将包含敏感信息的配置文件提交到 Git
# 自动生成：如果 .env 中缺少密钥，自动使用 openssl 生成

set -e

# 获取项目根目录
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
ENV_FILE="${PROJECT_ROOT}/.env"

# 生成随机密钥的函数
generate_secret() {
    local length=$1
    local method=$2  # hex 或 base64
    
    if [ "$method" = "hex" ]; then
        openssl rand -hex "$length"
    else
        openssl rand -base64 "$length"
    fi
}

# 检查并生成密钥
ensure_secret() {
    local key_name=$1
    local key_length=$2
    local key_method=$3  # hex 或 base64
    local description=$4
    
    # 如果 .env 文件不存在，从 .env.example 创建
    if [ ! -f "$ENV_FILE" ]; then
        if [ -f "${PROJECT_ROOT}/.env.example" ]; then
            echo "📋 从 .env.example 创建 .env 文件..."
            cp "${PROJECT_ROOT}/.env.example" "$ENV_FILE"
        else
            echo "❌ 错误: .env.example 文件不存在"
            exit 1
        fi
    fi
    
    # 读取当前值
    local current_value=""
    if grep -q "^${key_name}=" "$ENV_FILE"; then
        current_value=$(grep "^${key_name}=" "$ENV_FILE" | cut -d'=' -f2- | sed 's/^[[:space:]]*//;s/[[:space:]]*$//')
    fi
    
    # 如果值为空，生成新密钥
    if [ -z "$current_value" ]; then
        echo "🔑 生成 ${description}..."
        local new_secret=$(generate_secret "$key_length" "$key_method")
        
        # 更新 .env 文件
        if grep -q "^${key_name}=" "$ENV_FILE"; then
            # 如果键已存在，替换值
            if [[ "$OSTYPE" == "darwin"* ]]; then
                # macOS 使用不同的 sed 语法
                sed -i '' "s|^${key_name}=.*|${key_name}=${new_secret}|" "$ENV_FILE"
            else
                # Linux
                sed -i "s|^${key_name}=.*|${key_name}=${new_secret}|" "$ENV_FILE"
            fi
        else
            # 如果键不存在，追加到文件末尾
            echo "${key_name}=${new_secret}" >> "$ENV_FILE"
        fi
        
        echo "   ✓ ${key_name} 已生成并保存到 .env"
    else
        echo "   ✓ ${key_name} 已存在，跳过生成"
    fi
}

# 确保必要的密钥存在
echo "🔐 检查密钥配置..."
ensure_secret "APISIX_ADMIN_KEY" 16 "hex" "APISIX Admin API Key"
ensure_secret "JWT_SECRET" 32 "base64" "JWT Secret Key"

# 重新加载 .env 文件（可能已更新）
if [[ -f "$ENV_FILE" ]]; then
    set -a
    source "$ENV_FILE"
    set +a
fi

# 验证密钥已设置
if [ -z "$APISIX_ADMIN_KEY" ]; then
    echo "❌ 错误: APISIX_ADMIN_KEY 未设置"
    exit 1
fi

if [ -z "$JWT_SECRET" ]; then
    echo "❌ 错误: JWT_SECRET 未设置"
    exit 1
fi

# 配置文件和模板路径
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
echo ""
echo "📝 生成 APISIX 配置文件..."
cp "$CONFIG_TEMPLATE" "$CONFIG_FILE"

# 替换环境变量
# 处理 ${{VAR}} 语法（新格式，无默认值）
if [[ "$OSTYPE" == "darwin"* ]]; then
    # macOS
    sed -i '' "s|\${{APISIX_ADMIN_KEY}}|${APISIX_ADMIN_KEY}|g" "$CONFIG_FILE"
else
    # Linux
    sed -i "s|\${{APISIX_ADMIN_KEY}}|${APISIX_ADMIN_KEY}|g" "$CONFIG_FILE"
fi

# 从模板生成 global.yaml
cp "$GLOBAL_TEMPLATE" "$GLOBAL_FILE"

echo "✅ APISIX 配置文件已生成:"
echo "   • config.yaml (APISIX_ADMIN_KEY 已注入)"
echo "   • global.yaml (JWT_SECRET 将在 make update-routes 时注入)"
