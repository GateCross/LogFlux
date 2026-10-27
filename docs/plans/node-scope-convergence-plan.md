# 节点能力收口方案

> 日期：2026-10-27
> 基线提交：`3498d4b` fix: 访问/错误日志只按 logger 命名空间分流
> 范围：`backend/internal/svc`、`backend/internal/logic/caddy`、`frontend/apps/src/views/caddy/{catalog,config,certificates,settings}`
> 触发：站点页布局被节点相关的两张卡片占据，真正的站点列表被挤到首屏之外

---

## 1. 结论先行

**节点数据层不动，节点在界面上占的地方全部收口；同时修掉一个会绊倒新用户的旧缺陷。**

| 动作 | 内容 | 改动量 |
|---|---|---|
| P0a | 首次启动自动建默认 `local` 节点（一次性标记键，删光后不重建） | 后端 ~30 行 + 单测 |
| P0b | 修默认 `:80` 入口块模板的悬空 import，并让 `waf_protect` / `geo_auth` 成为系统保证存在的 snippet | 后端 ~80 行 + 单测 |
| P1 | 「节点状态」卡片改为多节点才显示 | 1 行 |
| P2 | 节点选择器单节点时隐藏 | 4 行 |
| 不做 | 表结构、`serverId` 路径契约、`remote` 支持、节点增删改入口、从 `/etc/caddy/Caddyfile` 播种配置 | — |

改完后站点页顶部只剩一条工具栏，站点列表直接上移一屏。

### 置信度轨迹（Jev）

| 轮次 | 方案 | `go` 概率 | 被指出的漏洞 |
|---|---|---|---|
| 1 | P0 + P1(删卡) + P2 | 0.70 | P1 删卡会丢信息（0.48） |
| 2 | P0 + P1(条件渲染) + P2 | 0.70 | P0a 只解决「没有节点」，没解决「没有站点」 |
| 3 | 拆出 P0b + 否决配置播种 | 0.80 | `ensureWafProtectSnippet` 会误覆盖用户配置 |
| 4 | 新增 `ensureSnippetDefinition` + `geo_auth` 条件化 | **0.91** | — |

每一轮都是 Jev 先否掉一个我以为是「显然对的」做法：

- 原打算删掉状态卡 → 改成条件渲染
- 原打算复用 `ensureWafProtectSnippet` → 它会把 include 写法的 WAF 配置静默降级成只检测，改成「存在即不碰」
- 原打算把 `geo_auth` 一起内联 → 它受 `ip_region` 开关控制，而模板是 `//go:embed` 常量拿不到 db，改成条件 import

---

## 2. 为什么不删节点

`serverId` 不是纯 UI 概念，它贯穿四个业务域的唯一键：

| 域 | 键 | 位置 |
|---|---|---|
| 证书库存 | `(server_id, primary_domain)` | `model/caddy/certificate.go:34` |
| 日志源 | `(path, server_id)` | `model/ingest/log_source.go:22` |
| WAF 策略 | `server_id` | `waf_policy_*` |
| 配置历史/回滚 | `server_id` | `caddy_config_history` |

后端 22 个非测试文件、前端 33 个文件（含 5 个测试）引用它。砍掉数据层等于重写半个 Caddy 域，收益抵不上风险。

**但界面上的节点 UI 是 5 处重复表达同一件事**，而单容器部署下它恒等于「就一个 local」：

| 位置 | 单节点时的实际作用 |
|---|---|
| 站点页工具栏 `local` 下拉 | 空操作 |
| 站点页「节点状态」卡（216 行） | 需手动点探测，不点永远是「未探测」 |
| 配置工作台工具栏下拉 | 空操作 |
| 证书中心筛选下拉 | 空操作 |
| 全局设置 WAF 卡下拉 | 空操作 |

---

## 3. 真正的病：首次部署门槛

`caddy_servers` 表**没有种子数据**。`NewServiceContext` 只做 AutoMigrate（`service_context.go:68`），`AddCaddyServerLogic` 只在用户手动提交时执行。

所以新部署的人打开站点页看到的是：

```
请选择 Caddy 节点以查看站点
```

而 Docker 部署本来就只有一个容器，Caddy 就在旁边 `localhost:2019`。用户必须先知道这个地址、手动添加一个「服务器」，产品才能用。这是当前最贵的一处摩擦。

---

## 4. P0a：首次启动自动建默认节点

### 4.1 位置

`backend/internal/svc/service_context.go`，在 `initCronWorkspace(&c)` 之后（第 112 行）加一行调用。

### 4.2 实现

新增 `ensureDefaultCaddyServer(db)`：

```
常量 defaultCaddyServerName    = "local"
常量 defaultCaddyServerURL     = "http://localhost:2019"
常量 systemConfigKeyNodeSeeded = "caddy_node_seeded"
```

逻辑：

1. `system_configs` 读 `caddy_node_seeded`，键存在 → 直接返回（**幂等，且删光后不重建**）
2. `caddy_servers` count > 0 → 只写标记键，不插数据（覆盖老部署升级）
3. count == 0 → 插入 `{Name:"local", Url:"http://localhost:2019", Type:"local", Modules:"{}"}`，再写标记键

### 4.3 两个必须注意的点

| 点 | 原因 |
|---|---|
| `Modules` 必须写 `"{}"` | 模型声明是 `gorm:"type:jsonb"`，空串写入 Postgres 会报错（`caddy_server.go:18`） |
| 标记键必须在「插入」之外单独写 | 否则用户删光节点重启会被自动重建（Jev 判定这是本方案最大风险，0.59；标记方案 0.95） |

### 4.4 为什么不用 config.yaml 配置 URL

`localhost:2019` 在 Docker（同容器）和本地开发（`go run` + 本地 Caddy）下都成立，没有第二个取值。加配置项是为一个永不变化的值引入可配置性。

### 4.5 单测

`backend/internal/svc/service_context_test.go`（新建），sqlmock 三例：

| 场景 | 期望 |
|---|---|
| 标记键存在 | 不查 count、不插入 |
| 标记键不存在 + count=1 | 只写标记，不插入 |
| 标记键不存在 + count=0 | 插入 1 条 + 写标记 |

---

## 5. P0b：修默认 `:80` 入口块模板的悬空 import

### 5.1 这个缺陷为什么必须在本方案里修

P0a 让新用户第一次打开站点页不再空态，**下一步就是点「添加站点」**。而那条路径现在必定失败：

```
1. SiteCreateModal.handleConfirm
     → previewCaddyConfigApi（只跑 Prepare + adapt，不跑 mergeManagedBlocks）
     → 预览通过 ✓
2. pushCaddyConfigApi → UpdateCaddyConfig → Apply
     → mergeManagedBlocks → mergeEntryBlock 用 default_entry_block.caddyfile 无条件替换 :80 块
     → adapt → ✗ 失败
```

`default_entry_block.caddyfile` 第 12-14、31-34 行 import 了六个 snippet：

| import | 是否在仓库中定义 | 处理 |
|---|---|---|
| `waf_protect` | ✗（由 `ensureWafProtectSnippet` 在保存路径注入） | P0b(a) 提到合并层保证 |
| `compression` | ✗ | 内联成 `encode zstd gzip` |
| `subdomain-log` | ✓（`default_log_snippet.caddyfile`，`mergeCaddyLogSnippet` 注入） | 无需改动 |
| `health_check` | ✗ | 内联 |
| `geo_auth` | ✗ | 条件 import（见 5.2b） |
| `api_proxy` | ✗ | 内联 |
| `frontend_simple` | ✗ | 内联 |

除 `subdomain-log` 外的业务 snippet 在全仓都搜不到定义——它们只作为作者生产配置的样本存在于 `testdata_live_caddyfile.txt`。Caddy 对未定义的 snippet import 按文件 glob 处理，无匹配即 adapt 报错。

结果：**预览通过、保存失败，报错是一句晦涩的 snippet not found**。P0a 把这个坑从「用户摸不到」变成「新用户第一下就撞」。

### 5.2 修法（两步）

**(a) 让 `waf_protect` 成为系统保证存在的 snippet**

在 `caddy_config_apply_service.go` 的 `mergeCaddyManagedBlocks`（第 211-223 行）末尾、`mergeCaddyLogSnippet` 之后追加：

```go
merged, err = ensureSnippetDefinition(merged, "waf_protect", renderWafProtectSnippet(detectCaddyNewline(merged)))
if err != nil {
    return "", err
}
```

**不能直接用现有的 `ensureWafProtectSnippet`**（`waf_integration_helpers.go:124`）。它的判定是「含 `coraza_waf` 且含 `` directives ` `` 才认为已就绪，否则**整块替换**成 DetectionOnly 默认模板」。用户若把 directives 写成 include 文件形式（不含反引号），会被静默降级成 `SecRuleEngine DetectionOnly`——WAF 从拦截变只检测，这是不可接受的副作用。

新增 `ensureSnippetDefinition(config, name, template string) (string, bool, error)`，只做「缺失才补、存在就原样返回」：

```go
for _, block := range blocks {
    if block.Kind == "snippet" && block.Name == name {
        return config, false, nil  // 存在即不动，内容归用户所有
    }
}
// 不存在 → 按原 insertPayload 逻辑插到第一个 site 块之前
```

比原来的 `ensureWafProtectSnippet` 收敛一点：**只看块名，不看块内容**。既然存在就不碰，那内容检查没有意义，反而正是它导致误替换。

两个调用点：
- `mergeCaddyManagedBlocks` 末尾补 `ensureSnippetDefinition(merged, "waf_protect", renderWafProtectSnippet(detectCaddyNewline(merged)))`
- `geo_auth` 同理，条件化调用（见 5.2b）

放在 `mergeEntryBlock` **之后**调用——模板里的 `import` 在那一步才落进配置。

与同层的 `mergeCaddyLogSnippet` 对称：后者已在 `mergeCaddyManagedBlocks` 里无条件调用，`subdomain-log` 因此恒有定义；`waf_protect` 缺了同一层保证。

**(b) 把模板里四处业务 snippet 改为内联**

重写 `default_entry_block.caddyfile`，`health_check` / `api_proxy` / `frontend_simple` 三处 import 换成内联块，内容照抄 `renderManagedCaddyfile`（`managed_caddyfile.go:87-99`）已有的 `handle /api/*` 与 `handle` file_server 写法。

`geo_auth` 单独处理，不能内联：它的内容受 `ip_region` 配置开关控制，且 `isIPRegionEnabled(db)`（`managed_caddyfile.go:212`）只在 `renderManagedCaddyfile` 路径里读库——入口块模板是 `//go:embed` 常量，拿不到 db。

实现落在 `caddyConfigApplyService`（它持有 `svcCtx.DB`）：

```
mergeManagedBlocks:
  entryBlock := loadCaddyEntryBlock(db)
  if !isIPRegionEnabled(db) {
      entryBlock = 删掉 "import geo_auth" 那一行
  }
  mergeCaddyManagedBlocks(config, opts, entryBlock, logSnippet)
  → 若 isIPRegionEnabled(db) 为真，再 ensureSnippetDefinition(..., "geo_auth", 模板)
```

`(geo_auth)` 模板内容照抄 `testdata_live_caddyfile.txt:42-51` 的 forward_auth 写法。

**`import geo_auth` 必须留在 `route @lan` 内的 `handle {}` 里，不能提到 `:80` 顶层。** Caddy 的指令顺序中 `forward_auth` 排在 `route` 之前，提到顶层会让 pve/ikuai 这些内网源站的流量也被 geo 校验拦下——那是行为回归，不是等价改写。

模板从此只 import 系统自身保证的 snippet：`waf_protect`、`subdomain-log`、`geo_auth`（条件）。

### 5.3 顺带影响

`entry_block_merge_test.go:86` 的断言要求模板含 `import frontend_simple`，需同步改为检查内联后的 `file_server` 与 `@lan`。

### 5.4 单测

`backend/internal/logic/caddy/entry_block_merge_test.go` 追加三条：

1. **悬空 import 属性测试**：对空配置跑 `mergeCaddyManagedBlocks`，收集结果中所有 `import <name>` 行，断言每个 name 都能在结果里找到 `(name) {` 定义。这是能拦住同类回归的最小检查——以后谁往模板里加业务 snippet 而忘了定义，测试直接红。
2. **`ensureSnippetDefinition` 不覆盖用户内容**：造一份含 `(waf_protect) { ... include 写法 ... }` 的配置，跑一遍后断言该块字节级不变。
3. **`geo_auth` 跟随 `ip_region` 开关**：开关关时结果中无 `import geo_auth` 也无 `(geo_auth) {`；开关开时两者都在。

### 5.5 已知取舍

| 取舍 | 说明 |
|---|---|
| `waf_protect` 在应用入口块时被注入 | 实现上加了 `entryBlock != ""` 条件：空模板表示不接管入口块，此时不注入。生产路径 `entryBlock` 来自 `loadCaddyEntryBlock` 恒非空，故实际等价于总会注入。即使 WAF 关闭也会多出一个没人引用的 `(waf_protect)` 定义——Caddy 允许未引用 snippet，无运行时影响 |
| 已存在的 `waf_protect` 一律不碰 | 包括写坏的。这条路径不做修复，只保证「引用必有定义」这个最低契约 |

---

## 6. P1：「节点状态」卡片改为多节点才显示

### 6.1 为什么不直接删

Jev 对「删掉后信息丢失」给 0.48（近乎五五）。卡片确实独有两样东西：

- 多节点**总览网格**（一屏看全部节点在线情况）
- `lastProbedAt`（最近探测时间）

单节点时这些是纯噪音，多节点时是唯一入口。所以**条件渲染**，不是删除。

### 6.2 改动

`frontend/apps/src/views/caddy/catalog/index.vue` 第 85 行的 `<Card>` 加条件（`StatusActions` 在第 86 行）：

```vue
<Card v-if="servers.length > 1" variant="borderless" class="mb-4">
  <StatusActions ... />
</Card>
```

`StatusActions.vue` 保留（216 行），`useServiceCatalogPage` 的返回值保留。单节点用户看不到它，多节点用户行为不变。

---

## 7. P2：节点选择器单节点时隐藏

四处 `Select` 加同一个条件，判据统一为「选项多于一个」：

| 文件 | 位置 | 判据 |
|---|---|---|
| `catalog/components/EditPanel.vue` | 节点下拉 | `serverOptions.length > 1` |
| `config/components/ServerToolbar.vue` | 节点下拉 | `serverOptions.length > 1` |
| `certificates/index.vue` | 筛选下拉 | `filterOptions.length > 1` |
| `settings/components/SettingsWafCard.vue` | 节点下拉 | `serverOptions.length > 1` |

**保留不动的**：

- 配置工作台的「节点管理」下拉（`ServerToolbar.vue` 的 `nodeMenuItems`）—— 它是增删改节点的唯一入口，删了就没法加远程节点
- 配置工作台工具栏的在线状态 Tag —— P1 的信息转移目标

**证书中心注意**：`filterOptions` 里没有「全部服务器」这一项（`allow-clear` 走 placeholder），所以判据就是 `servers.length > 1`，不要数成 N+1。

---

## 8. 明确否决：从 `/etc/caddy/Caddyfile` 播种初始配置

曾考虑：P0a 播种节点时顺手把镜像里的 `/etc/caddy/Caddyfile` 读进 `caddy_servers.Config`，让首屏直接有内容。

**否决，两个理由**：

1. `caddy_config_source.go:9` 已写明「`/etc/caddy/Caddyfile` 仅作为容器首次启动的 bootstrap 模板，**不作为配置来源**」。那是既有决策，动它等于推翻一条已固化的边界。
2. 该文件里只有一个 `:80` 系统入口块。`sitePrimaryHost([':80'])` 返回空字符串（`service-catalog-utils.test.ts:112` 有断言），播种进去会在站点页长出一张无域名的假站点卡。

Jev 对该否决给 0.94。

---

## 9. 影响面核对

### 9.1 空态分支

P0a 之后，以下三条空态文案在生产环境变为不可达（保留无害，防御性）：

| 文件 | 文案 |
|---|---|
| `catalog/components/StatusActions.vue:51` | 暂无已登记 Caddy 节点 |
| `config/index.vue:118` | 未找到 Caddy 服务器，请先添加节点 |
| `settings/components/SettingsWafCard.vue:268` | 尚无 Caddy 节点 |

不删，因为「用户手动删光节点」这一状态仍然可能到达。

### 9.2 不受影响

- `serverId` 路径参数契约（`/caddy/server/:serverId/config` 等）
- 证书库存 `(server_id, primary_domain)` 合并键
- 日志源 `(path, server_id)` 复合唯一索引
- `remote` 节点支持与其「仅声明」盲区标注
- 站点详情页 `?serverId=&host=` 深链
- 已有部署的 Caddyfile 内容（P0b 只在下次 apply 时注入一个未引用的 `(waf_protect)`）

---

## 10. 已知取舍

| 取舍 | 说明 |
|---|---|
| 用户删光节点后不能靠重启恢复 | 由标记键保证。想恢复就在配置工作台「节点管理 → 添加节点」手动加回 |
| 首次升级的老部署会写一个标记键 | `system_configs` 多一行 `caddy_node_seeded`，无副作用 |
| 老部署下次 apply 会多出 `(waf_protect)` 定义 | 未引用，无运行时影响 |
| 单节点用户永远看不到节点状态卡 | 在线与否由配置读取成功/失败体现（`configErrorMessage` 的 Alert），延迟信息在工具栏 Tag 上 |

---

## 11. 验证

```bash
# 后端单测
cd backend && go test ./internal/svc/ -run TestEnsureDefaultCaddyServer -v
cd backend && go test ./internal/logic/caddy/ -run 'TestMergeManagedBlocks|TestMergeEntryBlock' -v

# 前端单测（P1/P2 只改模板条件，跑既有套件确认未破）
cd frontend && pnpm test:unit -- caddy

# 类型检查
cd frontend && pnpm check:type
```

手动验证（按顺序，每步都应是绿的）：

1. 清空 `caddy_servers` + 删除 `system_configs` 里 `caddy_node_seeded`，重启 → 自动出现 `local` 节点，站点页无空态
2. 点「添加站点」建一个站 → **预览通过且保存成功**（这是 P0b 的验收点，改之前这一步必失败）
3. 删掉 `local` 节点，重启 → **不再重建**，站点页回到空态
4. 手动加第二个节点 → 节点选择器与状态卡同时出现
