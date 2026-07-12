---
name: icon-manager
description: 管理前端 Iconify 图标。使用场景："添加图标"、"图标不显示"、"图标报错"。
version: 3.0.0
---

# 图标管理专家

你是专注于 Iconify 集成的前端设计工程师。

---

## 背景

本项目使用 **离线 Iconify**：

1. 组件使用 `@iconify/vue/offline`（不会请求 CDN）
2. 图标集通过 `@iconify-json/*` 本地打包
3. 启动时在 `packages/icons/src/iconify/load.ts` 中 `addCollection` 注册

**禁止**运行时请求 `api.unisvg.com` / `api.iconify.design`。

已安装的图标集（见 `packages/icons/package.json`）：
- `@iconify-json/mdi`
- `@iconify-json/carbon`
- `@iconify-json/ic`
- `@iconify-json/ant-design`
- `@iconify-json/ep`
- `@iconify-json/fluent-mdl2`
- `@iconify-json/lucide`
- `@iconify-json/material-symbols`

---

## 能力一：添加新图标集

### 工作流

1. **安装图标集**
   ```bash
   cd frontend && pnpm add @iconify-json/<set-name> --filter @vben/icons
   ```

2. **注册图标集**

   编辑 `frontend/packages/icons/src/iconify/load.ts`：
   ```typescript
   import { icons as newSet } from '@iconify-json/<set-name>';
   // 在 setupIconifyOffline() 中：
   registerCollection(newSet);
   ```

3. **使用**
   ```vue
   <IconifyIcon icon="prefix:icon-name" />
   ```
   或路由 meta：`icon: 'prefix:icon-name'`

---

## 能力二：修复图标问题

### "图标不显示" 诊断步骤

1. **检查是否离线注册**：`load.ts` 中是否 `addCollection` 对应前缀
2. **检查命名**：`mdi:home` 正确；`mdi-home` 错误
3. **检查入口**：组件应使用 `@iconify/vue/offline` 或 `@vben/icons` 的 `IconifyIcon`
4. **检查控制台**：若出现 `api.unisvg.com` 请求，说明某处仍引用了在线 `@iconify/vue`

### 常见问题

| 问题 | 原因 | 解决方案 |
|------|------|---------|
| 图标空白 | 图标集未注册 | 在 `load.ts` 中 addCollection |
| Console 请求 unisvg/iconify | 使用了在线 Icon 组件 | 改为 `@iconify/vue/offline` |
| TypeScript 报错 | 图标集未安装 | `pnpm add @iconify-json/* --filter @vben/icons` |

---

## 导航速查

| 功能 | 路径 |
|------|------|
| **离线注册** | `frontend/packages/icons/src/iconify/load.ts` |
| **图标组件封装** | `frontend/packages/@core/base/icons/src/` |
| **图标依赖** | `frontend/packages/icons/package.json` |
| **启动钩子** | `frontend/apps/src/bootstrap.ts`（import `@vben/icons`） |
