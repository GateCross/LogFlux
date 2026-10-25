# 站点优先重构：一步建站 + 证书中心

> 更新：2026-10-25
> 取代：`caddy-site-capability-model-plan.md`（其站点/能力模型被本方案吸收，原文档已加取代标注）
> 前置：`caddy-config-framework-refactor-plan.md` 的 P0–P2 作为本方案阶段 0
> 范围：`frontend/apps/src/views/caddy/`、`backend/{api,model,internal/logic/caddy,internal/tasks}`、`docker/caddy.Dockerfile`
> 决策依据：功能面对标雷池社区版的差距勘察（2026-10-25）+ Jev 两项架构决策（证书库存双源 0.77 / 保存语义保留确认制 0.87）

---

## 0. 北极星

**"上线一个新站 = 一个表单 + 一次确认，HTTPS 自动就绪，全程不离开当前页面。"**

对标雷池的差距结论（上轮勘察）：不在 WAF 引擎，在产品模型。雷池把"站点"当一等对象（添加网站 = 域名 + 源站地址，一步完成），LogFlux 把"Caddyfile 文本"当一等对象（6 步向导 + 跨页面 preview/apply + WAF 另页挂载 + 复杂站点进 preservedBlocks 只读）。

### 保存语义决策：保留确认制（不对标雷池的"保存即生效"）

现状是代码级硬约束"无确认不热加载"（`site-wizard-utils.ts:243-246`，`wizardAutoLoadEnabled()` 恒 false）。Jev 判断（置信 0.87）：这是带 WAF 的统一入口，误配置会让全部站点下线，确认制是值得保留的安全阀，不为对齐雷池而放弃。

**但确认的形态彻底改变**：从"6 步向导 + 保存草稿 + 跨页面 preview + 再 apply"压缩为"1 个表单 + 1 个弹窗内 diff 确认"。确认成本从十几次点击降到 1 次。

| 维度 | 现在 | 本方案 |
|---|---|---|
| 新建站点入口 | 配置管理页 → 向导按钮 | 服务目录 → 「添加站点」 |
| 表单 | 6 步向导（域名→上游→TLS→WAF→预览→应用） | 1 个表单（必填 2 项，其余折叠） |
| 生效动作 | 写草稿 → 切页面 preview → 再 apply | 点保存 → 弹窗看 diff → 确认（adapt→load→落库 原子） |
| 开 WAF | 建站后另去防火墙面板勾站点再 apply | 表单里一个开关 |
| HTTPS | 默认 auto，manual 要手填容器内路径 | 默认自动签发；已有证书从证书库下拉选 |
| 复杂站点 | 整块只读，只能 raw 手改 | 能力逐项识别，仅真正的分流逻辑进逃生舱 |

---

## 1. 本轮新核实的事实（方案依据）

| # | 事实 | 证据 | 对方案的意义 |
|---|---|---|---|
| 1 | 自研 Caddy 构建已带 `caddy-dns/cloudflare` 插件 | `docker/caddy.Dockerfile:14-19`（xcaddy build：coraza、coreruleset、geoip2、**cloudflare**、transform-encoder） | DNS-01 签发（Cloudflare）**零构建改动**即可支持，只差 Caddyfile 生成和 token 注入 |
| 2 | 后端与 Caddy 同容器，共享 `/data/caddy`、`/config` 卷 | `docker/docker-compose.yml` volumes；后端以 logflux 用户运行 | 证书库存可直接扫 `/data/caddy/certificates/`（PEM 解析有效期/签发者/SAN），上传证书写 `/config/certs/` 后 Caddy 可直接引用 |
| 3 | `caddy_servers` 支持 remote 类型，远端只有 Admin API HTTP 通道 | `model/caddy/caddy_server.go` | 卷扫描只覆盖 local 服务器，remote 是天然盲区（证书库存双源决策的动因） |
| 4 | 通知事件是静态清单 | `logic/notification/get_event_list_logic.go:34-83` | 新增"证书事件"组 + 2 个事件类型即可接通到期告警，通知渠道/规则/模板全复用 |
| 5 | Caddy 证书存储布局为 `/data/caddy/certificates/<issuer>/<domain>/{cert,key}` | Caddy v2 存储约定，卷已持久化 | 扫描器实现路径确定，且 Caddy 自动续期后文件原地更新，扫描即感知 |
| 6 | TLSConfig 只有 `auto/manual/off/internal`，manual 手填路径 | `frontend/.../types.ts:75-79` | "已有证书"场景用户被迫知道容器内路径——证书中心消灭这个心智负担 |

---

## 2. 目标体验

### 2.1 一步建站

```
服务目录（主入口，站点卡片网格）
  └─「添加站点」→ 单表单弹窗
       ├─ 域名        必填，多行（回车添加），支持 host:port / :80
       ├─ 上游        必填，反代地址（localhost:8080）或上游池下拉
       ├─ HTTPS       三选一，默认「自动签发」
       │     ├─ 自动签发（HTTP-01/TLS-ALPN，Caddy 原生，零配置）
       │     ├─ DNS 签发（Cloudflare；泛域名/无公网 80 时用）
       │     ├─ 使用已有证书（下拉选证书库存 / 或就地上传）
       │     └─ 暂不启用（内网/纯 HTTP）
       └─ 高级（折叠，默认收起）
             ├─ WAF 开关 + 强度（low_fp/balanced/high_blocking）
             ├─ 负载策略 / 健康检查
             └─ 端口 / 监听选项
  └─ 点「保存」→ diff 确认弹窗（新增站点块高亮，其余行灰色"无变化"）
       └─ 确认 → adapt → load → 落库（原子，失败展示 Caddy 原始错误，不落库）
```

**6 步向导（`SiteWizardModal.vue` + `site-wizard-utils.ts`）删除**，`SiteWizardWafDraft` 等过渡机制一并清理（framework 计划 P0 已列）。

### 2.2 站点抽屉（编辑既有站点）

服务目录点卡片 → 右侧抽屉（不跳页），三 Tab：

| Tab | 内容 |
|---|---|
| 基本 | 域名 · 上游 · TLS（与建站表单同一套控件，复用组件） |
| 防护 | WAF 开关+强度 · IP 限制 · 安全头 · 错误页 |
| 高级 | 压缩 · 访问日志 · 健康端点 · 站点级逃生舱（rawExtras，只改这一个块） |

每个 Tab 底部一个「保存」，同样走 diff 确认弹窗。能力逐项识别（三态：on / off / unknown），复杂站点不再整块投降进 preservedBlocks。

### 2.3 证书中心

新页面 `/caddy/certificates`（挂在服务目录右上角入口）：

- **库存表**：域名 · 签发者（Let's Encrypt / 内部 CA / 上传）· 有效期 · 剩余天数徽章 · 来源（ACME 自动 / DNS 自动 / 上传 / 手动指定）· 所属服务器
- **缺口视图**：Caddyfile 声明了 HTTPS 但卷里没有对应证书的站点（含"从未签发成功"的诊断价值）
- **上传**：PEM + 私钥 → 存 `/config/certs/<serverID>/<domain>/` → 自动出现在站点 TLS 下拉里
- **联动**：服务目录站点卡片的证书徽章（剩 14 天内黄色/过期红色）；站点抽屉 TLS Tab 直接选库内证书

---

## 3. 架构

### 3.1 站点模型

沿用被取代文档的 `Site` 结构（id/domains/enabled/tls/upstream/capabilities/rawExtras），Caddyfile 文本仍是唯一真相源，能力是投影，不落新表。仅一处扩展——`TlsConfig`：

```typescript
interface TlsConfig {
  mode: 'auto' | 'dns' | 'internal' | 'off' | 'manual' | 'certRef';
  dnsProvider?: 'cloudflare';        // 构建里已有插件；alidns/dnspod 属阶段 5
  certId?: string;                   // certRef: 证书库存引用，渲染时解析为文件路径
  certFile?: string;                 // manual: 保留，仅逃生舱场景
  keyFile?: string;
}
```

`certRef` 是证书中心与站点模型的接合点：用户选的是"库里的那张证书"，路径解析是渲染细节，不再暴露给用户。

### 3.2 能力契约（三态）与 import 优先

沿用原文档设计：每个 capability 实现 `parse / render / isActive` 三函数；`parse` 返回 null 即 `unknown` 态（UI 禁用开关，提示"检测到无法识别的配置，请用高级模式"）；render 优先产出 `import waf_protect` / `import geo_auth` 等 snippet 挂载行，snippet 定义在全局设置页管理。站点块保持 4~6 行可读。

### 3.3 写路径与保存链路

```
表单/抽屉「保存」
  → 前端定点替换站点块（applyCapabilityToSiteSiteBlock，幂等有单测）
  → diff 确认弹窗（纯前端 + /adapt 校验）
  → 确认 → 后端单一写入器 writer.Apply（framework P2）：
        adapt → load → 落库，BaseHash 乐观锁，任一步失败回滚运行态
```

单一写入器是一步建站的承重墙：没有它，建站、抽屉、WAF 面板各写各的，confirm 弹窗里那份 diff 就不可信。这就是阶段 0 必须先行的原因。

### 3.4 证书中心（Jev 决策：双源合并库存，置信 0.77）

**真相源分工**：

| 源 | 提供什么 | 覆盖 |
|---|---|---|
| 卷扫描：读 `/data/caddy/certificates/**` 下 PEM，解析 NotBefore/NotAfter/Issuer/SAN | 有效期、签发者等**真值** | 仅 local 服务器（同容器同用户，读权限天然满足） |
| Caddyfile 推导：解析站点域名 + tls 指令 | "应有证书"清单、签发意图（auto/dns/manual） | 全部服务器，含 remote |

两源按 `(serverID, 主域名)` 合并；remote 服务器行标注"远端（仅声明）"，有效期列显 unknown——盲区如实呈现，不假装知道。

**表结构**（`model/caddy/certificate.go`）：

```go
type Certificate struct {
    ID           uint   // 主键
    ServerID     uint   // 关联 caddy_servers
    Domains      string // SAN 列表，逗号分隔
    PrimaryDomain string
    Issuer       string // Let's Encrypt / LogFlux 内部 CA / 上传
    Source       string // acme_http / acme_dns / internal / upload / manual_declared
    NotBefore    time.Time
    NotAfter     time.Time
    StoragePath  string // 卷内路径或 /config/certs/ 路径
    Status       string // valid / expiring / expired / missing / pending
    LastSyncedAt time.Time
}
```

**同步**：`internal/tasks/cert_sync.go` 每日任务（复用现有调度器）+ 证书中心手动刷新按钮。Caddy 自动续期会原地更新 PEM 文件，扫描即感知，无需事件钩子。

**到期通知**：扫描时按 D-14（warning）/ D-7（warning）/ D-1（critical）/ 过期（critical）发事件，新增事件组：

```go
// get_event_list_logic.go 追加
types.EventItem{Value: notification.EventCertExpiring,  Label: "证书即将到期", Group: "证书事件"},
types.EventItem{Value: notification.EventCertExpired,   Label: "证书已过期",   Group: "证书事件"},
```

渠道/规则/模板/站内信全复用现有通知系统——这是雷池社区版的弱项，是 LogFlux 的强项，零新架构。

**DNS-01（Cloudflare）**：全局设置页声明"DNS 提供商 = cloudflare"；生成的 Caddyfile 全局块写入 `{$CF_API_TOKEN}` 占位；token 经 `docker-compose.yml` environment 注入。**token 不进数据库、不进前端**。第一版仅此一路（零后端代码，文档说明注入方法）；alidns/dnspod = 构建加插件 + 同款占位，属阶段 5。

**上传**：`POST /api/caddy/certificates`（multipart：PEM + key + 域名 + serverID）→ 校验证书与私钥配对（`tls.X509KeyPair`）→ 写 `/config/certs/<serverID>/<domain>/{fullchain.pem,privkey.pem}` → 入库 `Source=upload`。remote 服务器上传不支持（无文件通道），表单按服务器类型禁用并说明。

**自动续期**：ACME/internal 证书由 Caddy 全权续期，面板只读展示；upload/manual 证书到期前只发通知提醒人工更换。**面板不做任何签发编排**——这是与雷池（nginx 需自建 ACME）的架构优势，声明意图、展示结果，不代理 Caddy 职责。

### 3.5 构建调整清单

| 阶段 | `docker/caddy.Dockerfile` 改动 |
|---|---|
| 阶段 1–2 | 无 |
| 阶段 3 | 无（cloudflare 已在） |
| 阶段 5（可选） | `--with github.com/caddy-dns/alidns --with github.com/caddy-dns/dnspod`；`--with github.com/mholt/caddy-ratelimit`（CC 防护） |

---

## 4. 阶段划分（每阶段可独立上线）

> **执行进度（2026-10-25）**：阶段 0 全部完成（P0 死代码/integration 端点/scope 键清理、P1 modules 过期修复、P2 单一写入器 `writer.Apply` 收敛三条写路径 + rollback 补 baseHash + 3 个语义锁定测试）；阶段 1 完成：能力识别层（`site-capabilities.ts`，三态 parse + 定点替换，含幂等/字节稳定/CRLF 测试）、一步建站弹窗（表单→diff 确认→adapt 校验→load→防护跟随全局自动挂载）、删除 6 步向导及全部接线、**站点卡片能力徽章 + 站点抽屉**（基本 Tab 外科编辑域名/上游/TLS、防护 Tab 六开关逐项覆盖全局默认、复杂站点原文只读 + 防护仍可调，`site-block-edit.ts` 含 CRLF/子块保持测试）。防护默认统一已按用户拍板落地（建站零防护问题，全局启用即自动挂载）。**待续**：阶段 3 证书中心；可选：防护开关与全局策略联动的提示细化。验证基线：后端 `go build`/`go test ./...` 全绿，前端 typecheck 通过、vitest 64 文件 523 用例全绿。

### 阶段 0：工程债清偿（= framework 计划 P0+P1+P2）

直接执行 `caddy-config-framework-refactor-plan.md` 的 P0（删死代码约 690 行）→ P1（修 modules 过期数据丢失）→ P2（单一写入器 writer.Apply）。验收见原文档。**不完成 P2 不开阶段 1 的保存链路。**

### 阶段 1：站点识别 + 一步建站 + 抽屉基本 Tab

**改动**：
- 新增 `views/caddy/sites/`：`SiteCreateModal.vue`（单表单）、`SiteDrawer.vue`（抽屉，本阶段只开放基本 Tab）、`site-capabilities.ts`（能力 parse，只读）
- 服务目录改主入口：卡片网格 + 「添加站点」按钮 + 能力/证书徽章位
- `caddy-config-blocks.ts` 后置能力提取；`applyCapabilityToSiteBlock` 纯函数 + 幂等单测
- diff 确认弹窗组件（复用现有 `caddy-config-diff`）
- **删** `SiteWizardModal.vue`、`site-wizard-utils.ts`（含其测试）
- 保存链路：表单 → 定点替换 → 弹窗 diff → 确认 → `writer.Apply`

**验收**：
- 新建反代站点：1 表单 1 确认 ≤ 5 次点击，`/load` 成功后服务目录出现新卡片
- 改既有站点上游：Caddyfile 其余行**字节级不变**
- 线上 4 个站点中 ≥ 3 个的 `rawExtras` 为空（能力识别率）
- 保存失败：Caddy 原始错误展示在弹窗内，库与运行态均无变化

### 阶段 2：防护 Tab（能力写回）

> **决策修订（2026-10-25，用户拍板 + Jev 置信 1.0）**：防护策略默认统一——新建站点不问任何防护问题，全局 WAF 启用则保存后自动挂载（`import waf_protect`），未启用则不挂；站点级差异（WAF 开关、IP 限制、安全头、错误页）在抽屉防护 Tab 里按站点覆盖，不单独设站点级策略对象，强度等参数仍全局一份。已落地：建站表单的 WAF 开关已删，改为自动继承。

**改动**：`applyCapabilityToSiteBlock` 接通写路径；抽屉防护 Tab 四开关（WAF / IP 限制 / 安全头 / 错误页）；全局 `ip_region` 降级为"默认值"，站点可覆盖；`useSimpleWaf.ts` 从全局表单改为当前站点上下文。

**验收**：任一能力开关切换幂等；给站点关 WAF，Caddyfile 恰好少一行 `import waf_protect`；"平衡→严格"只改 `waf_policies` 一行，Caddyfile 不变。

### 阶段 3：证书中心

**改动**：
- 后端：`model/caddy/certificate.go` + AutoMigrate；`api/caddy_certificate.api`（list / upload / refresh / delete）；`logic/caddy/cert_sync_service.go`（双源扫描合并）；`internal/tasks/cert_sync.go` 每日任务；通知事件 2 项 + 发送点
- 前端：`views/caddy/certificates/`（库存表 + 缺口视图 + 上传）；建站表单与抽屉 TLS Tab 接"证书下拉/上传"；服务目录证书徽章
- 文档：`docker/README.md` 补 CF_API_TOKEN 注入说明

**验收**：
- 自动签发的站点，建站确认后 ≤ 2 分钟内证书中心出现该证书（触发一次手动刷新或等每日任务）
- 到期 D-14 触发通知（造一张临近过期证书验证）
- 上传证书 → 选它建站 → `/load` 成功 → 站点 HTTPS 可访问
- remote 服务器证书行显示"远端（仅声明）"

### 阶段 4：旧页面收敛

配置管理页降级：blocks/raw/waf 视图收进「高级管理」，顶部导航只留「服务目录 / 证书中心 / WAF 来源 / 全局设置」；服务目录卡片不再跳转旧页面。（框架计划的 P3 单一变换层建议在本阶段前完成，用于抽屉与全局设置页共用文本手术。）

### 阶段 5（可选，按需启动）

CC 频次限制（caddy-ratelimit 模块 + 站点能力开关 + IP 维度 zone）· alidns/dnspod 插件 · 日志导出 CSV · 站点级黑白名单细化。

---

## 5. 明确不做

- **不改"确认后才热加载"语义**（Jev 0.87）：只压缩确认形态，不取消确认
- **不做 ACME 签发编排**：Caddy 全权负责签发与续期，面板声明意图 + 展示结果
- **不做 remote 服务器证书文件扫描**：无文件通道，盲区显式标注
- **不做站点级 WAF 策略对象**：强度等参数仍是全局 `waf_policies` 一份，站点只做"挂载/取消挂载"级的覆盖（默认继承全局，见阶段 2 决策修订）
- **不新建 site_capabilities 表**：Caddyfile 文本仍是唯一真相源
- **不删 preservedBlocks**：全局 snippet 与不可识别块仍需它，只是不再吞掉整个站点
- **token 不进数据库**：DNS token 只走环境变量

---

## 6. 风险与对策

| 风险 | 对策 |
|---|---|
| 一步建站直接改生产 Caddyfile | diff 确认弹窗 + BaseHash 乐观锁 + adapt/load 失败不落库（阶段 0 保障） |
| 定点替换破坏用户手写注释/缩进 | `applyCapabilityToSiteBlock` 只增删能力自身行，插在块头部，幂等有单测；其余字节不动 |
| 证书扫描与 Caddy 续期竞态 | 只读扫描，PEM 解析失败跳过该文件下次再试；Daily 全量兜底 |
| 上传私钥落盘安全 | 文件权限 0600，仅 certRef 引用路径，前端永不回显内容 |
| CF_API_TOKEN 环境变量未配置导致 dns 签发静默失败 | 建站表单选 DNS 签发时预检：后端探测 token 是否已注入（只报有无，不读值） |
| remote 盲区被误读为"没证书" | 库存行 source 列显式"远端（仅声明）"，缺口视图只对 local 服务器生效 |

---

## 7. 与既有文档的关系

| 文档 | 关系 |
|---|---|
| `caddy-config-framework-refactor-plan.md` | **保持有效**，作为阶段 0（P0–P2）；P3–P6 与本方案并行，P3 建议在阶段 4 前完成 |
| `caddy-site-capability-model-plan.md` | **被本方案取代**（站点模型、能力三态、抽屉设计已吸收并修订：TLS 扩展 dns/certRef、保存语义定为确认制） |

---

## 8. 总验收

| 场景 | 标准 |
|---|---|
| 新站上线 | 打开服务目录 → 添加站点 → 填 2 项 → 确认 diff → 完成，全程 ≤ 5 次点击、不离开服务目录 |
| HTTPS | 选自动签发的新站，建站即 HTTPS；证书 2 分钟内出现在证书中心 |
| 到期告警 | 临近过期证书触发 Telegram/邮件通知（复用现有通知规则） |
| 安全底线 | 任何保存失败：运行态回滚、库不变、错误原文可见；任何热加载必经确认 |
| 兼容 | 现有 4 个线上站点（含 3 个 import cloudflare 复杂站）能力识别 ≥ 3 个，rawExtras 为空 |
