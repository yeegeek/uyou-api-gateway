# 生产部署指南

本指南提供了将基于 uyou-api-gateway 框架开发的应用部署到生产环境的建议和步骤。

## 目录

1. [部署理念：不可变基础设施](#1-部署理念不可变基础设施)
2. [环境划分](#2-环境划分)
3. [部署架构](#3-部署架构)
4. [构建与打包 (CI)](#4-构建与打包-ci)
5. [部署流程 (CD)](#5-部署流程-cd)
6. [配置管理](#6-配置管理)
7. [监控与告警](#7-监控与告警)

---

## 1. 部署理念：不可变基础设施

生产环境的部署应遵循 **不可变基础设施 (Immutable Infrastructure)** 的原则。这意味着:

- **不直接修改服务器**: 任何变更 (如更新代码、修改配置) 都通过构建一个新的 Docker 镜像来完成。
- **滚动更新**: 新版本的部署通过逐步替换旧版本的容器实例来实现, 而不是在现有容器上执行更新操作。

**优势**:
- **一致性**: 保证开发、测试和生产环境的一致性。
- **可靠性**: 部署过程可预测, 易于回滚 (只需重新部署旧版本的镜像)。
- **可追溯性**: 每个部署版本都对应一个唯一的 Docker 镜像 Tag。

## 2. 环境划分

建议至少划分以下三个环境:

- **开发 (Development)**: 开发者本地环境, 使用 `make run` 快速启动和调试。
- **预发 (Staging)**: 与生产环境配置完全一致的测试环境, 用于部署前的最终验证。
- **生产 (Production)**: 面向最终用户的正式环境。

## 3. 部署架构

推荐使用 **Kubernetes (K8s)** 作为容器编排平台。K8s 提供了服务发现、自动扩缩容、滚动更新、健康检查等生产级特性。

```mermaid
graph TD
    subgraph "外部流量"
        A[Internet] --> B[Cloud Load Balancer]
    end

    subgraph "Kubernetes 集群"
        B --> C[Ingress Controller (Nginx/Traefik)]
        C --> D[APISIX Cluster (Deployment)]
        D -- "gRPC" --> E[UserService (Deployment)]
        D -- "gRPC" --> F[OrderService (Deployment)]
        
        subgraph "基础设施"
            G[etcd Cluster (StatefulSet)]
            H[PostgreSQL (StatefulSet/Cloud SQL)]
            I[Redis Cluster (StatefulSet/Cloud Memorystore)]
        end

        D -- "配置" --> G
        E -- "数据" --> H
        F -- "数据" --> H
        E -- "缓存" --> I
    end
```

**组件说明**:
- **Load Balancer**: 云服务商提供的负载均衡器 (如 AWS ALB, GCP Load Balancer), 将流量分发到 K8s 集群。
- **Ingress Controller**: K8s 的流量入口, 将外部 HTTP/S 请求路由到集群内的 APISIX 服务。
- **APISIX Cluster**: 以 K8s `Deployment` 的形式部署, 可以轻松扩展实例数量。
- **Microservices**: 每个微服务 (如 UserService, OrderService) 都作为独立的 `Deployment` 部署。
- **Stateful Components**: `etcd`, `PostgreSQL`, `Redis` 等有状态服务应使用 `StatefulSet` 或云服务商提供的托管数据库/缓存服务, 以保证数据持久性和稳定性。

## 4. 构建与打包 (CI)

持续集成 (CI) 流程负责在代码提交后自动构建和打包应用。推荐使用 GitHub Actions, GitLab CI, 或 Jenkins。

**CI 流程示例 (GitHub Actions)**:

对于每个微服务 (例如 `user-service`):

```yaml
# .github/workflows/ci.yml
name: CI for user-service

on:
  push:
    branches:
      - main

jobs:
  build-and-push:
    runs-on: ubuntu-latest
    steps:
      - name: Checkout
        uses: actions/checkout@v3

      - name: Set up Docker Buildx
        uses: docker/setup-buildx-action@v2

      - name: Login to Docker Hub
        uses: docker/login-action@v2
        with:
          username: ${{ secrets.DOCKERHUB_USERNAME }}
          password: ${{ secrets.DOCKERHUB_TOKEN }}

      - name: Build and push
        uses: docker/build-push-action@v4
        with:
          context: .
          push: true
          tags: your-repo/user-service:${{ github.sha }}
```

**关键步骤**:
1. **代码检出**: 获取最新代码。
2. **登录镜像仓库**: 登录到 Docker Hub, Harbor 或其他镜像仓库。
3. **构建并推送**: 使用服务的 `Dockerfile` 构建镜像, 并使用 Git commit SHA 作为唯一的 Tag 推送到仓库。

## 5. 部署流程 (CD)

持续部署 (CD) 流程负责将 CI 构建的新镜像部署到 K8s 集群。推荐使用 Argo CD (GitOps) 或 Spinnaker。

**使用 Argo CD (GitOps) 的流程**:

1. **创建 K8s 清单**: 为每个服务创建 K8s `Deployment` 和 `Service` 的 YAML 文件, 并存放在一个独立的 Git 仓库 (配置仓库) 中。

   ```yaml
   # user-service-deployment.yaml
   apiVersion: apps/v1
   kind: Deployment
   metadata:
     name: user-service
   spec:
     replicas: 3
     template:
       spec:
         containers:
           - name: user-service
             image: your-repo/user-service:latest-tag # 镜像地址和 Tag
   ```

2. **配置 Argo CD**: 在 Argo CD 中创建一个 Application, 监控这个配置仓库。

3. **触发部署**: 当需要部署新版本时, CI 流程在构建完镜像后, 自动更新配置仓库中 `Deployment` YAML 文件的 `image` Tag。

4. **自动同步**: Argo CD 检测到配置仓库的变化, 会自动将新的清单应用到 K8s 集群, 触发滚动更新。

## 6. 配置管理

生产环境的配置 (如数据库密码, API Key) 不能硬编码在代码或镜像中。

**推荐方案**:
- **Kubernetes Secrets**: 用于存储敏感信息, 如数据库密码、JWT 密钥等。应用通过环境变量或挂载卷的方式引用这些 Secret。
- **Kubernetes ConfigMaps**: 用于存储非敏感配置, 如数据库主机名、Redis 地址等。

**示例**: 在 `Deployment` 中引用 `ConfigMap` 和 `Secret`。

```yaml
spec:
  template:
    spec:
      containers:
        - name: user-service
          env:
            - name: DB_PASSWORD
              valueFrom:
                secretKeyRef:
                  name: postgres-secret
                  key: password
            - name: DB_HOST
              valueFrom:
                configMapKeyRef:
                  name: app-config
                  key: postgres_host
```

## 7. 监控与告警

- **APISIX 监控**: APISIX 内置 `prometheus` 插件, 可以暴露丰富的指标 (如 QPS, 延迟, 状态码)。
- **服务监控**: 在你的 Go 服务中引入 Prometheus 客户端库, 暴露业务相关的指标 (如用户注册数, 订单创建数)。
- **日志**: 将所有服务的日志输出到 `stdout`, 由 K8s 的日志收集系统 (如 Fluentd, Logstash) 统一收集到 ELK 或 Loki 进行存储和查询。
- **链路追踪**: 使用 OpenTelemetry 等工具实现分布式链路追踪, 快速定位性能瓶颈。
- **告警**: 使用 Prometheus + Alertmanager, 根据关键指标 (如 5xx 错误率, CPU/内存使用率) 设置告警规则, 及时通知开发团队。
