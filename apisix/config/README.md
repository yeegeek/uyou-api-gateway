# APISIX 配置文件说明

## 📁 文件结构

```
apisix/config/
├── config.yaml.tmpl      # APISIX 主配置模板（提交到 Git）
├── global.yaml.tmpl       # 全局配置模板（提交到 Git）
├── config.yaml           # 生成的配置文件（不提交，包含敏感信息）
├── global.yaml           # 生成的配置文件（不提交，包含敏感信息）
└── routes/               # 路由配置文件目录
```

## 🔒 安全说明

**重要**：`config.yaml` 和 `global.yaml` 包含敏感信息（如 `APISIX_ADMIN_KEY`），已添加到 `.gitignore`，**不会提交到 Git**。

实际配置文件由 `scripts/apisix-start.sh` 从模板文件（`.tmpl`）自动生成。

## 🚀 使用方式

### 首次使用

1. **设置环境变量**（在 `.env` 文件中）：
   ```bash
   APISIX_ADMIN_KEY=your_secret_key_here
   JWT_SECRET=your_jwt_secret_here
   ```

2. **生成配置文件**：
   ```bash
   # 方式 1: 手动生成
   bash scripts/apisix-start.sh
   
   # 方式 2: 启动服务时自动生成
   make start
   
   # 方式 3: 更新路由时自动生成
   make update-routes
   ```

### 配置文件生成流程

1. **`config.yaml`**：
   - 从 `config.yaml.tmpl` 复制
   - 替换 `${{APISIX_ADMIN_KEY:=...}}` 为环境变量值
   - 由 `scripts/apisix-start.sh` 处理

2. **`global.yaml`**：
   - 从 `global.yaml.tmpl` 复制
   - `JWT_SECRET` 在 `make update-routes` 时由 `scripts/merge-routes.sh` 注入

## 🔧 修改配置

### 修改非敏感配置

直接编辑模板文件（`.tmpl`），然后重新生成：

```bash
# 编辑模板
vim apisix/config/config.yaml.tmpl

# 重新生成配置文件
bash scripts/apisix-start.sh
```

### 修改敏感配置

在 `.env` 文件中修改环境变量，然后重新生成：

```bash
# 编辑 .env
vim .env

# 重新生成配置文件
bash scripts/apisix-start.sh
```

## ⚠️ 注意事项

1. **不要直接编辑** `config.yaml` 或 `global.yaml`，它们会被脚本覆盖
2. **修改模板文件**（`.tmpl`）后需要重新生成配置文件
3. **生产环境**必须修改 `APISIX_ADMIN_KEY` 和 `JWT_SECRET`
4. 配置文件会在以下时机自动生成：
   - `make start` 启动服务前
   - `make update-routes` 更新路由前
