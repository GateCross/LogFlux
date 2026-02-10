# LogFlux 全面引入 Caddy + Coraza WAF + OWASP CRS 执行计划（一次性切换版）

## 1. 目标与范围

### 1.1 目标
- 在一次变更窗口内，将 LogFlux 当前网关全面升级为 **Caddy + Coraza WAF + OWASP CRS**。
- 切换后直接进入 **阻断模式**（`SecRuleEngine On`），覆盖线上入口流量。
- 保留现有 Caddy 功能（GeoIP、反代、静态资源、日志采集、Admin API 热加载）。

### 1.2 范围
- **包含**：
  - Caddy 镜像模块扩展（Coraza + CRS）
  - Caddy 配置改造（WAF 全量接入）
  - Docker 编排与挂载调整
  - 构建发布、验证、切换、回退
- **不包含**：
  - 应用后端业务逻辑修改
  - 新增额外安全网关产品

## 2. 接入策略（非分阶段）

### 2.1 切换原则
- 不再做 DetectionOnly 试运行，直接上线阻断。
- 在同一发布单内完成镜像构建、配置替换、服务重启与验证。
- 以“可快速回退”为核心约束，确保 5~10 分钟内可恢复旧版本。

### 2.2 目标拓扑

```
Client
  -> Caddy(coraza_waf + CRS, blocking enabled)
      -> /api/* -> backend:8888
      -> /*     -> frontend static
```

## 3. 一次性执行清单（Runbook）

## A. 变更前准备（必须在变更窗口前完成）

1. 代码与配置冻结
- 冻结 `docker/Caddyfile`、`docker/caddy.Dockerfile`、`docker/docker-compose.yml` 相关变更。
- 明确本次唯一发布分支，例如：`release/waf-full-cutover`。

2. 回退资产就绪
- 记录当前稳定镜像标签（应用镜像 + Caddy 镜像）。
- 备份当前 Caddy 配置（文件与数据库中的历史版本）。
- 确认后端回滚接口可用：`POST /api/caddy/server/:serverId/config/rollback`。

3. 资源容量确认
- 预估 WAF 启用后的额外 CPU/内存开销，预留容量。
- 确认日志盘空间可承载审计日志增长。

4. 验证样例准备
- 准备业务关键路径的回归请求集（登录、鉴权、查询、导出、管理操作）。
- 准备攻击样例（SQLi/XSS/路径穿越/异常请求头/超长参数）。

## B. 实施改造（一次提交）

1. 扩展 Caddy 模块
- 修改 `docker/caddy.Dockerfile`，在 `xcaddy build` 中加入 Coraza 与 CRS 模块。
- 保持现有模块不变：
  - `github.com/zhangjiayin/caddy-geoip2`
  - `github.com/caddy-dns/cloudflare`
  - `github.com/caddyserver/transform-encoder`

2. 新增 WAF 规则目录
- 新增目录（建议）：`docker/waf/`
- 规则文件（建议）：
  - `docker/waf/coraza.conf`
  - `docker/waf/crs-setup.conf`
  - `docker/waf/exclusions.conf`
  - `docker/waf/custom-rules.conf`

3. 改造 Caddy 配置
- 修改 `docker/Caddyfile`：
  - 全局添加：`order coraza_waf first`
  - 增加 WAF snippet，并在站点中 `import`
  - 启用 CRS + 自定义排除规则
  - 直接使用 `SecRuleEngine On`

4. 编排与挂载调整
- 修改 `docker/docker-compose.yml`，挂载 `docker/waf` 到容器内（只读）。
- 确保审计日志路径与现有日志卷一致（如 `/var/log/caddy`）。

5. 文档同步
- 修改 `docker/README.md`：新增 WAF 构建、运行参数、日志与回退说明。

## C. 构建与发布

1. 构建 WAF 版 Caddy 镜像
- `docker build -f docker/caddy.Dockerfile -t <caddy-image:waf-tag> .`

2. 构建应用镜像（引用新 Caddy）
- 确认 `docker/Dockerfile` 使用新的 `CADDY_IMAGE`。
- 构建并推送应用镜像。

3. 发布上线
- 更新 `docker/.env` 的镜像标签。
- 执行：`docker compose -f docker/docker-compose.yml up -d --no-build`

## D. 上线后立即验证（15~30 分钟内完成）

1. 服务健康
- `GET /api/health` 正常。
- 前端首页可访问，核心 API 可用。

2. WAF 生效验证
- 正常请求：业务无明显异常。
- 攻击样例：确认被拦截并写入审计日志。

3. 性能与稳定性观察
- 对比上线前基线：
  - 5xx 错误率不明显上升
  - P95 延迟在可接受区间
  - CPU/内存未超过预警阈值

## E. 回退流程（统一预案）

触发条件（任一满足即执行回退）：
- 关键业务链路持续失败（> 5 分钟）
- 错误率显著升高且无法快速定位
- WAF 拦截误报导致核心功能不可用

回退动作：
1. 镜像回退：将 `CADDY_IMAGE`/应用镜像切回上一稳定标签。
2. 重新拉起：`docker compose -f docker/docker-compose.yml up -d --no-build`
3. 配置回退：必要时调用配置历史回滚接口恢复旧 Caddy 配置。

回退目标：10 分钟内恢复到变更前稳定状态。

## 4. 建议基线配置（直接阻断版）

> 以下为模板，实际模块名和 include 路径以所选 Coraza Caddy 模块文档为准。

```caddyfile
{
  order coraza_waf first
}

(waf_protect) {
  coraza_waf {
    load_owasp_crs
    directives `
      Include @coraza.conf-recommended
      Include @crs-setup.conf.example
      Include @owasp_crs/*.conf

      SecRuleEngine On
      SecAuditEngine RelevantOnly
      SecAuditLogFormat JSON
      SecAuditLog /var/log/caddy/waf_audit.log

      Include /etc/caddy/waf/exclusions.conf
      Include /etc/caddy/waf/custom-rules.conf
    `
  }
}

:443 {
  import waf_protect
  # 其余既有 import 与路由保持不变
}

:80 {
  import waf_protect
  # 其余既有 import 与路由保持不变
}
```

## 5. 验收标准（一次切换版）

- 生产镜像中可确认 Coraza/CRS 模块存在并加载成功。
- 全站入口在阻断模式下稳定运行。
- 攻击样例可拦截，正常业务请求成功率保持基线水平。
- 审计日志可检索、可用于误报定位。
- 已验证回退流程可在 10 分钟内完成。

## 6. 风险声明

一次性全量阻断上线会显著提高误报与业务抖动风险。建议至少满足以下条件再执行：
- 业务方已提供完整回归流量与关键接口清单。
- 变更窗口内有开发、运维、业务三方值守。
- 已完成“镜像级 + 配置级”双回退演练。

---

## 附录：上线日简版操作卡

1. 发布新镜像并更新 compose 环境变量。
2. `docker compose up -d --no-build` 完成切换。
3. 立即执行健康检查与关键链路回归。
4. 执行攻击样例验证拦截是否生效。
5. 连续观察 30 分钟核心指标。
6. 不满足阈值立即执行回退。
