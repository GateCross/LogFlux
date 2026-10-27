# 单实例改造方案：移除「节点」概念

> 日期：2026-10-27
> 基线提交：`4376821` feat: 首次启动播种默认节点并修复入口块悬空 snippet
> 前置方案：`node-scope-convergence-plan.md`（P0a/P0b/P1/P2 已落地）、`node-management-removal-plan.md`（作废，被本文取代）
> 决策：**确定永远只有一个 Caddy 节点**，不存在第二实例
> 状态：**P4-0 ~ P4-4 已全部落地**（见第 13 节实施记录）

---

## 1. 结论先行

**「节点」这个概念从产品里删掉。** 不是删入口，是删概念。

| 阶段 | 内容 | 改动量 |
|---|---|---|
| P4-0 | 配置存储迁移：`caddy_servers.Config/Modules` → `system_configs` | 后端 ~120 行 + 迁移 + 单测 |
| P4-1 | 后端 `serverId` 参数全链路移除（路径、类型、SQL 条件） | 后端 ~70 处 / 22 文件 |
| P4-2 | 前端 `serverId` 参数全链路移除 | 前端 ~195 处 / 28 文件 |
| P4-3 | 删节点管理 UI 与 remote 支持 | 前端 ~150 行 |
| P4-4 | 文案层清掉「节点」字样 | UI 40 处 |

**这是一次大改，不是小修。** 上一轮方案（B 档：只删 UI 留表）的结论是 0.38 低置信度；把「不会再有第二个节点」这个前提补上后，Jev 对全删的判断从 **0.09 跳到 0.85**。

---

## 2. 为什么前提一变，结论就反转

上一轮保留 `serverId` 的唯一理由是「将来可能多节点」。这个理由消失后，剩下的全是成本：

| 现状 | 问题 |
|---|---|
| 五处节点选择器 | 单实例下全是空操作 |
| UI 40 处「节点」字样 | 用户看到「节点」会问「我该配几个？」 |
| 证书表格「所属服务器」列 | 单节点下这一列恒为同一个值 |
| `serverId` 路径参数 | 每个配置接口都要带一个永远等于 1 的数字 |
| `Type` 字段与 remote 分支 | 已有 remote 分支全部是半成品（见第 3 节） |
| `findPreferredCaddyServer` 的 local 优先回落 | 为多节点设计的分支，单实例下恒走第一条 |

Jev 对「概念是否值得删」给 **0.84**（worth_it）。

---

## 3. remote 是半成品（删它的直接依据）

`remote` 只有一种能力：通过 admin API 读写 `/config/` 和 `/load`。其余全部不可用：

| 能力 | local | remote | 位置 |
|---|---|---|---|
| 读/写/预览配置 | ✅ | ✅ | — |
| 证书卷扫描 | ✅ | ❌ 跳过 | `cert_sync_service.go:99` |
| 上传证书 | ✅ | ❌ 直接报错 | `cert_sync_service.go:898` |
| WAF 规则 | ✅ 本地 `/config/security` | ❌ 规则在本地 | `managed_caddyfile.go:27` |
| WAF 面板 | ✅ | ❌ **面板硬查 `type="local"`** | `waf_logic_helpers.go:455` |
| 日志采集 | ✅ `os.Stat` 本地文件 | ❌ 远端文件不存在 | `common/ingest/` |
| IP 区域控制 | ✅ `forward_auth localhost:8888` | ❌ 远端访问不到本机 | — |

**最严重**：`waf_logic_helpers.go:455` 的 `primaryCaddyServer` 硬编码 `Where("type = ?", "local")`。用户界面选中 remote 节点，WAF 页面操作的仍是 local——**界面与操作对象不一致，无任何提示**。

---

## 4. P4-0：配置存储迁移（前置，必须先做）

### 4.1 为什么必须先做

`caddy_servers` 表不只是节点列表，它还存着：

```
Config   text     -- Caddyfile 全文，配置的唯一存储
Modules  jsonb    -- 结构化配置 JSON
```

删表就必须先给配置找个新家。Jev 对「迁到 `system_configs`」的确定性 **0.99**。

### 4.2 为什么迁到 system_configs 而不是新建表

`system_configs` 已经有四个同类单例配置：

| 键 | 内容 |
|---|---|
| `caddy_entry_block` | `:80` 入口模板 |
| `caddy_global_options` | 全局选项 |
| `caddy_log_snippet` | 日志 snippet |
| `caddy_log_retention` | 日志保留天数 |
| `caddy_node_seeded` | 上轮新增的播种标记 |

Caddyfile 全文与它们同类，迁过去与现有模式一致。新建一张 `caddy_config` 表是重复造轮子。

### 4.3 实现

新增 `backend/internal/logic/caddy/caddy_config_store.go`：

```go
const systemConfigKeyCaddyConfig = "caddy_config"

// caddyConfigValue 当前生效配置的存储结构。
type caddyConfigValue struct {
    Config  string `json:"config"`
    Modules string `json:"modules"`
}

// loadCaddyConfig 读取当前生效配置；键不存在时返回空值（首次部署）。
func loadCaddyConfig(db *gorm.DB) (string, string) { ... }

// saveCaddyConfig 写入当前生效配置。
func saveCaddyConfig(db *gorm.DB, config, modules string) error { ... }
```

### 4.4 迁移步骤

在 `service_context.go` 的 AutoMigrate 之后、`ensureDefaultCaddyServer` 之前插入 `migrateCaddyConfigToSystemConfig(db)`：

```
1. system_configs 已有 caddy_config 键 → 直接返回（幂等）
2. caddy_servers 表不存在 或 为空 → 返回
3. 取 id 最小的一行，把它的 Config/Modules 写入 caddy_config
4. 写一个迁移标记 caddy_config_migrated
```

**注意**：迁移只取一行。若老部署真有多个节点（理论上不该有），只保留 id 最小的那个的配置，其余丢弃——这与「单实例」的决策一致，且日志里要明确告警。

### 4.5 老部署的多节点数据怎么处理

启动时检测：

```
若 caddy_servers 行数 > 1 → logx.Errorf 告警
  "检测到 %d 个 Caddy 节点，单实例模式只保留 id=%d 的配置，
   其余节点的配置历史与证书库存将失去关联，请手动备份"
```

不自动删除用户数据，只告警。

---

## 5. P4-1：后端移除 serverId

### 5.1 现状分布

| 类别 | 数量 |
|---|---|
| 路径参数声明（`path:"serverId"`） | 2 处 |
| 纯透传 `req.ServerId` | 12 处 |
| `WHERE server_id` 条件 | 16 处 |

### 5.2 后端已部分支持 `serverId=0`

`findPreferredCaddyServer(db, 0)` 会回落「优先 local，其次任意」，已被 5 处调用：

- `cert_sync_service.go:894`
- `caddy_config_apply_service.go:60`
- `update_caddy_entry_block_logic.go:38`（显式传 0）
- `simple_waf_config_service.go:226`

但另有 5 处是**硬 `First(&server, req.ServerId)`**，传 0 会报「服务器不存在」：

- `preview_caddy_config_logic.go:31`
- `update_caddy_config_logic.go:33`
- `get_caddy_config_logic.go:30`
- `rollback_caddy_config_logic.go:30`
- `caddy_config_apply_service.go:299`

**支持是半成品，这是本次改造必须先统一的地方。**

### 5.3 API 契约改动

`caddy_server.api` 里所有 `ServerId uint \`path:"serverId"\`` 全部删除，路径从 `/caddy/server/:serverId/config` 改为 `/caddy/config`：

| 原路径 | 新路径 |
|---|---|
| `GET /caddy/server/:serverId/config` | `GET /caddy/config` |
| `POST /caddy/server/:serverId/config` | `POST /caddy/config` |
| `POST /caddy/server/:serverId/config/preview` | `POST /caddy/config/preview` |
| `GET /caddy/server/:serverId/config/history` | `GET /caddy/config/history` |
| `GET /caddy/server/:serverId/config/history/:historyId` | `GET /caddy/config/history/:historyId` |
| `POST /caddy/server/:serverId/config/rollback` | `POST /caddy/config/rollback` |
| `GET/PUT/DELETE /caddy/server` 三个 CRUD | **全删** |

改完要跑 `goctl api go` 重新生成 types 与 handler。

### 5.4 数据表的处理

**`caddy_config_history` 保留 `server_id` 列，写 0。**

理由：删列要迁移，且历史数据要保留。`CertificateQuery.ServerID=0` 本来就是「全部服务器」的合法语义（`certificate.go:58`），沿用这个约定。

**但有个坑**：老数据是 `server_id=1`，新代码写 0，历史查询 `WHERE server_id = 0` 会查不到旧记录——**历史断代**。

迁移时必须补一步：

```sql
UPDATE caddy_config_history SET server_id = 0;
UPDATE caddy_certificates SET server_id = 0;
UPDATE log_sources SET server_id = 0;
```

在 `migrateCaddyConfigToSystemConfig` 里一并执行，幂等（重复执行结果相同）。

### 5.5 单测

| 文件 | 用例 |
|---|---|
| `caddy_config_store_test.go`（新建） | 读写往返、键不存在返回空 |
| `service_context_test.go`（追加） | 迁移幂等、多节点告警、历史表 server_id 归零 |
| 既有 5 个 `*_config_logic_test.go` | 去掉 `ServerId` 字段后修正 sqlmock 期望 |

---

## 6. P4-2：前端移除 serverId

### 6.1 现状分布

| 类别 | 数量 |
|---|---|
| API 层路径拼装 | 55 处 |
| views 层 | 139 处 |
| query key | 9 处 |

views 层集中在六个文件：

| 文件 | 处数 |
|---|---|
| `sites/SiteDetail.vue` | 19 |
| `config/composables/useCaddyConfigIO.ts` | 16 |
| `config/composables/useSimpleWaf.ts` | 14 |
| `sites/SiteCreateModal.vue` | 10 |
| `config/composables/useConfigHistory.ts` | 8 |
| `sites/use-site-detail-edit.ts` | 7 |

### 6.2 改造顺序

**先 API 层，再 views 层。** API 层改完，views 层的类型错误会由 `vue-tsc` 逐个指出，照着修即可。

API 层（`api/caddy/server.ts`）：

```ts
// 改前
export async function getCaddyConfigApi(serverId: number, config?: RequestClientConfig)
  → requestClient.get(`/caddy/server/${serverId}/config`, config)

// 改后
export async function getCaddyConfigApi(config?: RequestClientConfig)
  → requestClient.get('/caddy/config', config)
```

**删除**：`getCaddyServerListApi`、`addCaddyServerApi`、`updateCaddyServerApi`、`deleteCaddyServerApi`、`getCaddyServerStatusApi`（节点状态探测在单实例下由配置读取成败体现）。

### 6.3 query key 简化

```ts
// 改前
config: (serverId: number) => ['caddy', 'config', serverId] as const,
catalog: (serverId: number) => ['caddy', 'catalog', serverId] as const,
certificates: (serverId?: number) => ['caddy', 'certificates', serverId ?? 'all'] as const,

// 改后
config: () => ['caddy', 'config'] as const,
catalog: () => ['caddy', 'catalog'] as const,
certificates: () => ['caddy', 'certificates'] as const,
```

### 6.4 站点详情深链

`useServiceCatalogPage.ts:285` 现在传 `query: { serverId, host }`。改为只传 `host`：

```ts
router.push({ name: 'CaddySiteDetail', query: { host } });
```

`SiteDetail.vue:75` 的 `serverId` computed 整个删除，`enabled` 判断从 `Boolean(serverId.value)` 改为 `Boolean(host.value)`。

---

## 7. P4-3：删节点管理 UI 与 remote

| 文件 | 动作 |
|---|---|
| `config/components/ServerToolbar.vue` | 删节点 Select、节点管理下拉、在线状态 Tag；只留配置操作按钮 |
| `config/components/ServerFormModal.vue` | **整个删除**（没有节点可管理了） |
| `config/composables/useCaddyServers.ts` | **整个删除**（215 行） |
| `catalog/components/EditPanel.vue` | **整个删除**（81 行，只是节点工具栏） |
| `catalog/components/StatusActions.vue` | **整个删除**（216 行，多节点总览） |
| `settings/components/SettingsWafCard.vue` | 删节点 Select 与 `advancedServerId` 相关逻辑 |
| `certificates/index.vue` | 删「所属服务器」列、「全部服务器」筛选、`serverNameMap` |
| `certificate-utils.ts` | 删 `isLocalServer` |
| `shared/caddy-server-status.ts` | 整个删除（节点状态纯函数） |
| `shared/use-caddy-server-status.ts` | 整个删除（133 行） |
| `api/caddy/server.ts` | 只留配置相关 API |

**连带删除的测试**：`caddy-server-status.test.ts`、`useCaddyServers.test.ts`、`certificate-utils.test.ts` 里的 `isLocalServer` 块。

---

## 8. P4-4：文案清理

UI 层「节点」40 处。分类处理：

| 场景 | 改法 |
|---|---|
| 「请选择 Caddy 节点」 | 删除（选择器没了） |
| 「当前节点暂无站点」 | 改「暂无站点」 |
| 「节点列表读取失败」 | 删除（无列表了） |
| 「所属服务器」列 | 删除 |
| 「远端（仅声明）」 | 删除 |
| 代码注释里的「节点」 | 改「实例」或直接删（不影响功能，但保持一致性） |

---

## 9. 影响面与风险

### 9.1 数据库迁移风险（最高）

| 风险 | 缓解 |
|---|---|
| 迁移中断导致配置丢失 | 迁移在事务内执行；`caddy_servers` 表**不删**，只做数据搬迁 |
| 历史断代 | 迁移时 `UPDATE ... SET server_id = 0`，与旧数据对齐 |
| 多节点老数据 | 只迁 id 最小的，其余告警不删 |

**`caddy_servers` 表在本次改造中不 DROP。** 留着不影响功能，一旦出问题还能回滚。真要删表是下一轮的事。

### 9.2 无法回退的部分

- API 路径变更（`/caddy/server/:serverId/config` → `/caddy/config`）——老客户端会 404
- 前端删除 6 个组件文件

**建议**：分两次提交。第一次 P4-0（迁移，可回滚），验证通过后再做 P4-1~P4-4。

### 9.3 不受影响

- 站点页、站点详情、访问日志、WAF 面板的业务逻辑（只是少了 serverId 参数）
- 证书库存的 `(server_id, primary_domain)` 唯一键（server_id 恒为 0）
- 日志源的 `(path, server_id)` 唯一键（server_id 恒为 0）

---

## 10. 验证

```bash
# 后端
cd backend && go build ./... && go test ./internal/... 

# 前端
cd frontend && pnpm exec vue-tsc --noEmit -p apps/tsconfig.json
cd frontend && pnpm exec vitest run --dom apps/src/views/caddy
```

手动验证：

1. 老数据库启动 → 配置仍在、历史记录完整、无「节点」字样
2. 站点页 → 无节点选择器、无节点状态卡、站点列表直接可见
3. 点站点卡片进详情 → 能正常读写配置
4. 配置工作台 → 能保存、能预览、能回滚历史
5. 证书中心 → 无「所属服务器」列，上传正常
6. 全局设置 → WAF 面板正常，无节点选择器

---

## 11. 工作量估算

| 阶段 | 文件数 | 预估 |
|---|---|---|
| P4-0 存储迁移 | 3 新增 + 2 改 | 中 |
| P4-1 后端去 serverId | 22 改 | 大 |
| P4-2 前端去 serverId | 28 改 | 大 |
| P4-3 删 UI 与 remote | 11 删 + 4 改 | 中 |
| P4-4 文案 | 分散 | 小 |

**建议用 workflow 分三轨并行**：后端迁移、后端去 serverId、前端去 serverId（前端两条有依赖，需串行）。

---

## 12. 待确认

1. **`caddy_servers` 表留还是删？** 方案建议先留（可回滚），确认稳定后下一轮再删。
2. **老部署若真有多节点数据，只保留 id 最小的那个配置**——是否接受？
3. **API 路径变更会破坏任何现有集成吗？** 若有外部脚本调 `/caddy/server/:id/config`，需要保留兼容路径。

---

## 13. 实施记录（2026-10-27）

P4-0 ~ P4-4 全部落地，后端 `go build ./... && go vet ./... && go test ./...` 全绿，
前端 `vue-tsc --noEmit` 零错误、`vitest run apps/src/views/caddy apps/src/api` 251 用例通过。

### 13.1 与原方案的差异

| 项 | 原方案 | 实际落地 | 原因 |
|---|---|---|---|
| 迁移代码位置 | `internal/logic/caddy/caddy_config_store.go` | `model/caddy/config_store.go` | `svc` 包不能反向依赖 `logic/caddy`（import cycle），迁移只能放在 `svc` 可依赖的 model 层 |
| 配置读写函数 | `loadCaddyConfig` / `saveCaddyConfig`（包内私有） | `caddymodel.LoadConfig` / `SaveConfig`（导出） | 同一原因：`svc` 启动迁移与 `logic/caddy` 都要用 |
| 服务器解析 | `findPreferredCaddyServer(db, serverID)` | `findPreferredCaddyServer(db)` | 参数已无意义，直接删掉；内部仍是「local 优先，其次 id 最小」 |
| 证书上传目录 | 保留 `/config/certs/<serverID>/<域名>/` | 新上传写 `/config/certs/<域名>/` | 单实例下 serverID 恒为 0，多一层 `0/` 无意义；存量文件靠库存 `storagePath` 定位，删除仍正常 |
| 证书声明状态 | `local` 为 missing、`remote` 为 pending | 恒为 missing | 单实例没有 remote，pending 分支不可达 |
| 前端 `ServerToolbar` | 只删节点 Select 与在线状态 Tag | 连带删掉「刷新 / 节点管理」整组按钮 | 节点 CRUD 与状态探测接口已删，按钮点不到东西 |

### 13.2 删除的文件

后端：
`add_caddy_server_handler.go`、`delete_caddy_server_handler.go`、`get_caddy_servers_handler.go`、
`get_caddy_server_status_handler.go`、`update_caddy_server_handler.go`，
`add_caddy_server_logic.go`、`delete_caddy_server_logic.go`、`get_caddy_servers_logic.go`、
`get_caddy_server_status_logic.go`、`update_caddy_server_logic.go`，
以及三个状态探测测试（`get_caddy_server_status_logic_test.go`、
`get_caddy_server_status_preservation_test.go`、`get_caddy_server_status_bug_condition_test.go`）。

前端：
`config/components/ServerFormModal.vue`、`config/composables/useCaddyServers.ts`、
`config/composables/useCaddyServers.test.ts`、`catalog/components/EditPanel.vue`、
`catalog/components/StatusActions.vue`、`shared/caddy-server-status.ts`、
`shared/caddy-server-status.test.ts`、`shared/use-caddy-server-status.ts`。

### 13.3 API 契约变化

| 原路径 | 新路径 |
|---|---|
| `GET /caddy/server/:serverId/config` | `GET /caddy/config` |
| `POST /caddy/server/:serverId/config` | `POST /caddy/config` |
| `POST /caddy/server/:serverId/config/preview` | `POST /caddy/config/preview` |
| `GET /caddy/server/:serverId/config/history` | `GET /caddy/config/history` |
| `GET /caddy/server/:serverId/config/history/:historyId` | `GET /caddy/config/history/:historyId` |
| `POST /caddy/server/:serverId/config/rollback` | `POST /caddy/config/rollback` |
| `GET/POST/PUT/DELETE /caddy/server[/:id]`、`GET /caddy/server/status` | **删除** |
| `POST /caddy/certificates/refresh` 的 `serverId` | 请求体删除，改为无参 |

`CertificateItem` / `SimpleWafConfig` / `WafEngineStatusResp` 的 `serverId` 字段一并删除。
`caddy_config_history` / `caddy_certificates` / `log_sources` 的 `server_id` **列保留**，写入恒为 0。

### 13.4 迁移行为

`caddymodel.MigrateLegacyConfig(db)` 在 `NewServiceContext` 中于 `AutoMigrate` 之后、
`ensureDefaultCaddyServer` 之前执行：

1. `system_configs` 已有 `caddy_config` 键 → 跳过搬迁（幂等）
2. `caddy_servers` 不存在或为空 → 跳过
3. 取 id 最小一行，把 `Config` / `Modules` 写入 `caddy_config`
4. 多节点时 `logx.Errorf` 告警，只保留 id 最小的配置，不删任何数据
5. `caddy_config_history` / `caddy_certificates` / `log_sources` 的 `server_id <> 0` 全部归零
6. 写标记键 `caddy_config_migrated`

`caddy_servers` 表**不删**，配置列保留但运行时不读写，只作迁移来源与回滚依据。

### 13.5 新增与改写的测试

| 文件 | 覆盖 |
|---|---|
| `model/caddy/config_store_test.go`（新建） | 读写往返、键不存在返回空、迁移跳过/单节点搬迁/多节点只取最小 id/缺表跳过 |
| `internal/svc/service_context_test.go`（追加） | 启动顺序：迁移先于播种 |
| `internal/logic/caddy/caddy_config_apply_service_test.go`（改写） | 落库失败回滚运行态、乐观并发的锁外/锁内校验、回滚受 baseHash 保护 |
| `internal/logic/caddy/caddy_config_service_test.go`（改写） | 未接管提示、预览只 /adapt、更新走 adapt→load→history |
| `internal/logic/caddy/cert_sync_service_test.go`（改写） | 去掉 remote/pending 分支，server_id 归零 |
| `frontend type-contract.test.ts`（改写） | 去掉 server/status 类型，改为配置历史与配置模型 |
| `frontend certificate-utils.test.ts`（改写） | 去掉 `isLocalServer` 与按 serverId 过滤 |

### 13.6 清理轮（同日，ponytail 复查）

首轮提交后按 ponytail 复查，清掉改造留下的死代码：

| 位置 | 删除内容 | 依据 |
|---|---|---|
| `logic/caddy/caddy_config_service.go` | `caddyConfigFromServer` | 改造后无人调用，只剩它自己读 `server.Config/Modules` |
| `logic/caddy/caddy_config_source.go` | `loadCaddyConfigValue` | 与 `loadCurrentCaddyConfig` 重复，直接内联 |
| `logic/caddy/caddy_config_apply_service.go` | `verifyBaseHash` | 只是 `checkBaseHash` 的同义包装，调用点改用原名 |
| `logic/caddy/caddy_config_service_test.go` | `caddyConfigJSONFor` | 与 `caddyConfigJSON` 重复，合并成一个 |
| `svc/service_context.go` | 播种时不再写 `Type`/`Modules` 的注释误导 | 见下 |

**一次回退**：曾试图连 `Type`/`Modules` 一起从播种里删掉，但
`Modules` 是 jsonb 列，空串写入 Postgres 会直接报错（`TestEnsureDefaultCaddyServerSeedsLocalNodeWhenEmpty` 当场失败），
`Type` 又参与 `findPreferredCaddyServer` 的 local 优先查询。两列都保留，并补了注释说明原因。

**Jev 复核结论**（三项都问了，置信度是「值得做」的概率）：

| 问题 | 值得做 | 处置 |
|---|---|---|
| 现在 DROP `caddy_servers` 表 | 0.17 | 不做。方案本来就说留一轮，删表不可回滚 |
| 保留 `/caddy/server/:serverId/config` 兼容别名 | 0.28 | 不做。仓库内无外部调用方证据 |
| 把 Admin 地址硬编码、删掉整条读表链路 | 0.56 | 不做。读表只多一次查询，换来的是地址可换（如改 admin 端口），删了反而把可配置性焊死 |

### 13.7 仍未做

- `caddy_servers` 表的 DROP（按方案留待下一轮）
- 老路径 `/caddy/server/:id/config` 的兼容别名（方案第 12 节第 3 条待确认；当前按「直接删」执行）
- `simpleWafCandidate` 的 `LastGood` / `LastModules` / `CaddyChanged` 三个字段无消费方（改造前就这样，非本次引入）


