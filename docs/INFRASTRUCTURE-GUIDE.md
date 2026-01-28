# 微服务基础设施工具指南

本文档用通俗易懂的方式解释微服务开发中常用的基础设施工具，帮助你理解它们的作用和使用场景。

---

## 📦 1. CI/CD 配置 (GitHub Actions)

### 是什么？
**CI/CD = 持续集成/持续部署**

想象一下：
- **CI (持续集成)**：每次你提交代码，自动运行测试、检查代码质量
- **CD (持续部署)**：测试通过后，自动构建 Docker 镜像并部署到服务器

### 为什么需要？
**没有 CI/CD：**
```
1. 写代码
2. 手动运行测试
3. 手动构建 Docker 镜像
4. 手动上传到服务器
5. 手动部署
→ 容易出错，耗时，容易忘记步骤
```

**有 CI/CD：**
```
1. git push
2. 自动测试 ✅
3. 自动构建 ✅
4. 自动部署 ✅
→ 全自动，不会忘记步骤，减少人为错误
```

### 实际例子

```yaml
# .github/workflows/deploy.yml
name: Deploy Service

on:
  push:
    branches: [main]

jobs:
  deploy:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      
      # 1. 运行测试
      - name: Run Tests
        run: go test ./...
      
      # 2. 构建 Docker 镜像
      - name: Build Docker Image
        run: docker build -t my-service:latest .
      
      # 3. 推送到镜像仓库
      - name: Push to Registry
        run: docker push my-service:latest
      
      # 4. 部署到服务器
      - name: Deploy
        run: ssh server "docker pull my-service:latest && docker-compose up -d"
```

**使用场景：**
- ✅ 团队协作：多人提交代码，自动验证
- ✅ 减少错误：避免手动操作遗漏步骤
- ✅ 快速迭代：代码提交后自动部署

**什么时候需要？**
- 有多个开发者协作时
- 需要频繁部署时
- 想要自动化流程时

---

## 🗄️ 2. 数据库迁移工具 (golang-migrate)

### 是什么？
**数据库迁移 = 版本控制的数据库结构变更**

想象一下：
- 你的代码用 Git 管理版本（v1.0, v1.1, v1.2...）
- 数据库结构也需要版本管理（表结构变更、索引添加等）

### 为什么需要？
**没有迁移工具：**
```
开发环境：手动执行 SQL 创建表
测试环境：手动执行 SQL 创建表
生产环境：手动执行 SQL 创建表
→ 容易不一致，忘记执行某些 SQL，回滚困难
```

**有迁移工具：**
```
开发环境：migrate up → 自动执行所有迁移
测试环境：migrate up → 自动执行所有迁移
生产环境：migrate up → 自动执行所有迁移
→ 保证一致性，可以回滚，版本可控
```

### 实际例子

```bash
# 1. 创建迁移文件
migrate create -ext sql -dir migrations -seq add_user_table

# 生成两个文件：
# - 000001_add_user_table.up.sql   (升级)
# - 000001_add_user_table.down.sql (回滚)
```

```sql
-- migrations/000001_add_user_table.up.sql
CREATE TABLE users (
    id BIGSERIAL PRIMARY KEY,
    username VARCHAR(255) NOT NULL,
    email VARCHAR(255) UNIQUE NOT NULL,
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_users_email ON users(email);
```

```sql
-- migrations/000001_add_user_table.down.sql
DROP INDEX IF EXISTS idx_users_email;
DROP TABLE IF EXISTS users;
```

**使用方式：**
```bash
# 升级到最新版本
migrate -path migrations -database "postgres://user:pass@localhost/dbname?sslmode=disable" up

# 回滚一个版本
migrate -path migrations -database "postgres://user:pass@localhost/dbname?sslmode=disable" down 1

# 查看当前版本
migrate -path migrations -database "postgres://user:pass@localhost/dbname?sslmode=disable" version
```

**使用场景：**
- ✅ 团队协作：多人修改数据库结构
- ✅ 环境一致性：开发/测试/生产保持一致
- ✅ 版本回滚：出问题时可以回滚数据库结构

**什么时候需要？**
- 使用关系型数据库（PostgreSQL/MySQL）时
- 数据库结构会频繁变更时
- 需要保证多环境一致时

---

## 📊 3. Prometheus + Grafana（监控仪表盘）

### 是什么？
**Prometheus = 指标收集器**
**Grafana = 可视化仪表盘**

想象一下：
- 你的服务就像一辆车
- **Prometheus** 是仪表盘上的各种传感器（速度、油量、温度）
- **Grafana** 是仪表盘上的显示屏（图表、告警）

### 为什么需要？
**没有监控：**
```
用户：服务怎么这么慢？
你：不知道，让我看看日志...
→ 被动发现问题，难以定位问题
```

**有监控：**
```
Grafana 显示：CPU 使用率 95%，内存不足
你：立即扩容或优化代码
→ 主动发现问题，快速定位
```

### 实际例子

你的服务已经集成了 Prometheus（`pkg/metrics/metrics.go`），会自动暴露指标：

```go
// 你的代码中已经有这些指标
http_requests_total{method="GET", endpoint="/api/users"}
http_request_duration_seconds{method="GET"}
grpc_requests_total{method="CreateUser"}
```

**Grafana 仪表盘示例：**
```
┌─────────────────────────────────────────┐
│  📊 服务监控仪表盘                        │
├─────────────────────────────────────────┤
│  CPU 使用率: 45%  ████████░░░░          │
│  内存使用: 2.1GB / 4GB  ████████░░░░    │
│  请求数/秒: 1,234  ↗️ +12%              │
│  错误率: 0.1%  ✅                        │
│  平均响应时间: 120ms  ✅                 │
└─────────────────────────────────────────┘
```

**配置示例：**

```yaml
# docker-compose.dev.yml
services:
  prometheus:
    image: prom/prometheus:latest
    volumes:
      - ./prometheus/prometheus.yml:/etc/prometheus/prometheus.yml
    ports:
      - "9090:9090"
  
  grafana:
    image: grafana/grafana:latest
    ports:
      - "3000:3000"
    environment:
      - GF_SECURITY_ADMIN_PASSWORD=admin
```

**使用场景：**
- ✅ 性能监控：CPU、内存、请求数
- ✅ 错误监控：错误率、异常告警
- ✅ 业务指标：用户注册数、订单数等

**什么时候需要？**
- 服务上线后需要监控时
- 需要了解服务性能时
- 需要设置告警时

---

## 📝 4. 日志聚合 (ELK Stack)

### 是什么？
**ELK = Elasticsearch + Logstash + Kibana**

想象一下：
- 你有 10 个微服务，每个服务都有日志文件
- **Elasticsearch**：存储所有日志（搜索引擎）
- **Logstash**：收集和解析日志
- **Kibana**：可视化查询日志

### 为什么需要？
**没有日志聚合：**
```
服务1日志：/var/log/service1.log
服务2日志：/var/log/service2.log
服务3日志：/var/log/service3.log
...
→ 需要登录每台服务器查看日志，难以关联分析
```

**有日志聚合：**
```
所有日志 → Elasticsearch → Kibana 统一查询
→ 一个界面查看所有日志，可以搜索、过滤、分析
```

### 实际例子

**Kibana 查询界面：**
```
搜索框：[user_id:123 AND error]
结果：
- 2024-01-27 10:23:45 [user-service] ERROR: User not found
- 2024-01-27 10:23:46 [order-service] ERROR: Failed to create order
→ 发现用户服务报错导致订单服务失败
```

**配置示例：**

```yaml
# docker-compose.dev.yml
services:
  elasticsearch:
    image: docker.elastic.co/elasticsearch/elasticsearch:8.11.0
    environment:
      - discovery.type=single-node
  
  logstash:
    image: docker.elastic.co/logstash/logstash:8.11.0
    volumes:
      - ./logstash/pipeline:/usr/share/logstash/pipeline
  
  kibana:
    image: docker.elastic.co/kibana/kibana:8.11.0
    ports:
      - "5601:5601"
```

**使用场景：**
- ✅ 多服务日志统一查看
- ✅ 错误排查：搜索错误关键词
- ✅ 性能分析：分析慢请求日志

**什么时候需要？**
- 有多个微服务时
- 需要快速定位问题时
- 需要日志分析时

---

## 🔍 5. 分布式追踪 (Jaeger)

### 是什么？
**分布式追踪 = 追踪一个请求经过的所有服务**

想象一下：
```
用户请求：创建订单
  ↓
API Gateway (10ms)
  ↓
用户服务 (50ms)
  ↓
订单服务 (100ms)
  ↓
支付服务 (200ms)
  ↓
总计：360ms
```

**Jaeger** 可以可视化这个调用链，看到每个服务的耗时。

### 为什么需要？
**没有追踪：**
```
用户：订单创建很慢
你：不知道是哪个服务慢
→ 需要逐个服务查看日志，耗时
```

**有追踪：**
```
Jaeger 显示：
- API Gateway: 10ms ✅
- 用户服务: 50ms ✅
- 订单服务: 100ms ✅
- 支付服务: 200ms ⚠️ ← 瓶颈在这里！
→ 立即定位到支付服务慢
```

### 实际例子

**Jaeger UI 界面：**
```
┌─────────────────────────────────────────┐
│  🔍 请求追踪                            │
├─────────────────────────────────────────┤
│  Trace ID: abc123                       │
│  总耗时: 360ms                          │
│                                         │
│  ┌─ API Gateway (10ms)                 │
│  │  ┌─ 用户服务 (50ms)                  │
│  │  │  ┌─ 订单服务 (100ms)              │
│  │  │  │  └─ 支付服务 (200ms) ⚠️        │
└─────────────────────────────────────────┘
```

**代码集成：**

```go
import "go.opentelemetry.io/otel"

// 在你的 handler 中
func (h *Handler) CreateOrder(ctx context.Context, req *pb.CreateOrderReq) (*pb.CreateOrderResp, error) {
    // 创建 span（追踪单元）
    ctx, span := otel.Tracer("order-service").Start(ctx, "CreateOrder")
    defer span.End()
    
    // 业务逻辑
    order, err := h.logic.CreateOrder(ctx, req)
    
    // 记录属性
    span.SetAttributes(
        attribute.String("order.id", order.ID),
        attribute.Int("order.amount", order.Amount),
    )
    
    return order, err
}
```

**使用场景：**
- ✅ 性能优化：找到慢服务
- ✅ 错误排查：追踪错误传播路径
- ✅ 服务依赖分析：了解服务调用关系

**什么时候需要？**
- 有多个微服务相互调用时
- 需要性能优化时
- 需要排查复杂问题时

---

## 🛡️ 6. 限流熔断（APISIX 插件）

### 是什么？
**限流 = 限制请求数量**
**熔断 = 服务异常时快速失败**

想象一下：
- **限流**：餐厅限流，人太多时排队
- **熔断**：电路保险丝，电流过大时自动断开

### 为什么需要？
**没有限流：**
```
恶意用户：每秒发送 10000 个请求
你的服务：崩溃 💥
→ 服务被攻击，正常用户无法使用
```

**有限流：**
```
恶意用户：每秒发送 10000 个请求
限流器：只允许每秒 100 个请求
你的服务：正常运行 ✅
→ 服务稳定，正常用户不受影响
```

**没有熔断：**
```
支付服务：挂了
订单服务：一直等待支付服务响应（30秒超时）
用户：等待 30 秒后看到错误
→ 用户体验差
```

**有熔断：**
```
支付服务：挂了
熔断器：检测到错误率 > 50%，立即熔断
订单服务：立即返回错误（1秒内）
用户：快速看到错误，可以重试
→ 快速失败，用户体验好
```

### 实际例子

**APISIX 配置（你的项目已支持）：**

```yaml
# apisix/config/routes/user-routes.yaml
routes:
  - uri: /api/v1/users
    plugins:
      # 限流：每秒最多 100 个请求
      limit-req:
        rate: 100
        burst: 200
        rejected_code: 429
      
      # 熔断：错误率 > 50% 时熔断
      circuit-breaker:
        failure_ratio: 0.5
        timeout: 10
        min_health_capacity: 10
```

**使用场景：**
- ✅ 防止 DDoS 攻击
- ✅ 保护后端服务不被压垮
- ✅ 快速失败，提升用户体验

**什么时候需要？**
- 服务上线后
- 有恶意请求风险时
- 需要保护后端服务时

---

## 📋 总结：什么时候需要什么？

### 🟢 立即需要（项目已有）
- ✅ **限流熔断**：APISIX 已配置，开箱即用

### 🟡 开发阶段需要
- ⚠️ **数据库迁移**：如果使用 PostgreSQL/MySQL，建议添加
- ⚠️ **CI/CD**：多人协作时建议添加

### 🔵 上线后需要
- 📊 **Prometheus + Grafana**：监控服务性能
- 📝 **日志聚合 (ELK)**：多服务时统一查看日志
- 🔍 **分布式追踪 (Jaeger)**：排查复杂问题时使用

---

## 🚀 快速开始建议

### 阶段 1：开发阶段（现在）
```bash
# 1. 添加数据库迁移（如果使用 PostgreSQL）
# 2. 添加 CI/CD（如果多人协作）
```

### 阶段 2：测试阶段
```bash
# 1. 添加 Prometheus + Grafana（监控测试环境）
# 2. 验证限流熔断配置
```

### 阶段 3：生产阶段
```bash
# 1. 完善监控告警
# 2. 添加日志聚合
# 3. 添加分布式追踪（按需）
```

---

## 💡 实用建议

1. **不要一次性添加所有工具**：先解决当前痛点
2. **从简单开始**：先添加 Prometheus（你的服务已支持）
3. **按需添加**：遇到问题时再添加对应工具
4. **文档优先**：添加工具时记得写文档

---

**记住：工具是为了解决问题，不是为了使用而使用。先开发核心功能，遇到问题时再添加对应的基础设施工具！** 🎯
