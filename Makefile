.PHONY: help start stop restart logs status new-service update-routes validate clean

# 默认目标
.DEFAULT_GOAL := help

# ==================== 基础设施管理 ====================

## start: 启动 API Gateway 基础设施 (APISIX + etcd + Redis)
start:
	@echo "🚀 启动 API Gateway 基础设施..."
	@docker compose up -d
	@echo ""
	@echo "✅ 基础设施已启动！"
	@echo ""
	@echo "📡 服务访问地址:"
	@echo "   • API Gateway:  http://localhost:9080"
	@echo "   • Admin API:    http://localhost:9180"
	@echo "   • etcd:         http://localhost:2379"
	@echo "   • Redis:        localhost:6379"
	@echo ""
	@echo "💡 提示:"
	@echo "   • 使用 'make new-service' 创建新的微服务"
	@echo "   • 使用 'make update-routes' 同步路由配置到 APISIX"
	@echo "   • 使用 'make logs' 查看日志"

## stop: 停止所有服务
stop:
	@echo "🛑 停止基础设施..."
	@docker compose down
	@echo "✅ 已停止！"

## restart: 重启所有服务
restart: stop start

## logs: 查看服务日志
logs:
	@docker compose logs -f

## status: 查看服务状态
status:
	@echo "📊 服务状态:"
	@docker compose ps

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
	@echo "   5. make run    # 启动服务(包含数据库)"
	@echo ""
	@echo "   6. 配置路由: 在 ../../apisix/config/routes/ 创建路由文件"
	@echo "   7. cd ../../ && make update-routes  # 同步路由到 APISIX"

## update-routes: 更新 APISIX 路由配置
update-routes:
	@echo "🔄 更新 APISIX 路由配置..."
	@./scripts/merge-routes.sh
	@echo "✅ 路由配置已更新！"

## validate: 验证配置文件
validate:
	@echo "🔍 验证配置文件..."
	@./scripts/validate-config.sh

# ==================== 工具命令 ====================

## clean: 清理生成的文件和容器
clean:
	@echo "🧹 清理环境..."
	@docker compose down -v
	@echo "✅ 清理完成！"

## help: 显示帮助信息
help:
	@echo "uyou API Gateway - 可用命令"
	@echo ""
	@echo "基础设施管理:"
	@sed -n 's/^##//p' ${MAKEFILE_LIST} | column -t -s ':' | grep -E "start|stop|restart|logs|status" | sed -e 's/^/  /'
	@echo ""
	@echo "服务开发:"
	@sed -n 's/^##//p' ${MAKEFILE_LIST} | column -t -s ':' | grep -E "new-service|update-routes|validate" | sed -e 's/^/  /'
	@echo ""
	@echo "工具命令:"
	@sed -n 's/^##//p' ${MAKEFILE_LIST} | column -t -s ':' | grep -E "clean|help" | sed -e 's/^/  /'
	@echo ""
	@echo "快速开始:"
	@echo "  1. make start          # 启动基础设施"
	@echo "  2. make new-service    # 创建微服务"
	@echo "  3. make update-routes  # 配置路由"
	@echo ""
	@echo "详细文档: docs/QUICKSTART.md"
