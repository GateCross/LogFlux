.PHONY: help build-frontend build-docker deploy up down restart logs status clean

help:
	@echo "LogFlux 部署管理"
	@echo ""
	@echo "可用命令:"
	@echo "  make build-frontend  - 构建前端"
	@echo "  make build-docker    - 构建 Docker 镜像"
	@echo "  make deploy          - 完整部署 (构建前端 + Docker + 启动)"
	@echo "  make up              - 启动容器"
	@echo "  make down            - 停止并删除容器"
	@echo "  make restart         - 重启容器"
	@echo "  make logs            - 查看日志"
	@echo "  make status          - 查看容器状态"
	@echo "  make clean           - 清理构建文件和容器"

build-frontend:
	@echo "构建前端..."
	cd frontend && pnpm install && pnpm run build

build-docker:
	@echo "构建 Docker 镜像..."
	docker-compose -f docker/docker-compose.yml build

deploy: build-frontend build-docker up
	@echo "部署完成!"
	@echo ""
	@make status

up:
	@echo "启动容器..."
	docker-compose -f docker/docker-compose.yml up -d

down:
	@echo "停止容器..."
	docker-compose -f docker/docker-compose.yml down

restart:
	@echo "重启容器..."
	docker-compose -f docker/docker-compose.yml restart

logs:
	docker-compose -f docker/docker-compose.yml logs -f

status:
	@echo "容器状态:"
	@docker-compose -f docker/docker-compose.yml ps
	@echo ""
	@echo "访问地址:"
	@echo "  HTTP:  http://localhost"
	@echo "  HTTPS: https://localhost"

clean:
	@echo "清理构建文件和容器..."
	docker-compose -f docker/docker-compose.yml down -v
	rm -rf frontend/dist
	@echo "清理完成!"
