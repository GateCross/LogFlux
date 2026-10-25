# Caddy 配置能力化重构：站点 + 能力模型

> **已被取代（2026-10-25）**：本文档的核心设计（站点模型、能力三态、抽屉）已吸收进 `caddy-site-first-cert-center-plan.md`，后续以该方案为准。

> 更新：2026-10-21
> 目标：把"编辑 Caddyfile"升级为"管理站点 + 声明能力"
> 范围：`frontend/apps/src/views/caddy/`、`backend/internal/logic/caddy/`

---

## 0. 一句话诊断

**现在四个页面（配置管理 / 服务目录 / 访问控制 / WAF 来源）都是同一份 Caddyfile 的不同皮肤，但用户脑子里装的是"站点"。**

证据：

| 用户想做的事 | 现在要去哪 | 实际改了什么 |
|---|---|---|
| 给某站点开 WAF | 配置管理 → 点「防火墙」按钮 | Caddyfile 的 directives 块 + waf_policies 表 |
| 给某站点设 IP 限制 | 访问控制页 | `system_config` 的 `ip_region` |
| 给某站点加日志 | 配置管理 → blocks 视图 → 改站点块 | Caddyfile |
| 看某站点状态 | 服务目录（只读） | 不改，点一下就跳走 |

四个入口，三份存储，用户要在脑子里自己拼装"pve.example.com 这个站现在到底是什么状态"。

而 `testdata_live_caddyfile.txt` 证明了这个拼装有多难：4 个站点，3 个带 `import cloudflare` + `import geo_auth` + `import deny_403_redirect`，1 个 `:80` 带 `route @lan` 分流 —— **全部掉进 `preservedBlocks`（只读保留）**。用户打开 blocks 视图看到的是"复杂站点，无法结构化编辑"，只能去 raw 模式手改。

**`preservedBlocks` 不是"保留"，是"投降"。**

---

## 1. 新模型：站点是唯一对象，能力是勾选项

### 1.1 数据结构

```typescript
/** 站点：用户心智里的唯一对象 */
interface Site {
  id: string;                    // 稳定 UUID，存在 modules JSON 里
  name: string;                  // 用户命名，可空（回退到首域名）
  domains: string[];             // 域名或监听地址，如 ['pve.example.com'] 或 [':80']
  enabled: boolean;

  tls: TlsConfig;
  upstream: UpstreamConfig;
  capabilities: SiteCapabilities;
  rawExtras: string;             // 站点级逃生舱，只影响这个站点
}

interface TlsConfig {
  mode: 'auto' | 'internal' | 'off' | 'manual' | 'dns';
  certFile?: string;             // manual
  keyFile?: string;              // manual
  dnsProvider?: 'cloudflare' | 'alidns' | 'dnspod';  // dns
  dnsTokenRef?: string;          // 引用密钥，不存明文
}

interface UpstreamConfig {
  kind: 'proxy' | 'static' | 'redirect' | 'respond';
  target?: string;               // proxy: 单一目标；static: 目录；redirect: URL
  pool?: string;                 // proxy: 上游池名
  lbPolicy?: 'round_robin' | 'least_conn' | 'ip_hash';
  healthCheck?: { path: string; interval?: string; timeout?: string };
  transport?: { tls: boolean; insecureSkipVerify?: boolean; readTimeout?: string };
  headerUp?: string[];           // 如 ['Host {upstream_hostport}', '-Via']
  headerDown?: string[];
  status?: number;               // respond
  body?: string;                 // respond
  code?: number;                 // redirect
}

/** 能力：每个都是"开/关 + 少数枚举值"，不是文本 */
interface SiteCapabilities {
  waf: { mode: 'off' | 'detect' | 'block'; strength: 'low_fp' | 'balanced' | 'high_blocking' };
  geo: { mode: 'off' | 'allow'; allowList: string[] };
  compression: { mode: 'off' | 'on'; algorithms: Array<'zstd' | 'gzip'> };
  securityHeaders: { preset: 'off' | 'basic' | 'strict' };
  accessLog: { mode: 'off' | 'on'; template: string; rollSize?: string; rollKeep?: number };
  errorPage: { mode: 'off' | 'redirect'; to: string; code: 302 | 301 };
  healthEndpoint: { mode: 'off' | 'on'; path: string };
}
```

### 1.2 这个模型能表达什么

用 `testdata_live_caddyfile.txt` 验证：

```typescript
// pve.example.com —— 现在被判"复杂站点，无法编辑"
{
  id: 'a1b2...', name: 'Proxmox', domains: ['pve.example.com'],
  tls: { mode: 'dns', dnsProvider: 'cloudflare', dnsTokenRef: 'cf-main' },
  upstream: {
    kind: 'proxy', target: 'https://192.0.2.100:8006',
    transport: { tls: true, insecureSkipVerify: true },
  },
  capabilities: {
    waf: { mode: 'block', strength: 'balanced' },
    geo: { mode: 'allow', allowList: ['中国'] },
    errorPage: { mode: 'redirect', to: 'https://deny.example.com', code: 302 },
    accessLog: { mode: 'on', template: '/var/log/caddy/pve.log' },
    compression: { mode: 'off' },
    securityHeaders: { preset: 'off' },
    healthEndpoint: { mode: 'off', path: '' },
  },
  rawExtras: '',
}
```

**结果：`rawExtras` 为空。** 这个站点现在能完整结构化编辑了。

```typescript
// :80 —— 内网分流，确实需要逃生舱
{
  id: 'c3d4...', name: '内网入口', domains: [':80'],
  tls: { mode: 'off' },
  upstream: { kind: 'static', target: '/app/frontend' },
  capabilities: {
    waf: { mode: 'block', strength: 'balanced' },
    compression: { mode: 'on', algorithms: ['zstd', 'gzip'] },
    accessLog: { mode: 'on', template: '/var/log/caddy/lan.log', rollSize: '10MB', rollKeep: 10 },
    geo: { mode: 'off', allowList: [] },
    securityHeaders: { preset: 'off' },
    errorPage: { mode: 'off', to: '', code: 302 },
    healthEndpoint: { mode: 'off', path: '' },
  },
  rawExtras: `
@lan remote_ip 127.0.0.1/8 10.0.0.0/8 172.16.0.0/16
route @lan {
  @ikuai host ikuai.example.com
  handle @ikuai { reverse_proxy https://192.0.2.254:443 }
  handle { import health_check; import geo_auth; import api_proxy }
}
route { redir https://{host}{uri} permanent }
`,
}
```

**结果：WAF、压缩、日志三项被结构化识别，只有分流逻辑留在 `rawExtras`。**

对比现在：整个 `:80` 块（55 行）全部进 `preservedBlocks`，用户连"这个站开了 WAF 吗"都要自己读文本。

---

## 2. 能力契约：render / parse / 三态

每个 capability 必须实现两个纯函数 + 一个状态判断：

```typescript
interface Capability<T> {
  /** 从站点块文本识别当前值；识别不出返回 null */
  parse(siteBlock: string): T | null;
  /** 生成这个能力对应的 Caddyfile 行（不含缩进，调用方统一缩进） */
  render(value: T): string[];
  /** 是否启用（决定 UI 显示开关 on/off） */
  isActive(value: T): boolean;
}
```

### 2.1 三态是核心

`parse` 返回 `null` 表示"文本里有相关内容，但我认不出来"：

| 状态 | UI 表现 | 用户操作 |
|---|---|---|
| `on` | 开关打开，显示当前值 | 可直接改 |
| `off` | 开关关闭 | 可直接开 |
| `unknown` | 开关**禁用**，提示"检测到无法识别的 WAF 配置" | 只能去高级模式改 |

`unknown` 是安全阀。现在 `analyzeSiteForQuickConfig` 的做法是"发现有 import 就整体判 complex"（`quick-config-utils.ts:206`），这是全有或全无。三态让它变成"WAF 认不出来，但压缩和日志我认出来了"。

### 2.2 示例：WAF 能力

```typescript
const wafCapability: Capability<SiteCapabilities['waf']> = {
  parse(siteBlock) {
    if (/^\s*import\s+waf_protect\s*$/m.test(siteBlock)) {
      // 开关是开的，但强度在全局策略里，站点层面只知道开/关
      return { mode: 'block', strength: 'balanced' };
    }
    if (/^\s*#\s*logflux:waf-off\s*$/m.test(siteBlock)) {
      return { mode: 'off', strength: 'balanced' };
    }
    // 有别的 WAF 相关行（如内联 coraza_waf），认不出来
    if (/coraza_waf|modsecurity/i.test(siteBlock)) return null;
    return { mode: 'off', strength: 'balanced' };
  },

  render(value) {
    if (value.mode === 'off') return [];
    // 强度和模式在 waf_policies 里，站点只需挂载
    return ['import waf_protect'];
  },

  isActive: (v) => v.mode !== 'off',
};
```

**注意 `render` 只产出 `import waf_protect`。** 强度/模式仍由 `waf_policies` 表和 `buildWafPolicyDirectives` 决定 —— 保持现有机制，不新建真相源。

但 UI 上用户看到的是"这个站点：阻断 / 平衡"，而不是"这个站点 import 了 waf_protect，全局策略是 balanced"。

### 2.3 各能力的 render 输出

| 能力 | render 产出 |
|---|---|
| `waf` | `import waf_protect` |
| `geo` | `import geo_auth` |
| `compression` | `import compression` 或 `encode zstd gzip` |
| `securityHeaders` | `import security_headers` 或内联 `header { ... }` |
| `accessLog` | `import subdomain-log <name>` 或内联 `log { ... }` |
| `errorPage` | `import deny_403_redirect` |
| `healthEndpoint` | `handle /api/health { respond "OK" 200 }` |

**关键设计：优先用 `import`，snippet 定义放全局。**

理由：
- 站点块保持短（4~6 行），用户一眼能读完；
- snippet 复用，改一次全局生效（这正是 Caddy snippet 的设计意图）；
- `parse` 只需匹配 `import xxx`，比解析内联块可靠得多。

snippet 本身由"全局设置"页面管理，和站点解耦。

---

## 3. 新流程：三个页面 → 一个页面

### 3.1 现状

```
配置管理 ─┬─ blocks 视图（改站点）
          ├─ raw 视图（改全文）
          ├─ waf 视图（改全局 WAF）
          └─ preview 视图（看结果）
服务目录 ── 只读，点了跳走
访问控制 ── 改 system_config
WAF 来源 ── 改规则包版本
```

用户要在 4 个页面之间跳，每次跳都丢上下文。

### 3.2 新流程

```
服务目录（主入口）
  │  顶部：服务器选择 + 全局状态徽章
  │  主体：站点卡片网格
  │       每张卡：域名 / 上游 / 在线状态 / 近窗 4xx5xx / 能力徽章行
  │
  └─ 点卡片 → 右侧抽屉（不跳页）
       Tab 1  基本  域名 · 上游 · TLS
       Tab 2  防护  WAF · IP 限制 · 安全头 · 错误页
       Tab 3  高级  压缩 · 日志 · 健康端点 · 路由 · 逃生舱
       
       每个 Tab 底部一个「保存」→ adapt → load → 落库（三步原子）
       保存成功：抽屉顶部状态条变绿，卡片徽章同步更新
       保存失败：抽屉顶部显示 Caddy 原始错误，不落库

全局设置（独立页面，从服务目录右上角进入）
  只有真正全局的：admin API · trusted_proxies · auto_https · order · snippet 定义
  和站点无关，改它不会碰任何站点块

原文模式（降级为逃生舱，从站点抽屉「高级」Tab 进入）
  只读展示当前站点的完整块 + 可编辑
  保存时校验"只改了一个块"，否则拒绝
```

### 3.3 关键变化

| 维度 | 现在 | 新方案 |
|---|---|---|
| 主入口 | 配置管理页 | 服务目录 |
| 操作单位 | Caddyfile 全文 | 单个站点 |
| 站点详情 | 跳转到另一个页面的某个表单 | 抽屉，就地编辑 |
| WAF 配置 | 独立"防火墙"视图，全局 | 站点 Tab 2 的一个开关 |
| IP 限制 | 独立"访问控制"页面 | 站点 Tab 2 的一个开关 |
| 保存粒度 | 整份 Caddyfile | 一个站点块 |
| 生效方式 | 保存 → 预览 → 应用（三步） | 保存（一步，内部 adapt→load→落库） |
| 复杂站点 | "无法结构化编辑" | 能力被识别，只有分流逻辑进逃生舱 |

---

## 4. 数据流：Caddyfile 仍是唯一真相

这是硬约束，来自上一轮的框架分析。能力**不落新表**：

```
caddy_servers.config (TEXT)          ← 唯一真相
        │
        ├─ parseCaddyfileToBlocks()  → 站点块文本
        │       │
        │       └─ capability.parse(siteBlock) → SiteCapabilities
        │
        └─ 用户改能力
                │
                ├─ capability.render(newValue) → 指令行
                └─ applyCapabilityToSiteBlock(siteBlock, lines) → 新站点块
                        │
                        └─ 替换原块 → adapt → load → 落库
```

`caddy_servers.modules` 降级为**前端表单快照**（上一轮方案 P1 已定），后端不再从它渲染。

### 4.1 定点替换，不重建全文

```typescript
/**
 * 只改一个站点块内的能力行，其余字节不动。
 *
 * 比"重新渲染整个 Caddyfile"安全：用户手写的注释、缩进、顺序全部保留。
 */
export function applyCapabilityToSiteBlock(
  siteBlock: string,
  capabilityKey: keyof SiteCapabilities,
  lines: string[],   // capability.render() 的输出
): string;
```

实现要点：
- 先删掉这个能力当前产出的所有行（用 `parse` 的逆向匹配，如 `^\s*import waf_protect\s*$`）
- 再在站点块头部（`{` 之后、第一条其他指令之前）插入新行
- 保持原有缩进风格（从块内第一条非空行取）

这和现有的 `ensureSiteImport` / `removeSiteImport`（`waf_integration_helpers.go:205-238`）同构，但更精确：只动目标能力，不碰其他行。

### 4.2 幂等性保证

`applyCapabilityToSiteBlock(applyCapabilityToSiteBlock(block, k, v), k, v) === applyCapabilityToSiteBlock(block, k, v)`

必须有单测锁住。现在的 `ensureCorazaOrder` 等函数已有类似测试（`waf_simple_apply_e2e_test.go:87-98`），复用同样的断言风格。

---

## 5. 迁移路径：三步，每步可独立上线

### 第一步：能力识别层（只读，零风险）

**新增**：
- `frontend/apps/src/views/caddy/config/site-capabilities.ts` —— 类型 + 7 个 capability 的 `parse`
- 在 `parseCaddyfileToBlocks` 之后跑 `extractCapabilities(siteRaw)`，结果挂在 `Site.capabilities`

**改动**：
- 服务目录卡片增加"能力徽章"行：`WAF 阻断` `IP 限制` `TLS DNS` `日志`
- 复杂站点卡片不再只显示"复杂"，而是显示"WAF · IP · 日志（部分识别）"

**不动**：任何写路径。

**验收**：`pve.example.com` 卡片显示 `WAF 阻断 / IP 限制 / TLS DNS / 错误页`，`rawExtras` 为空。

### 第二步：站点抽屉（新增页面，旧页面保留）

**新增**：
- `views/caddy/sites/SiteDrawer.vue` —— 三 Tab 抽屉
- 路由不新增，用 `?site=<id>` query 控制打开

**改动**：
- 服务目录卡片点击 → 打开抽屉，不再跳转
- 抽屉内先只做 **读 + 改基本 Tab（域名/上游/TLS）**
- 保存走现有 `previewCaddyConfigApi` + `pushCaddyConfigApi`

**不动**：防护 Tab 先只读展示能力徽章。

**验收**：改 `pve.example.com` 的上游端口，保存后 Caddyfile 只有一行变化，其余 273 行字节级不变。

### 第三步：能力写回 + 防护 Tab

**新增**：
- `applyCapabilityToSiteBlock` 纯函数 + 单测
- 防护 Tab 的开关：WAF / IP 限制 / 安全头 / 错误页

**改动**：
- 点 WAF 开关 → `applyCapabilityToSiteBlock(block, 'waf', newValue)` → 新块 → 替换 → 保存
- 访问控制页面的 `ip_region` 全局开关改成"默认值"，站点可覆盖

**验收**：
- 给 `myserver.example.com` 关 WAF，Caddyfile 只少一行 `import waf_protect`
- 给 `pve.example.com` 从"平衡"改"严格"，只改 `waf_policies` 一行，Caddyfile 不变（因为强度是全局策略）

### 第四步（可选）：收敛 raw 视图

配置管理页的 `raw` / `preview` 两个一级视图降级为抽屉里的"高级"Tab，顶部导航只剩「服务目录 / 全局设置 / WAF 来源」。

---

## 6. 与现有代码的对应关系

| 现有 | 新方案 | 处置 |
|---|---|---|
| `types.ts` `Site` / `Route` / `Handle` | 新 `Site` + `UpstreamConfig` | 重写，但 `Route`/`Handle` 保留在 `rawExtras` 场景 |
| `quick-config-utils.ts` `analyzeSiteForQuickConfig` | `site-capabilities.ts` `extractCapabilities` | 替换（`simple/complex` 二分 → 能力三态） |
| `caddy-config-parser.ts` `parseCaddyfileToModules` | 保留 | 站点块文本提取仍需它 |
| `caddy-config-blocks.ts` `parseCaddyfileToBlocks` | 保留 + 后置能力提取 | 增量改 |
| `BlocksWorkbench.vue` | 拆分：站点列表 → 服务目录；站点表单 → SiteDrawer | 逐步废弃 |
| `WafPanel.vue` | 拆成 `SiteDrawer` 的防护 Tab + 全局设置页 | 逐步废弃 |
| `access/index.vue` | 合并进防护 Tab（站点级）+ 全局设置（默认值） | 逐步废弃 |
| `useSimpleWaf.ts` | 保留 API，`wafForm` 从全局改成"当前站点" | 改造 |
| `useCaddyConfigIO.ts` | 保留 `fetchConfig` / `confirmSave`，`previewBeforeSave` 简化 | 改造 |
| `ensureSiteImport` / `removeSiteImport` | 泛化成 `applyCapabilityToSiteBlock` | 演进 |
| `waf_integration_helpers.go` 文本手术 | 保留（后端仍需要） | 不动 |

---

## 7. 明确不做

- **不新建 `site_capabilities` 表**：Caddyfile 是唯一真相，能力是它的投影。
- **不做站点级 WAF 策略对象**：WAF 强度仍是全局 `waf_policies` 一行，站点只控制开/关。要做站点级强度，得等误报排除闭环落地后再说。
- **不重写前端 parser**：`parseCaddyfileToModules` 继续用于提取站点块文本。
- **不改 `rawExtras` 的编辑体验**：它就是纯文本，不提供任何结构化辅助。这是承认复杂度，不是掩盖它。
- **不删 `preservedBlocks`**：全局 snippet 和无法识别的块仍需要它，只是它不再吞掉整个站点。

---

## 8. 验收标准

| 阶段 | 验收 |
|---|---|
| 第一步 | 线上 4 个站点中至少 3 个 `rawExtras` 为空；能力徽章与实际 Caddyfile 一致 |
| 第二步 | 改单个站点的上游，Caddyfile 其余行字节级不变 |
| 第三步 | 任一能力开关的开启/关闭，`applyCapabilityToSiteBlock` 幂等；失败时 Caddyfile 不变 |
| 全部 | 用户从"打开页面"到"改完一个站点的 WAF"不超过 3 次点击、不离开当前页面 |
