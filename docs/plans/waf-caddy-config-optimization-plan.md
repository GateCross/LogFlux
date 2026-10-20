# WAF 配置与 Caddy 编辑逻辑梳理及优化方案

> 更新时间：2026-10-20
> 范围：`backend/internal/logic/caddy/`、`frontend/apps/src/views/caddy/config/`、`docker/Caddyfile`
> 参照：Caddy Admin API、coraza-caddy、OWASP CRS 4.x 官方文档、SafeLine / BunkerWeb 产品形态

---

## 0. 摘要：必须先修的正确性缺陷

| # | 缺陷 | 实际后果 |
|---|---|---|
| P0-1 | 策略指令写的是 `tx.paranoia_level`，CRS 4.x 读的是 `tx.blocking_paranoia_level` | **PL 恒为 1**，规则集永远停在最低档；"平衡 / 严格"实际只改了异常阈值，没有启用 PL2/PL3 的规则 |
| P0-2 | Host 头以 IP 开头的请求 `ctl:ruleEngine=Off` | 默认 `:80` 是全站 catch-all，**任意客户端构造 `Host: 1.2.3.4` 即可跳过全部规则** |
| P0-3 | `/api/` 全路径 `ctl:ruleEngine=Off` | 管理面（含登录）无任何 WAF 保护，且用户无法在 raw 模式删除（每次保存被重新注入） |

量化依据（对 `coreruleset` v4.25.0 全量规则集统计）：

| 检查项 | 结果 |
|---|---|
| 裸 `tx.paranoia_level` 引用 | **0 次** |
| `TX:BLOCKING_PARANOIA_LEVEL` | 16 次 |
| `TX:DETECTION_PARANOIA_LEVEL` | 200 次 |
| 官方 `id:900000/900001/900100/900110` | 默认全部注释掉 |

需要说清楚 P0-1 的边界：**强度控件的阈值部分是生效的**（low_fp 10/8 → balanced 5/4 → high_blocking 3/2 确实改变拦截灵敏度），失效的只有 PL 维度。因此准确表述是"强度只调阈值、不调规则集"，而不是"强度完全无效"。

`REQUEST-901-INITIALIZATION` 中 901120 在 `blocking_paranoia_level` 未定义时兜底为 1，901125 令 detection = blocking，所以从日志看不出异常——PL2/PL3 的 119 条规则（全量统计：PL1 213 条、PL2 89、PL3 30、PL4 9）从未被启用。

---

## 1. 现状：两条写入链路

```
                       ┌──────────────────────────────────────────┐
   前端「防火墙」tab ──►│ PUT  /caddy/waf/simple-config   (只写库) │
                       │ POST /caddy/waf/simple-config/preview    │
                       │ POST /caddy/waf/simple-config/apply      │──► adapt ─► load ─► 落库
                       └──────────────────────────────────────────┘
                                       ▲
                                       │ 读库 policy + 读 Caddyfile 文本
                       ┌───────────────┴──────────────────────────┐
   前端「配置管理」  ──►│ POST /caddy/server/:id/config            │──► prepare ─► adapt ─► load ─► 落库
                       └──────────────────────────────────────────┘
```

两条链路各自 `adapt → load → 落库`，**没有共享锁、没有版本号、没有 ETag**。

当前 WAF 生效方式（`managed_caddyfile.go` / `waf_integration_helpers.go`）：

```caddyfile
{
  order coraza_waf first          # 必须有，否则 coraza_waf 被排到响应类指令之后
}

(waf_protect) {
  coraza_waf {
    load_owasp_crs
    directives `
      Include /config/security/active/rules/@coraza.conf-recommended
      Include /config/security/active/rules/@crs-setup.conf.example
      SecRuleEngine DetectionOnly
      ...
      Include /config/security/active/rules/@owasp_crs/*.conf
    `
  }
}

example.com {
  import waf_protect
  ...
}
```

这个骨架是对的，问题出在**指令内容**和**维护骨架的手段**上。

---

## 2. 现状：WAF 配置数据流

```
waf_policies（单条 is_default 记录，scope=global）
   │ buildWafPolicyDirectives()          ← 生成 SecRule/SecAction 文本
   ▼
applyWafPolicyToCaddyConfig()            ← 用 strings.Index("directives `") 找到第一个块，整段替换
   ▼
caddy_servers.config（Caddyfile 全文）+ modules（结构化 JSON）
   ▼
POST /adapt → POST /load → 写 caddy_config_history
```

关键事实：

- `waf_policies` 实际只有一条默认策略，`scope` 恒为 `global`；
- "站点选择"**不落库**，`Get` 用 Caddyfile 里的 `import waf_protect` 反推（`simple_waf_config_service.go:339`）；
- `Save`（`/simple-config`）只写库，不碰 Caddy，但响应里的 `enabled` 被 `&& integrated` 改写（`simple_waf_config_service.go:343`），前端看到的是"关闭"；
- `Get` 会顺带创建默认策略并修正 `is_default` —— 读接口写库（`simple_waf_config_service.go:222`）。

---

## 3. 业界做法对照

### 3.1 Caddy Admin API（官方）

- 整份替换走 `POST /load`：**失败自动回滚、零停机**，这是唯一"无需自己实现回滚"的入口。
  参见 [Caddy API 文档](https://caddyserver.com/docs/api)。
- 已加载配置相同时不会触发 reload；强制重载需 `Cache-Control: must-revalidate`。
- 校验走 `POST /adapt`（`Content-Type: text/caddyfile`），只转 JSON 不加载，400 返回 `error` / `warnings`。
- 局部更新可用 `@id`：把 `routes/N/handle/M` 收成 `/id/<id>/...` 后 `PATCH`。
- 并发控制：`GET` 返回 `Etag`，写时带 `If-Match`，不匹配回 412（[admin.go](https://github.com/caddyserver/caddy/blob/master/admin.go)）。
- `admin.config.persist` 默认 true，只落"推上来"的配置；进程用 `caddy run --resume` 恢复上次成功配置。

**对我们的意义**：`/load` 已经是原子回滚点了，我们不需要再造一套"反向文本撤销"；需要补的是**乐观并发**和**变更幂等**。

### 3.2 coraza-caddy（官方插件）

- 全局块必须 `order coraza_waf first`，插件无默认 order（[README](https://github.com/corazawaf/coraza-caddy)）。这一点我们已做对。
- CRS 两种加载方式：
  - `load_owasp_crs` + `Include @coraza.conf-recommended` 等 `@` 路径 —— **`@` 前缀必须是路径开头**，不能拼磁盘前缀；
  - 或磁盘路径 `Include /config/security/active/rules/...`（我们现在的方式，可行）。
- 建议用 `handle_errors 403` 定制拦截页，这是产品化最低成本的"用户体验"补丁。
- 可设 `tx_id_req_header` 复用下游请求 ID。

### 3.3 OWASP CRS 4.x（官方，重点）

相关原文：[Paranoia Levels](https://coreruleset.org/docs/2-how-crs-works/2-2-paranoia_levels/)、[crs-setup.conf.example](https://github.com/coreruleset/coreruleset/blob/main/crs-setup.conf.example)。

- **PL 与异常阈值是两个正交维度**。官方建议：阈值固定 `inbound=5 / outbound=4`，用 PL 调强弱。
  > "keep things simple and separate: a CRS installation should aim for an anomaly threshold of 5 and a paranoia level as deemed appropriate."
- 阈值超过 5 会**成片废掉** LFI / RFI / RCE / 数据外泄防护（单条 CRITICAL = 5 分，阈值 10 意味着要命中两条才拦）。
- 变量名在 CRS 4 已改：`tx.paranoia_level` → **`tx.blocking_paranoia_level`**，并新增 `tx.detection_paranoia_level`（观察态，只执行不计分）。
  `REQUEST-901-INITIALIZATION.conf` 中：901120 兜底 blocking=1，901125 把 detection 设为等于 blocking，901500 强制 detection >= blocking，否则直接 500。
- 升 PL 的正确姿势：先抬 `detection_paranoia_level`，让高 PL 规则只跑不计分，误报排除干净后再抬 `blocking_paranoia_level`。
- 排除规则的优先级：`ctl:ruleRemoveTargetById`（精确到 URI + 参数）> `ctl:ruleRemoveById`；运行时排除必须写在 `Include CRS` **之前**，`SecRuleRemoveById` 写在 `Include` **之后** 仅作兜底。
- 新站灰度：`tx.sampling_percentage` 或先 DetectionOnly。

### 3.4 SafeLine（雷池）

产品形态可直接借鉴，实现细节不宜照搬（闭源）：

- 以**站点/应用**为单位绑定防护，而不是全局开关；
- 防护强度分"均衡 / 严格"，并**公开各自的误报率**（均衡 0.07%、严格 0.22%）——把"强度"的代价显式化；
- 事件 → 规则命中 → 处置动作 的闭环，误报处理是主流程而不是附属页。
  参见 [chaitin/SafeLine](https://github.com/chaitin/SafeLine)。

### 3.5 BunkerWeb

- 配置分 **Easy / Advanced / Raw** 三个入口，三者写同一份期望态，Raw 中的注释作为草稿不生效；
- 下发前有 `DISABLE_CONFIGURATION_TESTING` 开关，默认**先测再下发**；
- UI 不是第二份真相，调度器负责推送到实例。
  参见 [BunkerWeb Web UI](https://docs.bunkerweb.io/latest/web-ui/)。

**对我们的意义**：我们正好反过来了 —— 三个入口（分块 / 原文 / 防火墙）各自用文本手术改同一份 Caddyfile，谁改谁算。

---

## 4. 缺陷清单

严重度：P0 = 安全或功能失效；P1 = 数据/状态不一致；P2 = 可维护性与体验。

| 编号 | 级别 | 位置 | 现象 | 后果 |
|---|---|---|---|---|
| C-01 | P0 | `waf_policy_builder.go:279` | `setvar:tx.paranoia_level` | PL 恒为 1，PL2/PL3 的 119 条规则永不启用（阈值部分仍生效） |
| C-02 | P0 | `waf_integration_helpers.go:460` | Host 为 IP 即 `ctl:ruleEngine=Off` | 默认 `:80` catch-all 下，任意 Host 头即可绕过 WAF |
| C-03 | P0 | `waf_integration_helpers.go:424` | `/api/` 全关引擎 | 管理面无防护；raw 模式删不掉 |
| C-04 | P1 | `waf_policy_builder.go:281` | `id:900100` 被用来设出站阈值 | 与官方 900100（异常分值定义）撞号；官方该规则当前默认注释，属埋雷未炸 |
| C-05 | P2 | `waf_policy_builder.go:31-47` | 预设把 PL 与阈值绑死（low_fp=10/8） | 阈值 10 时单条 CRITICAL 规则（如 942100 SQLi）不达阈值，等于该档下此类攻击不拦；且 PL 失效使该绑定失去意义 |
| C-06 | P1 | `simple_waf_config_service.go:339-343` | `enabled = policy.Enabled && integrated` | 一次 Caddyfile 漂移就让 UI 显示"关闭"，与库内期望态不符 |
| C-07 | P1 | `simple_waf_config_service.go:402-414` | `SiteAddresses` 只在请求里用，从不落库 | 用户选站点后保存 → 刷新回显全部站点 |
| C-08 | P1 | `simple_waf_config_service.go:222` | 读接口内创建/修正默认策略 | GET 有副作用，并发首访可能撞唯一索引 |
| C-09 | P1 | `simple_waf_config_service.go:84` | `directives, _ :=` 忽略错误 | 策略非法时静默返回空指令 |
| C-10 | P1 | `waf_policy_builder.go:297` | `strings.Index("directives \`")` 只取第一个 | 多站点各自内联 Coraza 时只有第一处被更新 |
| C-11 | P1 | `caddy_config_apply_service.go:44-79` | 首次接管与常规路径的 load/persist 顺序相反 | 失败回滚语义不一致，难推理 |
| C-12 | P1 | `rollback_caddy_config_logic.go:41-64` | 先 load 再落库，落库失败不回退运行态 | 运行态与 DB 不一致，直到下次重启 |
| C-13 | P1 | `CaddyConfigHistory.Action` `size:20` | `waf_integration_disable`(23) / `_enable`(22) 超长 | 该端点一旦被调用即 500（varchar(20) 直接报错） |
| C-14 | P1 | 全链路 | 无 ETag / 版本号 / 锁 | 两人同时保存，后写覆盖前写，无告警 |
| C-15 | P1 | `caddy_config_service.go:166-177`、`232-251` | 每次保存都注入排除规则 | 用户显式删除无效，且 raw 内容被静默改写 |
| C-16 | P2 | `waf_integration_helpers.go:255` | 手写行级 Caddyfile 扫描器 | 反引号内多行、heredoc、quoted `{`、CRLF 混排下块边界会错位 |
| C-17 | P2 | `caddy_config_service.go:756` | `formatCaddyfile` 只按引号计数大括号 | 反引号块内的 `{`/`}` 会被重排缩进，可能破坏指令 |
| C-18 | P2 | `apply_waf_integration_logic.go` 全文件 | 前端 0 调用 | 死接口；并带出 C-13 |
| C-19 | P2 | `useCaddyConfigIO.ts:154,473,527` | `pendingWizardWaf` 只写不读 | 向导里的"可选 WAF"是死代码 |
| C-20 | P2 | `managed_caddyfile.go:53-85`、`caddy_config_source.go:10-40` | `buildPolicyCandidateCaddyConfig` / `shouldRenderManagedCaddyfile` / `readLocalCaddyfile` 无调用方 | 死代码，误导后续维护 |

---

## 5. 优化方案

### 5.1 策略模型：PL 与阈值解耦

`waf_policies` 增加一个字段，其余保持不动（不新建绑定表，见 5.4）：

```go
// 观察档位：只执行不计分，用于升 PL 前的灰度。
DetectionParanoiaLevel int64 `gorm:"not null;default:1" json:"detectionParanoiaLevel"`
```

强度只决定 PL：

| 强度 | blocking PL | detection PL | inbound / outbound |
|---|---|---|---|
| 低误报 `low_fp` | 1 | 1 | 5 / 4 |
| 平衡 `balanced` | 2 | 3 | 5 / 4 |
| 严格 `high_blocking` | 3 | 4 | 5 / 4 |

- 阈值固定官方推荐值，**不再暴露给简单模式**；`custom` 模板只允许高级接口设置阈值，且上限 5 / 4（超过直接拒绝并说明会废掉哪些防护类）。
- `derivePolicyCRSTemplateFromValues` 改为"按 blocking PL 反推"，阈值不参与，避免数值漂移导致模板退化成 `custom`。
  （注：当前 `normalizeSimpleWafStrength` 对 custom 会静默降级为 `balanced`，不会死循环；但降级同样意味着用户设置被无声改写，仍需消除。）

### 5.2 指令生成：对齐 CRS 4 变量

替换 `buildWafPolicyDirectives` 末尾三个 `SecAction`（`waf_policy_builder.go:276-282`）：

```go
lines = append(lines,
    fmt.Sprintf(`SecAction "id:900000,phase:1,pass,nolog,t:none,setvar:tx.blocking_paranoia_level=%d"`, blockingPL),
    fmt.Sprintf(`SecAction "id:900001,phase:1,pass,nolog,t:none,setvar:tx.detection_paranoia_level=%d"`, detectionPL),
    fmt.Sprintf(`SecAction "id:900110,phase:1,pass,nolog,t:none,setvar:tx.inbound_anomaly_score_threshold=%d,setvar:tx.outbound_anomaly_score_threshold=%d"`, inbound, outbound),
)
```

三条硬约束，写进注释并加校验：

- `detectionPL >= blockingPL`，否则 CRS 901500 直接 500；
- 这三条必须在 `Include @crs-setup.conf.example` **之后**、`Include @owasp_crs/*.conf` **之前**（现有 `composeManagedCorazaDirectives` 顺序已满足，加单测锁住）；
- `SecRuleEngine` 与 PL 无关，`DetectionOnly` 下 PL 仍需设置（决定执行到哪一层）。

> 上线注意：修复 C-01 后，PL 才真正开始起作用 —— "平衡 / 严格"会**按 PL2/PL3 启用那 119 条此前从未执行的规则**，误报会显著上升。存量环境切换时必须先强制 `DetectionOnly` + `detectionPL = blockingPL + 1` 观察一个周期，再允许切 `On`。这条要写进升级说明。
>
> 注意区分：阈值调整（5/4）在当前代码下**已经生效**，只是 PL 维度没生效；修复后是"两个维度同时起作用"，不是"从无效变有效"。

### 5.3 排除规则：从"关引擎"改为"定点排除"

删除 `ensureCrsFalsePositiveExclusions`（C-02）。`/api/` 的排除改写为只排除**已知误报字段**：

```go
// 仅对管理面配置保存接口关闭请求体检查：请求体里是 Caddyfile 原文，
// 会命中 SSRF / 协议类规则。URI 与 Host 仍参与检测。
const apiConfigExclusionRuleID = "1000100"
const apiConfigExclusionRule = `SecRule REQUEST_URI "@rx ^/api/caddy/server/\d+/config(/preview)?$" ` +
    `"id:` + apiConfigExclusionRuleID + `,phase:1,pass,nolog,noauditlog,ctl:requestBodyAccess=Off,ctl:ruleRemoveTargetById=920420"`
```

要点：

- 用 `@rx` 精确到端点，不再 `@beginsWith /api/`；
- 优先 `ctl:ruleRemoveTargetById` / `requestBodyAccess=Off`，不用 `ruleEngine=Off`；
- 排除了体检查后，`/api/` 上的 SQLi/XSS 仍需检测，登录接口不再裸奔。

同时把 C-15 的"每次保存都注入"改成：**排除规则只存在于策略渲染结果里**，由 5.2 的 directives 生成统一输出，`prepareQuick` / `prepareRaw` 不再改用户文本（只做 `/adapt` 校验）。这样 raw 模式所见即所存。

### 5.4 站点绑定：先删中间态，不建新表

`SiteAddresses` 的真正问题是"保存了但没生效"。而 Caddyfile 里哪些站点 `import waf_protect`，本身就是最准确的绑定事实。

ponytail：不新增 `waf_site_bindings` 表。前端把"保存设置 / 预览变更 / 应用到 Caddy"三个按钮收敛成 **预览 → 应用** 两个：

- `PUT /simple-config` 保留但语义改为"只保存草稿意图"，写入 `policy.Config["siteAddresses"]`；
- `GET` 优先读该键，缺失时回退到 `importedSites`；
- UI 上把"已生效 / 未生效（草稿）"分开显示，去掉 `enabled && integrated` 的静默与运算（C-06）。

若产品确认不需要草稿，直接删 `PUT`，只留 preview + apply —— 更懒也更不容易错。

### 5.5 Caddy 写入：乐观并发 + 漂移检测

Caddy `/load` 本身原子回滚，缺的只是"别覆盖别人的改动"。

1. `GET /caddy/server/:id/config` 返回新增字段 `hash`（已有 `hashConfig`，直接复用）。
2. 写接口（`POST config`、`POST simple-config/apply`、`POST rollback`）请求体带 `baseHash`；服务端比对 DB 当前 `hashConfig(server.Config)`，不一致返回业务错误"配置已被他人修改，请刷新后重试"。
3. 应用成功后重新计算并返回新 `hash`，前端把它作为下一次编辑的 `baseHash`。
4. 漂移检测：定时（或打开页面时）`GET /config/` 取运行态 JSON 哈希，与 DB 哈希不一致则前端横幅提示"运行态与配置库不一致"，并提供"以运行态为准 / 以库为准"。

这比接 `If-Match` 便宜得多，且不依赖 Caddy 内部 `@id` 约定。

### 5.6 文本手术：收敛到"标记块 + 幂等渲染"

不重写解析器（成本 > 收益），只做三件事：

1. **站点挂载加标记**。插入时同时写 `import waf_protect` 和一行注释，删除时按标记整行删，不靠缩进猜测：

```caddyfile
example.com {
  # logflux:waf-mount
  import waf_protect
  reverse_proxy localhost:8080
}
```

2. **`waf_protect` snippet 整体重渲染**。不按内容判断"是否已存在"，而是有则整块替换（`replaceLineRange` 已有），无则插入；内容永远由 `renderManagedWafSnippet` 单一函数产出，保证幂等。

3. **`applyWafPolicyToCaddyConfig` 改为遍历所有 `directives \`` 块**（修 C-10），或明确约定"每个 Caddyfile 只允许一个 `waf_protect` snippet，站点必须 `import`"，后者更简单 —— 建议后者，并在 preview 阶段校验"存在内联 coraza_waf 块"时给出告警。

若后续确实需要更强的编辑能力，再考虑引入 `github.com/caddyserver/caddy/v2/caddyconfig/caddyfile` 做**只读**解析（写仍走文本 + `/adapt` 兜底）。当前不必引依赖（C-16/C-17 保持为已知风险，用 `/adapt` 拦截）。

### 5.7 入口收敛与死代码清理

- 删 `/caddy/waf/integration/*`（后端 3 文件 + `frontend/apps/src/api/caddy/integration.ts`）；
- 删 `pendingWizardWaf` 相关分支，或把它接到 5.4 的 `siteAddresses`（二选一，别留半截）；
- 删 `buildPolicyCandidateCaddyConfig` / `shouldRenderManagedCaddyfile` / `defaultManagedCaddyfileOptions` / `readLocalCaddyfile` / `localCaddyfilePath`；
- `CaddyConfigHistory.Action` 扩到 `size:32`，并定义常量枚举（`update` / `rollback` / `simple_waf_apply` / `simple_waf_last_good`），前端"回滚/更新"标签按枚举渲染，别再 `=== 'rollback'` 二元判断；
- `/caddy/source` 与防火墙页的职责划清：**源管理只管规则包版本，防火墙页只管策略**，互不跳转塞参数。

### 5.8 误报闭环（P2，产品化）

现在日志页只能"看到"WAF 命中（`frontend/apps/src/views/logs/shared/waf-log.ts`），不能处置。最小闭环：

1. 命中详情页加"这是一条误报"按钮；
2. 后端按当前事务生成一条排除记录：Host + URI + 规则 ID + 目标参数 → 渲染成 `ctl:ruleRemoveTargetById`；
3. 排除记录入库（新表 `waf_rule_exclusions`），渲染时统一拼到 `Include @owasp_crs` 之前；
4. 撤销 = 删记录 + 重新渲染 + apply。

不做"一键关整条规则"——那会让防护退化成黑名单。

---

## 6. 分阶段任务表

### 阶段一：正确性（P0，1~2 天）

| 任务 | 文件 | 验收 |
|---|---|---|
| 改 CRS4 变量名 + 合并阈值到 900110 | `waf_policy_builder.go` | 单测断言三条 `SecAction` 内容与顺序 || 删除 Host 为 IP 的引擎关闭规则 | `waf_integration_helpers.go` | 构造 `Host: 1.2.3.4` + `?id=1' or 1=1--` 被拦 |
| `/api/` 改端点级 `requestBodyAccess=Off` | 同上 | 登录接口能拦 SQLi；配置保存不误拦 |
| 历史 action 扩长 + 枚举 | `model/caddy/caddy_config_history.go` | 22/23 字符可写入 |

### 阶段二：状态一致（P1，3~5 天）

- 新增 `DetectionParanoiaLevel` + 迁移；预设表改为固定 5/4；
- `GET` 去掉写库，改为"无默认策略时返回空态"，由 apply 时创建；
- `SiteAddresses` 落 `policy.Config`，UI 三按钮收敛为两按钮；
- `hash` + `baseHash` 乐观并发；
- `rollback` 补"落库失败回退运行态"；
- `apply` 幂等：`config` 未变直接返回成功，不触发 `/load`。

### 阶段三：可维护（P2，按需）

- 标记块化站点挂载；`waf_protect` 整块重渲染；
- 删除死代码与死接口；
- 误报闭环；
- 可选：`handle_errors 403` 定制拦截页 + `tx_id_req_header` 复用请求 ID。

---

## 7. 验收与测试

**后端单测（`go test ./internal/logic/caddy/...`，使用默认 GOCACHE）**

- `TestBuildWafPolicyDirectivesCRS4Vars`：断言含 `tx.blocking_paranoia_level=` / `tx.detection_paranoia_level=`，且**不含** `tx.paranoia_level=`（注意子串包含关系，用 `\btx\.paranoia_level` 或整行比对）；
- `TestPresetsUseOfficialThresholds`：三档 inbound/outbound 恒为 5/4；
- `TestDirectiveOrderBeforeRulesInclude`：三条 SecAction 的下标介于 `@crs-setup` 与 `@owasp_crs` 之间；
- `TestEnsureSiteMountIdempotent`：连续 ensure 两次，第二次 `changed == false`；
- `TestRemoveSiteMountByMarker`：手改缩进后仍能精确删除；
- `TestApplyNoChangeNoLoad`：配置无变更时 Caddy mock 未收到 `/load`。

**联调冒烟（容器内）**

```bash
# 1) 配置可适配
curl -s -X POST http://localhost:2019/adapt \
  -H 'Content-Type: text/caddyfile' --data-binary @/etc/caddy/Caddyfile | head

# 2) 确认运行态真的生效（PL 应为设定值，而非 1）
docker exec logflux-app grep -o 'tx.blocking_paranoia_level=[0-9]' \
  /config/security/active/rules/@crs-setup.conf.example || true

# 3) 绕过用例（修复后应返回 403）
curl -s -o /dev/null -w '%{http_code}\n' \
  -H 'Host: 1.2.3.4' 'http://127.0.0.1/?id=1%27%20or%201=1--'
```

**前端**

- `pnpm typecheck`；
- 手动回归：切换强度后刷新页面，PL 与阈值回显一致；保存草稿后刷新，站点选择不丢；他人并发保存时提示"配置已被修改"。

---

## 8. 明确不做

- 不新建 `waf_site_bindings` 表：Caddyfile 的 `import waf_protect` 已是最准确的绑定事实。
- 不引入 gRPC / 第二套配置存储：`caddy_servers.config` 保持唯一期望态。
- 不重写 Caddyfile 解析器：用 `/adapt` 做权威校验，文本编辑保持最小且幂等。
- 不做"规则级开关 UI"：CRS 规则上千条，产品上只暴露"强度 + 站点 + 误报排除"，规则清单仅只读展示。
- 不做 Coraza 在线热替换：引擎随镜像构建，只做版本检查与升级提示（现有设计正确）。
