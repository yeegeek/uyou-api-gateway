.PHONY: help start stop restart logs status new-service update-routes validate clean

# 默认目标
.DEFAULT_GOAL := help

# 检测是否使用 dev 环境
ifeq ($(filter dev,$(MAKECMDGOALS)),dev)
	COMPOSE_FILE := -f docker-compose.dev.yml
	ENV_SUFFIX := (开发环境)
else
	COMPOSE_FILE := -f docker-compose.yml
	ENV_SUFFIX := (生产环境)
endif

# APISIX 相关配置
APISIX_ADMIN_URL ?= http://localhost:9180
APISIX_ADMIN_KEY ?= edd1c9f034335f136f87ad84b625c8f1

# ==================== 基础设施管理 ====================

## start: 启动服务 (使用 'make start dev' 启动开发环境)
start:
	@echo "🚀 启动服务 $(ENV_SUFFIX)..."
	@docker compose $(COMPOSE_FILE) up -d
	@echo ""
	@echo "✅ 服务已启动！"
	@echo ""
	@echo "📡 服务访问地址:"
	@echo "   • API Gateway:  http://localhost:9080"
	@echo "   • Admin API:    http://localhost:9180"
	@echo "   • etcd:         http://localhost:2379"
ifeq ($(filter dev,$(MAKECMDGOALS)),dev)
	@echo "   • PostgreSQL:   localhost:5432"
	@echo "   • MongoDB:      localhost:27017"
	@echo "   • Redis:        localhost:6379"
endif
	@echo ""
	@echo "💡 提示:"
	@echo "   • 使用 'make new-service' 创建新的微服务"
	@echo "   • 使用 'make update-routes' 同步路由配置到 APISIX"
	@echo "   • 使用 'make logs $(if $(filter dev,$(MAKECMDGOALS)),dev,)' 查看日志"

## stop: 停止服务 (使用 'make stop dev' 停止开发环境)
stop:
	@echo "🛑 停止服务 $(ENV_SUFFIX)..."
	@docker compose $(COMPOSE_FILE) down
	@echo "✅ 已停止！"

## restart: 重启服务 (使用 'make restart dev' 重启开发环境)
restart:
	@echo "🔄 重启服务 $(ENV_SUFFIX)..."
	@docker compose $(COMPOSE_FILE) restart
	@echo "✅ 已重启！"

## logs: 查看服务日志 (使用 'make logs dev' 查看开发环境日志)
logs:
	@docker compose $(COMPOSE_FILE) logs -f

## status: 查看服务状态 (使用 'make status dev' 查看开发环境状态)
status:
	@echo "📊 服务状态 $(ENV_SUFFIX):"
	@docker compose $(COMPOSE_FILE) ps

## clean: 清理服务和数据卷 (使用 'make clean dev' 清理开发环境)
clean:
	@echo "🧹 清理环境 $(ENV_SUFFIX)..."
	@docker compose $(COMPOSE_FILE) down -v
	@echo "✅ 清理完成！"

# dev 是一个伪目标，用于配合其他命令使用
dev:
	@:

# ==================== 服务开发 ====================

## new-service: 创建新的微服务 (交互式)
new-service:
	@echo "🚀 创建新的微服务..."
	@echo ""
	@cd scaffold && go run generator.go
	@echo ""
	@echo "✅ 服务创建成功！"
	@echo ""
	@echo "📝 后续步骤:"
	@echo "   1. cd services/<service-name>"
	@echo "   2. 编辑 api/proto/*.proto 定义 API"
	@echo "   3. make proto  # 生成 gRPC 代码"
	@echo "   4. 实现业务逻辑"
	@echo "   5. cd ../../ && make start dev  # 启动开发环境"
	@echo ""
	@echo "   6. 配置路由: 在 apisix/config/routes/ 创建路由文件"
	@echo "   7. make update-routes  # 同步路由到 APISIX"

## update-routes: 合并并更新路由 (生产环境会同时归档 Proto)
update-routes:
	@echo "🔄 正在构建并更新 APISIX 路由配置..."
	@./scripts/merge-routes.sh
	@echo "✅ 路由配置已同步！"

## deploy-routes: 仅部署现有配置 (不依赖微服务源码)
deploy-routes:
	@echo "🚀 正在部署现有路由配置到 APISIX..."
	@./scripts/merge-routes.sh --deploy-only
	@echo "✅ 部署完成！"

## validate: 验证配置文件
validate:
	@echo "🔍 验证配置文件..."
	@./scripts/validate-config.sh

## apisix-status: 查看 APISIX 当前生效的配置 (Routes, Consumers, Global Rules)
apisix-status:
	@echo "📊 APISIX 当前配置:"
	@echo "--- 路由 (Routes) ---"
	@curl -s $(APISIX_ADMIN_URL)/apisix/admin/routes -H "X-API-KEY: $(APISIX_ADMIN_KEY)" | python3 -m json.tool | grep -E '"id":|"uri":|"name":' || echo "无"
	@echo "\n--- 消费者 (Consumers) ---"
	@curl -s $(APISIX_ADMIN_URL)/apisix/admin/consumers -H "X-API-KEY: $(APISIX_ADMIN_KEY)" | python3 -m json.tool | grep -E '"username":' || echo "无"
	@echo "\n--- 全局规则 (Global Rules) ---"
	@curl -s $(APISIX_ADMIN_URL)/apisix/admin/global_rules -H "X-API-KEY: $(APISIX_ADMIN_KEY)" | python3 -m json.tool | grep -E '"id":' || echo "无"

## apisix-clear: 清空 APISIX 所有配置 (危险操作)
apisix-clear:
	@echo "⚠️  正在清空 APISIX 所有配置..."
	@for resource in routes consumers global_rules protos; do \
		echo "清理 $$resource ..."; \
		items=$$(curl -s $(APISIX_ADMIN_URL)/apisix/admin/$$resource -H "X-API-KEY: $(APISIX_ADMIN_KEY)" | python3 -c "import sys, json; data=json.load(sys.stdin); print(' '.join([str(i.get('value', {}).get('id') or i.get('value', {}).get('username')) for i in data.get('list', []) if i.get('value')]))" 2>/dev/null); \
		for id in $$items; do \
			curl -s -X DELETE $(APISIX_ADMIN_URL)/apisix/admin/$$resource/$$id -H "X-API-KEY: $(APISIX_ADMIN_KEY)" > /dev/null; \
		done; \
	done
	@echo "✅ APISIX 已重置为初始状态！"

# ==================== 工具命令 ====================

## help: 显示帮助信息
help:
	@echo "╔════════════════════════════════════════════════════════╗"
	@echo "║        uyou API Gateway - 可用命令                     ║"
	@echo "╚════════════════════════════════════════════════════════╝"
	@echo ""
	@echo "基础设施管理:"
	@echo "  make start [dev]      启动服务 (加 dev 启动开发环境)"
	@echo "  make stop [dev]       停止服务"
	@echo "  make restart [dev]    重启服务"
	@echo "  make logs [dev]       查看日志"
	@echo "  make status [dev]     查看服务状态"
	@echo "  make clean [dev]      清理环境和数据卷"
	@echo ""
	@echo "服务开发:"
	@echo "  make new-service      创建新的微服务 (交互式)"
	@echo "  make update-routes    构建并更新 APISIX 路由配置"
	@echo "  make deploy-routes    仅部署现有配置 (生产环境)"
	@echo "  make validate         验证配置文件"
	@echo "  make apisix-status    查看 APISIX 当前生效的配置"
	@echo "  make apisix-clear     清空 APISIX 所有配置 (危险)"
	@echo ""
	@echo "工具命令:"
	@echo "  make help             显示此帮助信息"
	@echo ""
	@echo "环境说明:"
	@echo "  • 不加 dev: 使用 docker-compose.yml (生产环境)"
	@echo "  • 加 dev:   使用 docker-compose.dev.yml (开发环境)"
	@echo ""
	@echo "快速开始:"
	@echo "  1. make start dev          # 启动开发环境"
	@echo "  2. make new-service        # 创建微服务"
	@echo "  3. make update-routes      # 配置路由"
	@echo "  4. make logs dev           # 查看日志"
	@echo ""
	@echo "详细文档: docs/QUICKSTART.md"
