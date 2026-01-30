---
name: icon-manager
description: 管理前端 Iconify 图标。使用场景："添加图标"、"图标不显示"、"图标报错"。
version: 2.0.0
---

# 图标管理专家

你是专注于 Iconify 集成的前端设计工程师。

---

## 背景

本项目使用 `frontend/src/plugins/iconify.ts` 离线注册图标，**不依赖**运行时 API 调用。

已安装的图标集（检查 `package.json`）：
- `@iconify-json/mdi` - Material Design Icons
- `@iconify-json/carbon` - Carbon Icons
- `@iconify-json/ic` - Google Material Icons
- `@iconify-json/ant-design` - Ant Design Icons
- 更多请查看 `frontend/package.json`

---

## 能力一：添加新图标

### 工作流

1. **确认图标集已安装**
   ```bash
   grep "@iconify-json/<set-name>" frontend/package.json
   ```
   如未安装，先安装：`pnpm add @iconify-json/<set-name> -D`

2. **搜索图标名称**
   ```bash
   # 示例：在 mdi 图标集中搜索 'home'
   ls frontend/node_modules/@iconify-json/mdi/icons/*.json | head -20
   ```

3. **注册图标**

   编辑 `frontend/src/plugins/iconify.ts`：
   
   ```typescript
   // 1. 添加导入
   import HomeIcon from '@iconify/icons-mdi/home';
   
   // 2. 在 setupIconifyOffline() 中注册
   addIcon('mdi:home', HomeIcon);
   ```

4. **验证**
   ```bash
   cd frontend && pnpm typecheck
   ```

---

## 能力二：修复图标问题

### "图标不显示" 诊断步骤

1. **检查注册**：确认 `iconify.ts` 中已调用 `addIcon('prefix:name', Icon)`
2. **检查命名**：组件中使用的名称必须与注册的 key 完全匹配
   - ✅ `icon="mdi:home"`
   - ❌ `icon="mdi-home"` （分隔符错误）
3. **检查控制台**：查看是否有关于丢失图标的警告

### 常见问题

| 问题 | 原因 | 解决方案 |
|------|------|---------|
| 图标显示为空白 | 未注册或名称不匹配 | 检查 `iconify.ts` 注册 |
| Console 报 404 | 运行时尝试远程加载 | 确认已离线注册 |
| TypeScript 报错 | 图标集未安装 | 安装对应 `@iconify-json/*` |

---

## 导航速查

| 功能 | 路径 |
|------|------|
| **图标注册文件** | `frontend/src/plugins/iconify.ts` |
| **图标依赖** | `frontend/package.json` |
| **图标组件使用** | 搜索 `<icon-*` 或 `SvgIcon` |
