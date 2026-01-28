# 配置文件迁移指南

## 🔄 从旧版本迁移

如果你之前已经克隆了仓库，需要执行以下步骤：

### 1. 备份现有配置（如果已修改）

```bash
# 如果已经修改过配置文件，先备份
cp apisix/config/config.yaml apisix/config/config.yaml.backup
cp apisix/config/global.yaml apisix/config/global.yaml.backup
```

### 2. 从 Git 中移除配置文件（如果已提交）

**重要**：如果配置文件已经被 Git 跟踪，需要先从跟踪中移除：

```bash
# 从 Git 跟踪中移除（但保留本地文件）
git rm --cached apisix/config/config.yaml
git rm --cached apisix/config/global.yaml

# 提交更改
git commit -m "chore: 将 APISIX 配置文件移到 .gitignore，使用模板生成"
```

**注意**：执行 `git rm --cached` 后，文件会从 Git 跟踪中移除，但本地文件仍然保留。之后这些文件会被 `.gitignore` 忽略。

### 3. 生成新配置文件

```bash
# 确保 .env 文件中有必要的环境变量
# 然后运行生成脚本
bash scripts/apisix-start.sh
```

### 4. 验证配置

```bash
# 检查配置文件是否已生成
ls -la apisix/config/config.yaml
ls -la apisix/config/global.yaml

# 验证配置是否正确
make validate
```

## ✅ 完成

现在配置文件会从模板自动生成，敏感信息不会提交到 Git。
