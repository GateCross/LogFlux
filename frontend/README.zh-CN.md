<div align="center">
  <h1>LogFlux 前端</h1>
  <p>日志流量分析管理系统 — 前端项目</p>
</div>

## 简介

LogFlux 提供实时流量监控、Caddy 站点与配置管理、WAF 防火墙、证书中心、定时任务、通知管理等功能。

前端基于 [vue-vben-admin v5](https://github.com/vbenjs/vue-vben-admin) 构建，采用 Vue 3 + antdv-next + TypeScript 技术栈，pnpm + Turborepo monorepo 结构。

## 技术栈

- **框架**: Vue 3.5 + TypeScript 6
- **UI 组件库**: antdv-next（Ant Design Vue）
- **构建工具**: Vite 8
- **状态管理**: Pinia 3
- **路由**: Vue Router 5（静态定义，无后端下发菜单）
- **数据请求**: Vue Query + 自封装 RequestClient
- **图表**: ECharts
- **国际化**: Vue I18n
- **CSS**: Tailwind CSS 4
- **包管理**: pnpm（monorepo + Turborepo）

## 功能模块

| 菜单 | 路由 | 说明 |
|---|---|---|
| 仪表盘 | `/dashboard` | 请求趋势、状态码分布、Top 站点、地理分布地图 |
| 站点 | `/caddy/catalog` | 一步建站、站点卡片与详情、能力开关（WAF/IP 限制/安全头/日志/错误页） |
| 安全事件 | `/logs/security` | WAF 拦截与检测事件查询、统计 |
| 证书中心 | `/caddy/certificates` | 自动签发与上传证书合并、缺口视图、刷新 |
| 访问控制 | `/caddy/access` | IP 区域访问控制（ip2region） |
| 全局设置 | `/caddy/settings` | 全局选项、入口块、日志 snippet、WAF 卡片、WAF 源管理 |
| 日志 | `/logs/access`、`/logs/system` | 访问日志与系统日志分开查询 |
| 定时任务 | `/cron` | Cron 任务管理与执行日志 |
| 通知 | `/notification/*` | 通知渠道、规则、模板、发送日志 |
| 系统 | `/system/user` | 用户管理（无角色、无权限体系） |

配置管理页（`/caddy/config`：分块工作台、原文编辑器、历史回滚）路由保留但已从菜单隐藏，入口在「全局设置 → 高级配置」。

## 快速开始

```bash
# 安装依赖
pnpm install

# 启动开发服务器（默认 9527，代理到后端 8888）
pnpm dev:antd

# 构建生产版本
pnpm build:antd

# 类型检查
pnpm -F logflux-web run typecheck

# 单元测试
pnpm test:unit
```

## 项目结构

```
frontend/
├── apps/
│   └── src/               # 主应用源码
│       ├── api/           # 接口层（core/、caddy/、system/、根级域文件）
│       ├── router/        # 静态路由定义 + 登录守卫
│       ├── store/         # Pinia
│       ├── views/         # 业务页面
│       ├── locales/       # 国际化
│       └── utils/         # apiErrorMessage 等工具
├── packages/              # 共享包（@core、effects、stores、hooks、utils、styles、types 等）
└── internal/              # 内部工具包（lint-configs、vite-config、tsconfig 等）
```

## 开发约定

- 响应体 `transform` 已返回 `response.data.data`，业务代码**不要再写** `res.data.xxx`
- 错误文案统一用 `src/utils/api-error-message.ts` 的 `apiErrorMessage`
- 远程数据优先 Vue Query，query key 集中在 `src/api/query-keys.ts`
- 用户可见文案必须中文化；表格/表单/弹窗必须处理 loading、空数据、错误态
- 路由在 `src/router/routes/modules/`，`hideInMenu: true` 表示路由保留但不出现在菜单

完整开发规范见 [`../contexts/context.md`](../contexts/context.md)。

## 浏览器支持

支持现代浏览器，不支持 IE。

| Edge | Firefox | Chrome | Safari |
| :-: | :-: | :-: | :-: |
| last 2 versions | last 2 versions | last 2 versions | last 2 versions |
