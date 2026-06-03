<div align="center">
  <h1>LogFlux 前端</h1>
  <p>日志流量分析管理系统 — 前端项目</p>
</div>

## 简介

LogFlux 是一个日志流量分析管理系统，提供实时流量监控、Caddy 反向代理配置管理、WAF 防火墙、定时任务、通知管理等功能。

前端基于 [vue-vben-admin v5](https://github.com/vbenjs/vue-vben-admin) 构建，采用 Vue 3 + Ant Design Vue + TypeScript 技术栈。

## 技术栈

- **框架**: Vue 3 + TypeScript
- **UI 组件库**: Ant Design Vue 4
- **构建工具**: Vite
- **状态管理**: Pinia
- **路由**: Vue Router 5
- **图表**: ECharts 6
- **国际化**: Vue I18n
- **CSS**: Tailwind CSS 4
- **包管理**: pnpm (Monorepo)

## 功能模块

- **仪表盘**: 实时流量统计、QPS 趋势图、地理分布地图（国内/国际）、错误统计、实时日志
- **Caddy 配置**: 分块配置工作台、原始 Caddyfile 编辑器、配置预览与发布、历史版本管理
- **WAF 防火墙**: SimpleWaf 配置、CRS 规则管理、IP 区域访问控制
- **安全策略**: 观测日志、绑定管理、排除规则、CRS 调优
- **定时任务**: Cron 任务管理与执行日志
- **通知管理**: 通知渠道、规则、模板、发送日志
- **系统管理**: 用户管理、角色管理、菜单管理

## 快速开始

```bash
# 安装依赖
pnpm install

# 启动开发服务器
pnpm dev:antd

# 构建生产版本
pnpm build:antd
```

## 项目结构

```
frontend-next/
├── apps/
│   ├── web-antd/          # 主应用 (Ant Design Vue)
│   │   └── src/
│   │       ├── api/       # API 接口
│   │       ├── views/     # 页面视图
│   │       ├── router/    # 路由配置
│   │       ├── store/     # 状态管理
│   │       └── locales/   # 国际化
│   └── backend-mock/      # Mock 服务器
├── packages/              # 共享包
│   ├── @core/             # 核心包
│   ├── effects/           # 效果包 (插件、hooks 等)
│   └── locales/           # 国际化包
└── internal/              # 内部工具包
```

## 浏览器支持

支持现代浏览器，不支持 IE。

| Edge | Firefox | Chrome | Safari |
| :-: | :-: | :-: | :-: |
| last 2 versions | last 2 versions | last 2 versions | last 2 versions |
