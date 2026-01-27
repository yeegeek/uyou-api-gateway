# 故障排查指南

本文档提供常见问题的排查和解决方案。

## 目录

1. [服务启动问题](#1-服务启动问题)
2. [数据库连接问题](#2-数据库连接问题)
3. [APISIX 路由问题](#3-apisix-路由问题)
4. [JWT 认证问题](#4-jwt-认证问题)
5. [配置问题](#5-配置问题)

---

## 1. 服务启动问题

### 问题 1.1: 服务无法启动，端口被占用

**症状**: 
```
Error: listen tcp :50051: bind: address already in use
```

**解决方案**:
1. 检查端口是否被占用：
   ```bash
   lsof -i :50051
   # 或
   netstat -an | grep 50051
   ```

2. 停止占用端口的进程，或修改服务配置使用其他端口。

### 问题 1.2: 配置文件加载失败

**症状**:
```
panic: failed to unmarshal config: ...
```

**解决方案**:
1. 检查 `config/config.yaml` 文件格式是否正确（YAML 语法）。
2. 检查必需字段是否都已配置。
3. 查看详细错误信息，定位具体字段问题。

### 问题 1.3: 依赖服务未就绪

**症状**: 服务启动后立即退出，日志显示连接失败。

**解决方案**:
1. 确保所有依赖服务（数据库、Redis）已启动：
   ```bash
   docker compose -f docker-compose.dev.yml ps
   ```

2. 检查服务健康状态：
   ```bash
   docker compose -f docker-compose.dev.yml logs [service-name]
   ```

3. 使用 `depends_on` 确保服务启动顺序。

---

## 2. 数据库连接问题

### 问题 2.1: PostgreSQL 连接失败

**症状**:
```
failed to connect to PostgreSQL: connection refused
```

**解决方案**:
1. 检查 PostgreSQL 服务是否运行：
   ```bash
   docker compose -f docker-compose.dev.yml ps postgres
   ```

2. 检查连接配置（主机、端口、用户名、密码）是否正确。

3. 检查网络连接：
   ```bash
   docker compose -f docker-compose.dev.yml exec user ping postgres
   ```

4. 查看 PostgreSQL 日志：
   ```bash
   docker compose -f docker-compose.dev.yml logs postgres
   ```

### 问题 2.2: MongoDB 连接失败

**症状**:
```
failed to connect to MongoDB: no reachable servers
```

**解决方案**:
1. 检查 MongoDB URI 是否正确（包含数据库名称）。
2. 检查认证信息（用户名、密码）。
3. 确认 MongoDB 服务已启动并健康。

### 问题 2.3: 数据库连接池耗尽

**症状**: 服务运行一段时间后出现连接超时。

**解决方案**:
1. 检查连接池配置是否合理。
2. 确保数据库连接在使用后正确关闭。
3. 增加连接池大小（如果数据库支持）。

---

## 3. APISIX 路由问题

### 问题 3.1: 路由未生效

**症状**: 通过网关访问服务返回 404。

**解决方案**:
1. 检查路由配置是否正确：
   ```bash
   make apisix-status
   ```

2. 确认路由已同步：
   ```bash
   make update-routes
   ```

3. 检查路由文件格式（YAML 语法）。

4. 验证 proto 文件是否正确上传到 APISIX。

### 问题 3.2: gRPC 转换失败

**症状**: 返回 500 错误，日志显示协议转换失败。

**解决方案**:
1. 检查 proto 文件中的服务名和方法名是否匹配。
2. 确认 `grpc-transcode` 插件配置正确。
3. 检查请求体格式是否符合 proto 定义。

### 问题 3.3: 路由冲突

**症状**: 多个路由使用相同的 URI 和方法。

**解决方案**:
1. 运行配置验证：
   ```bash
   make validate
   ```

2. 检查路由 ID 是否唯一。
3. 确保 URI 和方法组合不冲突。

---

## 4. JWT 认证问题

### 问题 4.1: JWT 验证失败

**症状**: 返回 401 Unauthorized。

**解决方案**:
1. **检查 JWT Secret 是否一致**:
   - 确认 APISIX 和微服务使用相同的 `JWT_SECRET`。
   - 检查 `.env` 文件中的 `JWT_SECRET` 值。
   - 确认 `docker-compose.dev.yml` 中所有服务都配置了 `JWT_SECRET`。

2. **验证 Token 格式**:
   - 确认 Token 格式正确（Bearer token）。
   - 检查 Token 是否过期。

3. **同步 APISIX 配置**:
   ```bash
   make update-routes
   ```

### 问题 4.2: JWT Secret 不同步

**症状**: APISIX 验证通过，但微服务验证失败（或反之）。

**解决方案**:
1. 检查环境变量：
   ```bash
   docker compose -f docker-compose.dev.yml exec user env | grep JWT_SECRET
   docker compose -f docker-compose.dev.yml exec apisix env | grep JWT_SECRET
   ```

2. 确认 `.env` 文件存在且包含 `JWT_SECRET`。

3. 重启服务以加载新的环境变量：
   ```bash
   make restart dev
   ```

---

## 5. 配置问题

### 问题 5.1: 环境变量未生效

**症状**: 修改环境变量后，服务仍使用旧配置。

**解决方案**:
1. 确认环境变量命名正确（使用服务名前缀）。
2. 重启服务以加载新配置。
3. 检查 `docker-compose.dev.yml` 中的环境变量配置。

### 问题 5.2: 配置验证失败

**症状**: 服务启动时 panic，提示配置验证失败。

**解决方案**:
1. 查看具体验证错误信息。
2. 检查必需字段是否都已配置。
3. 验证字段值是否符合要求（如端口范围、字符串长度等）。

### 问题 5.3: 配置文件不存在

**症状**: 警告配置文件不存在，但服务仍能启动。

**说明**: 这是正常情况。如果所有配置都通过环境变量提供，配置文件可以不存在。

**解决方案**: 如果需要使用配置文件，创建 `config/config.yaml` 文件。

---

## 通用排查步骤

1. **查看日志**:
   ```bash
   # 查看所有服务日志
   make logs dev
   
   # 查看特定服务日志
   docker compose -f docker-compose.dev.yml logs -f [service-name]
   ```

2. **检查服务状态**:
   ```bash
   make status dev
   ```

3. **验证配置**:
   ```bash
   make validate
   ```

4. **重启服务**:
   ```bash
   make restart dev
   ```

5. **清理并重新启动**:
   ```bash
   make clean dev
   make start dev
   ```

---

## 获取帮助

如果以上方案无法解决问题，请：

1. 收集详细的错误日志。
2. 记录复现步骤。
3. 检查相关配置文件。
4. 提交 Issue 或联系维护团队。
