# 节点管理收口方案（放弃 remote）

> ⚠️ **本文已作废，被 `single-instance-refactor-plan.md` 取代，不要按本文实施。**
>
> 作废原因：本文的前提是「保留 `serverId`，因为将来可能多节点」。后来确认
> **永远只有一个 Caddy 节点**，前提不成立，结论随之反转——Jev 对「删表与
> `serverId`」的判断从 0.09 升到 0.85。
>
> 本文仍有参考价值的部分：第 3 节的 remote 能力缺口核查、第 8 节的
> `UpdateCaddyServer` 全量覆盖缺陷（新方案同样需要修）。

> 日期：2026-10-27
> 基线提交：`4376821` feat: 首次启动播种默认节点并修复入口块悬空 snippet
> 前置方案：`node-scope-convergence-plan.md`（已落地 P0a/P0b/P1/P2）
> 范围：`frontend/apps/src/views/caddy/config`、`frontend/apps/src/api/caddy`
> 决策：放弃「远程 Caddy 节点」产品目标

---

## 1. 结论先行

**删「添加节点」和「删除节点」，保留「编辑节点」但锁死类型与地址。**

| 动作 | 内容 | 改动量 |
|---|---|---|
| P3a | 删「添加节点」「删除节点」两个入口及其实现 | 前端 ~120 行 |
| P3b | 编辑节点：只能改名称，地址与类型只读 | 前端 ~30 行 |
| P3c | 删 remote 相关的 UI 特判（证书中心四处） | 前端 ~40 行 |
| 不做 | `caddy_servers` 表结构、`serverId` 路径契约、后端 CRUD 接口、`pending` 状态 | — |

---

## 2. 为什么不删到「彻底去节点」

`serverId` 贯穿两个表的唯一键：

| 表 | 键 | 位置 |
|---|---|---|
| `caddy_certificates` | `(server_id, primary_domain)` | `model/caddy/certificate.go:60` |
| `log_sources` | `(path, server_id)` | `model/ingest/log_source.go:22` |

后端 70 处引用 / 22 个文件，前端 195 处 / 28 个文件。

彻底删除无法回退，且 Jev 对这条路径的风险判断是 **1.0（确定高风险）**。而用户的实际痛点是「点了添加按钮得到半残节点」，不是「表里多一个 server_id 列」。

---

## 3. remote 为什么该放弃（事实依据）

`remote` 节点目前只有一种能力：通过 admin API 读写 `/config/` 和 `/load`。其余全部不可用：

| 能力 | local | remote | 代码位置 |
|---|---|---|---|
| 读/写/预览配置 | ✅ | ✅ | — |
| 证书卷扫描 | ✅ | ❌ 跳过 | `cert_sync_service.go:99` |
| 上传证书 | ✅ | ❌ 直接报错 | `cert_sync_service.go:898` |
| WAF 规则 | ✅ 读本地 `/config/security` | ❌ 规则在本地，推过去失效 | `managed_caddyfile.go:27` |
| WAF 面板 | ✅ | ❌ 面板硬查 `type="local"` | `waf_logic_helpers.go:455` |
| 日志采集 | ✅ `os.Stat` 读本地文件 | ❌ 远端文件本机不存在 | `common/ingest/` |
| IP 区域访问控制 | ✅ `forward_auth localhost:8888` | ❌ 远端访问不到本机后端 | — |

**最严重的一条**：`waf_logic_helpers.go:455` 的 `primaryCaddyServer` 硬编码 `Where("type = ?", "local")`。用户在界面上选中 remote 节点，WAF 页面操作的仍然是 local 节点——**界面显示与操作对象不一致，且没有任何提示**。

全仓也没有多节点部署痕迹：`docker-compose.yml` 单容器，`managed_caddyfile.go` 里 `admin localhost:2019` / `/app/frontend` / `/config/security` 全是容器内绝对路径，`service_context.go:181` 启动时只给 `type="local"` 补日志源。

---

## 4. P3a：删「添加节点」与「删除节点」

### 4.1 入口

`config/components/ServerToolbar.vue:51` 的 `nodeMenuItems` 当前 5 项：

```
探测当前节点 | ─── | 添加节点 | 编辑节点 | 删除节点
```

改为 2 项：

```
探测当前节点 | ─── | 编辑节点
```

保留「探测当前节点」——它是手动探测的唯一入口，P1 的条件渲染只影响站点页那张总览卡。

### 4.2 连带删除

| 文件 | 删除内容 |
|---|---|
| `ServerToolbar.vue` | `nodeMenuItems` 的 add/delete 两项、`handleNodeMenuClick` 里对应分支、`openAdd`/`deleteServer` 两个 emit、`ServerFormModal` 的 `type="add"` 分支 |
| `useCaddyServers.ts` | `openAddServerModal`、`deleteCurrentServer`、`saveServer` 里的 add 分支 |
| `api/caddy/server.ts` | `addCaddyServerApi`、`deleteCaddyServerApi` 两个函数（后端接口保留，只是前端不再调用） |
| `useCaddyConfigPage.ts` | `openAddServerModal`、`deleteCurrentServer` 的转发与解构 |
| `config/index.vue` | 空态里的「添加服务器」按钮改为提示「请在配置文件中确认 Caddy admin 地址」，因为节点已由后端自动播种 |

### 4.3 空态怎么处理

P0a 已自动播种 `local` 节点，所以 `config/index.vue:118` 的「未找到 Caddy 服务器，请先添加节点」在生产环境不可达。保留该分支作为防御，但按钮改为不可点击的说明文字。

---

## 5. P3b：编辑节点只改名称

### 5.1 为什么保留编辑

`caddy_servers.Name` 有 `uniqueIndex`（`caddy_server.go:11`），且节点名会出现在站点页、证书中心、WAF 面板的展示里。改个名字是合理需求。

### 5.2 为什么锁死地址与类型

地址是 `http://localhost:2019`——改错了整个产品失效，且用户没有理由改。类型只有 `local` 一个合法值了。

### 5.3 改动

`ServerFormModal.vue`（51 行）改为单用途：

| 字段 | 处理 |
|---|---|
| 名称 | 可编辑 |
| 地址 | `disabled`，并加提示「Caddy admin 地址，不可修改」 |
| 类型单选（本地/远程） | **删除**，表单不再有 `type` 字段 |
| 凭证（仅 remote 显示） | **删除** |

Modal 标题固定为「编辑节点」，去掉 `type === 'add'` 三元。

`useCaddyServers.ts` 的 `saveServer` 只保留 update 分支，payload 里不再传 `type`/`token`。

---

## 6. P3c：删 remote 的 UI 特判

`type` 字段在数据层保留（后端仍写 `"local"`），但前端不再对它做分支判断。

| 文件 | 改动 |
|---|---|
| `certificates/index.vue:178-180` | `uploadServerOptions` 去掉「（远端，无文件通道）」后缀与 `disabled` |
| `certificates/index.vue:184` | 删 `hasLocalServer` computed |
| `certificates/index.vue:191` | `openUpload` 里 `find(isLocalServer)` 改回 `servers.value[0]` |
| `certificates/index.vue:358-366` | Modal 去掉 `:ok-button-props` 的 disabled 与「没有可上传的本地服务器」Alert |
| `certificates/index.vue:277` | Tooltip 文案去掉「（remote 服务器仅同步声明）」 |
| `certificates/index.vue:46,51` | 删 `isLocalServer` import 与 remote 注释 |
| `certificate-utils.ts:97-100` | 删 `isLocalServer` 函数 |
| `certificate-utils.test.ts:116-121` | 删 `isLocalServer` 测试块 |
| `api/caddy/server.ts:25` | 注释「默认 local；local \| remote」→「固定 local」 |
| `api/caddy/certificate.ts:26,76` | 删 remote 相关注释 |

---

## 7. 保留不动

| 项 | 原因 |
|---|---|
| `caddy_servers` 表与 `Type` 字段 | 删字段要迁移，收益为零 |
| 后端 `AddCaddyServer` / `UpdateCaddyServer` / `DeleteCaddyServer` 三个 API | 前端不调用即可，删接口要改路由与 types |
| `CertStatusPending` 状态 | `certStatusAndDaysLeft` 里 `notAfter.IsZero()` 仍会返回它（扫描失败、声明无真值）。Jev 对该项 0.97 |
| `certMergeInput.ServerType` 与 `certDeclaredStatus(isLocal)` | 后端保留双分支，只是不再有 remote 行。删它要动合并逻辑，风险大于收益 |
| `findPreferredCaddyServer` 的 local 优先回落 | 单节点下等价，留着无害 |

**`pending` 状态保留的理由**：它不只服务 remote。`cert_sync_service.go:543` 的 `notAfter.IsZero()` 分支在卷扫描失败、声明行无真值时都会命中。删掉它会让这些情况误报成 `missing`（本机声明了 HTTPS 但没证书文件），语义反而更错。

---

## 8. 后端必须先修一处：`UpdateCaddyServer` 会清空未传字段

### 8.1 问题

`api/caddy_server.api:25-33` 把六个字段全标了 `optional`：

```
Name / Url / Token / Type / Username / Password  全部 json:",optional"
```

但实现是**无条件全量覆盖**（`update_caddy_server_logic.go:37-43`）：

```go
server.Name = req.Name
server.Url = req.Url
server.Token = req.Token
server.Type = req.Type
server.Username = req.Username
server.Password = req.Password
```

也就是说：**只传 `name` 调用这个接口，`url` 会被写成空串——节点当场失效，且返回 200。**

`optional` 在这份 API 契约里的含义是「可以省略」，实现却按「省略即清空」处理。两者矛盾。

### 8.2 为什么 P3b 会踩中

P3b 要把编辑表单收敛成「只改名称」，最自然的写法就是只传 `name`。不改后端就会把 `url` 清空。

### 8.3 修法

`update_caddy_server_logic.go` 改为「非空才覆盖」：

```go
if strings.TrimSpace(req.Name) != "" {
    server.Name = req.Name
}
if strings.TrimSpace(req.Url) != "" {
    server.Url = req.Url
}
if strings.TrimSpace(req.Type) != "" {
    server.Type = req.Type
}
// token/username/password 同理，但要允许显式清空——
// 这三个是凭据，用户可能真想清掉，改成"传了就覆盖、没传就保留"
```

凭据三字段与展示字段的处理不同，需要用指针区分「未传」与「传了空串」。若不改 `types.go`，一个更省的近似：**凭据字段只在非空时覆盖**，代价是清不掉。

**取舍**：单节点 + local 场景下凭据根本用不上（`postCaddyText` 只在 `server.Token != ""` 时带 header）。所以「只在非空时覆盖」够用，不为它改 types。

### 8.4 单测

`update_caddy_server_logic_test.go`（新建），sqlmock 两例：

| 场景 | 期望 |
|---|---|
| 只传 `name` | 生成的 UPDATE 语句里 `url` / `type` 仍是原值 |
| 传 `name` + `url` | 两个都更新 |

---

## 9. 是否要改后端其余部分

**不改。** 理由：

1. 三个 CRUD 接口保留，前端不调用即可。将来真要恢复 remote，后端是现成的。
2. `Type` 字段默认值 `'local'`，P0a 播种时写入 `local`。修完 8.3 后，前端不传 `type` 也不会清空。
3. `CertStatusPending`、`certMergeInput.ServerType`、`certDeclaredStatus(isLocal)` 全部保留（见第 7 节）。

---

## 10. 影响面核对

### 10.1 不受影响

- 站点页、站点详情、访问日志、WAF 面板（都不碰 `type`）
- 证书中心的列表与刷新（只删了上传的 remote 特判）
- `serverId` 路径参数契约
- 已有部署：`type` 字段仍是 `local`，行为不变

### 10.2 唯一的行为变化

用户不能再添加第二个节点。单容器部署下这本来也没有意义——`docker-compose.yml` 里只有一个 Caddy。

---

## 11. 已知取舍

| 取舍 | 说明 |
|---|---|
| 删除入口后无法再加节点 | 想恢复要改代码。单容器产品下这是对的默认 |
| 后端仍保留能创建 remote 的 API | 通过 API 直接调仍可造出半残节点。但 UI 不再引导，且这类用户能自己看文档 |
| `pending` 状态保留但注释过期 | 需同步改 `certificate.go:27` 的注释，从「远端服务器仅有 Caddyfile 声明」改为「无真值（零值有效期）」 |
| 老部署里已有的 remote 节点 | 不会被自动清理。若有，前端不再特判，其证书行会按 local 逻辑显示为 missing |

---

## 12. 验证

```bash
cd frontend
pnpm exec vue-tsc --noEmit -p apps/tsconfig.json
pnpm exec vitest run --dom apps/src/views/caddy/certificates apps/src/views/caddy/config
```

手动验证：

1. 配置工作台工具栏「节点管理」下拉只剩「探测当前节点」和「编辑节点」
2. 点「编辑节点」→ 弹窗标题「编辑节点」，名称可改，地址置灰，无类型单选、无凭证字段
3. 改个名称保存 → 列表与站点页展示同步更新
4. 证书中心 → 点「上传证书」→ 目标服务器下拉无「远端」标注、无禁用项
5. 清空 `caddy_servers` 重启 → 自动播种 `local`，站点页可用
