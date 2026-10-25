# Caddy 配置框架重构方案（文本权威 + 单一写入器 + 单一变换层）

> 更新：2026-10-21
> 范围：`backend/internal/logic/caddy/`、`backend/internal/waf/`、`frontend/apps/src/views/caddy/config/`
> 依据：对 64 个后端文件（10823 行）、29 个 handler、约 50 个前端文件的调用图勘察 + 反引号格式化缺陷实测

---

## 0. 核心判断

系统当前有**五个真相源**表示同一份 Caddyfile：

| # | 位置 | 权威性 |
|---|---|---|
| 1 | `caddy_servers.config`（TEXT） | 权威 |
| 2 | `caddy_servers.modules`（JSONB） | 会过期 |
| 3 | 前端 `formModel` | 运行时 |
| 4 | 前端 `preservedBlocks` | 后端不认识 |
| 5 | 前端 `configContent` | 文本副本 |

**决策：`config` 是唯一真相，`modules` 降级为纯前端表单快照，不参与任何后端决策。**

为什么不是反过来（结构化权威）：
- `preservedBlocks`（无法结构化的复杂块）本质是文本，强行结构化就是现在的双模型；
- 前端需要实时预览（`generatedConfig` 是 computed），不能每次调后端 render；
- 后端已经为此写了一套完整的 parser + renderer（656 行），但被前端的实现替代了，没人删。

---

## 1. 已核实的缺陷清单

| 编号 | 级别 | 位置 | 现象 |
|---|---|---|---|
| F-01 | P0 | `caddy_config_service.go:1214` | raw 模式保存保留旧 modules，导致 blocks 模式用过期数据覆盖 raw 修改 |
| F-02 | P0 | `caddy_config_service.go:835-1130` | `inferCaddyModulesFromConfig` 及 12 个下游函数，refs=1，纯死代码（296 行） |
| F-03 | P0 | `apply_waf_integration_logic.go` 等 | `/caddy/waf/integration/{apply,status}` 前端零调用，后端完整实现 + 3 个单测（约 390 行） |
| F-04 | P1 | `simple_waf_config_service.go:292` vs `caddy_config_apply_service.go:64` | 两条写路径，adapt/load 位置、history 条数、回滚目标、日志源同步全不一致 |
| F-05 | P1 | `caddy_config_service.go:187`、`simple_waf_config_service.go:528` | 三套 WAF 文本手术，后两者几乎逐行重复 |
| F-06 | P1 | `rollback_caddy_config_logic.go`、`apply_waf_integration_logic.go` | 4 条写路径只有 2 条做 baseHash 校验 |
| F-07 | P1 | `caddy_config_service.go:746` | `formatCaddyfile` 不感知反引号，块内独立花括号会导致缩进漂移 |
| F-08 | P2 | `caddy_config_history` | 只增不删，一次 WAF apply 写 2 条，无 retention |
| F-09 | P2 | `caddy_config_service.go:386-745` | `renderCaddyFormModel` 及 11 个下游（360 行），唯一调用点是不可达的防御分支 |
| F-10 | P2 | `simple_waf_config_service.go:282` | `policy.Config` 的 `scope` / `simpleMode` 只写不读 |
| F-11 | P2 | `useCaddyConfigIO.ts:514-527` | `SiteWizardWafDraft` 只用来拼一句 `message.success` |
| F-12 | P2 | `waf_policy_error_helpers.go` | 165 行映射表，90% 是同义改写 |

### F-01 复现路径

```
blocks 编辑 → 保存        config=X, modules=M1
raw 手改   → 保存        config=Y, modules=M1   ← resolveCaddyConfigModules 保留旧值
刷新页面                 loadModules(M1) 成功 → formModel=M1 → 提前 return
切 blocks 模式           generatedConfig 由 M1 生成
再次保存                 Y 的修改被覆盖
```

根因：`resolveCaddyConfigModules` 的 raw 分支返回 `existingModules`。raw 模式手改文本后旧 modules **一定**过期，保留它没有任何收益。

### F-07 实测结论

用临时探针跑了 4 组用例 + 线上真实 Caddyfile：

- 反引号块内出现**独立成词**的花括号（`SecMarker {`）→ directives 内容缩进被打乱，但反引号仍成对、`parseTopLevelCaddyBlocks` 仍能解析；
- 对 `testdata_live_caddyfile.txt` 只产生 3 行续行缩进变化，**语义不变**；
- Coraza directives 每行独立解析，缩进不敏感；
- 所有用例格式化幂等。

**结论：F-07 是 P1 的整洁性问题，不是"WAF 会失效"的安全缺陷。** 上一轮的判断过重了。

---

## 2. 分阶段方案

### P0 — 零风险清理（可独立合并，无行为变化）

| 任务 | 文件 | 删除量 |
|---|---|---|
| 删 `inferCaddyModulesFromConfig` 及 12 个下游 | `caddy_config_service.go:835-1130` | 296 行 |
| 删 `/caddy/waf/integration/{apply,status}` | 2 handler + 2 logic + 2 routes + 1 test + 4 types + 2 常量 | 约 390 行 |
| 删 `policy.Config` 的 `scope` / `simpleMode` 写入 | `simple_waf_config_service.go:282,505-506` | 3 行 |
| 删 `SiteWizardWafDraft` 链路 | `site-wizard-utils.ts`、`SiteWizardModal.vue`、`useCaddyConfigIO.ts` | 约 60 行 |

**验收**：`go build ./... && go test ./internal/logic/caddy/...` 全绿；`pnpm typecheck` 通过；手工回归配置保存/回滚/WAF 应用三条主链路。

**注意**：`renderCaddyFormModel`（360 行）**不在 P0**。它的唯一活调用点是 `caddy_config_service.go:161`：

```go
caddyfile := config
if strings.TrimSpace(caddyfile) == "" {
    caddyfile = renderCaddyFormModel(model)   // 前端总是传非空，不可达
}
```

前端确实从不传空 config，但这个分支对直接调 API 的调用方仍有效。删它 = 改 API 行为，需要单独决策（见 P5）。

---

### P1 — 修 modules 过期（消除数据丢失路径）

**改动**：`resolveCaddyConfigModules` 的 raw 分支不再保留旧 modules。

```go
func resolveCaddyConfigModules(mode, requestModules, existingModules string) string {
	normalizedMode := normalizeCaddyConfigMode(mode, requestModules)
	if normalizedMode == caddyConfigModeQuick {
		return normalizeCaddyModulesJSON(requestModules)
	}
	if strings.TrimSpace(requestModules) != "" {
		return normalizeCaddyModulesJSON(requestModules)
	}
	// raw 模式手改文本后旧 modules 必然过期，继续保留会让 blocks 模式
	// 用旧快照覆盖用户刚保存的修改。宁可留空，让前端回退到文本解析。
	return emptyModulesJSON
}
```

**配套**：前端 `fetchConfig` 的 `loadModules` 失败分支已有 `parseCaddyfileToBlocks` 兜底（`useCaddyConfigIO.ts:361-371`），modules 为空后会自动走这条路，无需改前端。

**验收**：构造 raw 保存 → 刷新 → 切 blocks，`formModel` 必须来自 `parseCaddyfileToBlocks` 而非旧 modules；补一个单测锁住 `resolveCaddyConfigModules(raw, "", oldModules) == emptyModulesJSON`。

---

### P2 — 单一写入器

把 `caddyConfigApplyService.apply` 提升为唯一写路径，用选项表达差异：

```go
type configWriteOptions struct {
	HistoryAction  string // 当前这条 history 的动作名
	SnapshotBefore bool   // true 时额外写一条 last_good（WAF 需要）
	BaseHash       string // 非空时做乐观并发校验
}

func (s *configWriter) Apply(server *caddymodel.CaddyServer, config, modules string, opts configWriteOptions) error
```

三处调用方改造：

| 调用方 | 当前 | 改造后 |
|---|---|---|
| `update_caddy_config_logic.go:49` | `applyService.apply(...)` | `writer.Apply(..., {HistoryAction: update, BaseHash: req.BaseHash})` |
| `simple_waf_config_service.go:142-154` | 自己 adapt→load→persist→rollback | `writer.Apply(..., {HistoryAction: simple_waf_apply, SnapshotBefore: true, BaseHash: ...})` |
| `rollback_caddy_config_logic.go:45-74` | 自己 adapt→load→persist→rollback | `writer.Apply(..., {HistoryAction: rollback, BaseHash: req.BaseHash})` |

**收益**：
- 三条路径共用同一套 adapt→load→persist→rollback 语义，消除 F-04；
- `rollback` 顺带补上 baseHash 校验，消除 F-06；
- `persistAppliedCandidate` 的双 history 需求用 `SnapshotBefore: true` 表达，不复制整个事务。

**验收**：`go test ./internal/logic/caddy/...` 全绿；三条路径各自补一个"落库失败时运行态被回滚"的测试。

---

### P3 — 单一文本变换层

新建 `backend/internal/caddyfile/`（与已有的 `internal/waf/` 平级，那个包是干净的），只放**纯函数**（无 DB、无 HTTP）：

```
internal/caddyfile/
  parse.go      ParseTopLevelBlocks / SplitLinesKeepEndings / DetectNewline
  format.go     Format（修 F-07）
  snippet.go    EnsureOrder / EnsureSnippet / EnsureSnippetBeforeSites
  site.go       EnsureSiteImport / RemoveSiteImport / FindSiteBlock
  policy.go     ApplyPolicyDirectives / FindDirectiveBlocks
```

三套文本手术收敛：

| 当前 | 改造后 |
|---|---|
| `ensureQuickWafImportDependencies`（`caddy_config_service.go:187`） | 组合 `caddyfile.EnsureOrder` + `EnsureSnippet` + `EnsureSnippetBeforeSites` |
| `buildSimpleWafCandidateConfig`（`simple_waf_config_service.go:528`） | 同上 + `EnsureSiteImport` + `ApplyPolicyDirectives` |
| `ApplyWafIntegration`（`apply_waf_integration_logic.go`） | P0 已删 |

**验收**：`go build ./internal/caddyfile/` 无 `logflux/model/*`、`logflux/internal/svc` 依赖；表驱动测试覆盖幂等性（连续调用两次，第二次 `changed == false`）。

---

### P4 — 修 `formatCaddyfile` 反引号

`countCaddyLineBraces` 加 `inBacktick` 状态，与 `updateCaddyParseState`（`waf_integration_helpers.go:313`）已有的实现对齐：

```go
inBacktick := false
for idx := 0; idx < len(sanitized); idx++ {
	ch := sanitized[idx]
	if inBacktick {
		if ch == '`' {
			inBacktick = false
		}
		continue
	}
	if !inQuote && ch == '`' {
		inBacktick = true
		continue
	}
	// ... 原有引号/花括号逻辑
}
```

**验收**：反引号块内含独立花括号（`SecMarker {`）时，格式化后 directives 内容逐行不变；保留现有幂等性断言。

---

### P5 — 删后端 render 双模型（需决策）

`renderCaddyFormModel` + 11 个下游（`renderCaddySite` / `renderCaddyTLS` / `renderCaddyRoute` / `renderCaddyHandle` / `renderCaddyReverseProxy` / `buildCaddyMatcherLines` / `selectCaddyMatcherName` / `resolveEffectiveHealthCheck` / `reverseProxyNeedsBlock` / `appendReverseProxyBlockLines` / `renderCaddyHeaderOnly`），共 360 行。

两个选项：

| 选项 | 做法 | 影响 |
|---|---|---|
| **1（建议）** | 删，`caddy_config_service.go:161` 的 fallback 改为 `return nil, fmt.Errorf("配置为空")` | 直接调 API 且只传 modules 的调用方会收到错误。前端不受影响 |
| 2 | 保留，加注释说明是 API 兼容分支 | 360 行死代码继续存在，但零风险 |

选 1 的前提：确认没有外部脚本/自动化只传 modules 不传 config。建议先查访问日志或加一轮埋点。

---

### P6 — 配置历史 retention

```go
// PruneCaddyConfigHistory 保留最近 keep 条，其余删除。
// 每次 apply 写 1~2 条完整 Caddyfile 文本，不清理会让表无限膨胀。
func PruneCaddyConfigHistory(db *gorm.DB, serverID uint, keep int) error
```

在 `configWriter.Apply` 成功后异步执行（`safego`），默认 `keep = 100`。

**验收**：连续 apply 120 次后，该 server 的 history 恰好 100 条。

---

## 3. 预期收益

| 指标 | 现在 | 重构后 |
|---|---|---|
| `logic/caddy` 非测试行数 | 7247 | 约 6100（P0 删 690 + P5 删 360，P2/P3 净增约 200） |
| 写路径 | 4 条（2 条有 baseHash） | 1 条（全部有 baseHash） |
| 文本手术实现 | 3 套 | 1 套 |
| 真相源 | 5 个 | 2 个（`config` + 前端 `formModel`） |
| 已知数据丢失路径 | 1 条（F-01） | 0 |

---

## 4. 明确不做

- **不删前端 parser**（`caddy-config-parser.ts` 791 行）：前端需要实时预览，不能每次调后端；
- **不引入 `github.com/caddyserver/caddy/v2/caddyconfig/caddyfile`**：写仍走文本 + `/adapt` 兜底，引入依赖收益不抵成本；
- **不做 `modules` schema 版本迁移**：降级为前端快照后不再需要版本管理；
- **不重写 `parseTopLevelCaddyBlocks`**：只迁移，不改逻辑（F-07 单独修）。

---

## 5. 与 WAF 简化方案的关系

此前针对 WAF 面板列的 P0–P5（删草稿态、`enabled`/`mode` 合一、请求体三控件合一、删死代码），其中：

- **"删死代码"已被本方案的 P0 覆盖**（integration 端点 + `SiteWizardWafDraft` + `policy.Config` 冗余键）；
- **"删草稿态"应等 P2 完成后再做**：草稿态问题的本质是 `config` 与 `waf_policies` 两个真相源不同步，单一写入器落地后，草稿态可以简化成"apply 失败时的降级路径"，而不是独立流程；
- **"`enabled`/`mode` 合一"**：前端 40 处引用，Jev 判收益/成本比 0.37，暂不做；
- **"请求体三控件合一"**：`SecRequestBodyNoFilesLimit` 在默认 `Reject` 下是硬上限，固定 1MB 有误杀风险，撤回；
- **"误报排除闭环"**：与框架重构正交，可并行推进，优先级高于任何 UI 简化。

---

## 6. 执行顺序与依赖

```
P0（零风险清理）
 ├─→ P1（modules 过期）
 └─→ P4（formatCaddyfile）  ─┐
                              ├─→ P3（单一变换层，依赖 P4 的 Format 和 P0 的删除）
P2（单一写入器）──────────────┘
                              └─→ P5（删 render 双模型，依赖 P2 确认无 modules-only 调用方）
                                        └─→ P6（retention）
```

P0 与 P2 可并行（不碰同一批文件）；P3 必须在 P0 之后（P0 已删掉第三套手术）。
