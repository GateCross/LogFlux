#!/bin/bash

set -e

echo "==================================="
echo "LogFlux Docker 部署脚本"
echo "==================================="

# 检查 Docker 是否安装
if ! command -v docker &> /dev/null; then
    echo "错误: Docker 未安装"
    exit 1
fi

if ! command -v docker-compose &> /dev/null; then
    echo "错误: Docker Compose 未安装"
    exit 1
fi

# 检查前端是否已构建
if [ ! -d "frontend/dist" ]; then
    echo "错误: 前端未构建,请先运行:"
    echo "  cd frontend && pnpm install && pnpm run build"
    exit 1
fi

# 检查配置文件
if [ ! -f "backend/etc/config.yaml" ]; then
    echo "错误: 配置文件不存在: backend/etc/config.yaml"
    exit 1
fi

# 提示用户检查配置
echo ""
echo "部署前检查清单:"
echo "  [√] Docker 已安装"
echo "  [√] Docker Compose 已安装"
echo "  [√] 前端已构建"
echo "  [√] 配置文件存在"
echo ""
echo "请确认以下配置是否正确:"
echo "  - backend/etc/config.yaml 中的数据库连接信息"
echo "  - backend/etc/config.yaml 中的 Redis 连接信息"
echo ""

read -p "是否继续部署? [y/N] " -n 1 -r
echo
if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    echo "已取消部署"
    exit 0
fi

# 构建镜像
echo ""
echo "==================================="
echo "构建 Docker 镜像..."
echo "==================================="
docker-compose -f docker/docker-compose.yml build

# 启动容器
echo ""
echo "==================================="
echo "启动容器..."
echo "==================================="
docker-compose -f docker/docker-compose.yml up -d

# 查看日志
echo ""
echo "==================================="
echo "部署完成!"
echo "==================================="
echo ""
echo "服务状态:"
docker-compose -f docker/docker-compose.yml ps
echo ""
echo "查看日志:"
echo "  docker-compose -f docker/docker-compose.yml logs -f"
echo ""
echo "停止服务:"
echo "  docker-compose -f docker/docker-compose.yml down"
echo ""
echo "重启服务:"
echo "  docker-compose -f docker/docker-compose.yml restart"
echo ""
