# LogFlux WAF 更新管理操作指南（API 示例 + 运维 SOP）

> 对应设计：`docs/plans/waf-update-management-design.md`  
> 对应任务：`docs/plans/waf-update-task-checklist.md`  
> 订阅建议：`docs/plans/waf-update-notification-subscription-guide.md`

## 1. 使用目标

本指南用于上线后运维与值班场景，覆盖：
- 下载源配置与版本同步；
- 手动上传规则包；
- 激活、回滚与问题处置；
- 日常巡检项与告警建议。

## 2. 预置约定

- API 前缀：`/api`
- 鉴权：`Authorization: Bearer <token>`
- 返回结构：`{ code, msg, data }`
- WAF 工作目录：`/config/security`

## 3. 常用 API 操作示例

## 3.1 新增远程 CRS 源

```bash
curl -X POST http://localhost:8888/api/caddy/waf/source \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "official-crs",
    "kind": "crs",
    "mode": "remote",
    "url": "https://api.github.com/repos/coreruleset/coreruleset/releases/latest",
    "authType": "none",
    "schedule": "0 0 */6 * * *",
    "enabled": true,
    "autoCheck": true,
    "autoDownload": true,
    "autoActivate": false
  }'
```

## 3.2 手动检查新版本

```bash
curl -X POST http://localhost:8888/api/caddy/waf/source/1/check \
  -H "Authorization: Bearer <token>"
```

## 3.3 执行同步（下载+校验+可选激活）

```bash
curl -X POST http://localhost:8888/api/caddy/waf/source/1/sync \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"activateNow": true}'
```

## 3.4 手动上传规则包

```bash
curl -X POST http://localhost:8888/api/caddy/waf/upload \
  -H "Authorization: Bearer <token>" \
  -F "kind=crs" \
  -F "version=v4.23.0-custom.1" \
  -F "checksum=<sha256>" \
  -F "activateNow=true" \
  -F "file=@./crs-v4.23.0-custom.1.tar.gz"
```

## 3.5 查看 release 列表

```bash
curl "http://localhost:8888/api/caddy/waf/release?page=1&pageSize=20" \
  -H "Authorization: Bearer <token>"
```

## 3.6 激活指定版本

```bash
curl -X POST http://localhost:8888/api/caddy/waf/release/12/activate \
  -H "Authorization: Bearer <token>"
```

## 3.7 执行回滚

```bash
curl -X POST http://localhost:8888/api/caddy/waf/release/rollback \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"target": "last_good"}'
```

## 3.8 查看更新任务日志

```bash
curl "http://localhost:8888/api/caddy/waf/job?page=1&pageSize=50&status=failed" \
  -H "Authorization: Bearer <token>"
```

## 3.9 清理历史 release（保留 active 版本）

```bash
curl -X POST http://localhost:8888/api/caddy/waf/release/clear \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"kind":"crs"}'
```

## 4. 标准操作流程（SOP）

## 4.1 常规月度更新（推荐）

1. 手动执行 `check` 确认有新版本。  
2. 执行 `sync` 下载并校验，不立即激活。  
3. 在测试环境激活并回归关键接口。  
4. 生产环境激活，观察 30 分钟核心指标。  
5. 记录 release 与变更单号。

## 4.2 紧急安全更新（高危漏洞）

1. 立即执行 `sync`（可 `activateNow=true`）。  
2. 验证拦截规则命中与关键业务可用性。  
3. 若误报高，先回滚到 `last_good`，再做排除规则修正。  
4. 复盘并更新值班手册。

## 4.3 手动上传场景（离线/内网）

1. 离线环境下载规则包并生成 SHA256。  
2. 通过 `upload` 接口提交。  
3. 激活前执行 `/adapt` 校验（系统自动）。  
4. 激活并验证。

## 5. 失败场景与处置

## 5.1 下载失败

可能原因：
- 网络不可达
- 源地址错误
- Token 失效

处置：
- 检查 `waf_update_jobs.message`
- 校验源配置与凭证
- 先切手动上传保障可用

## 5.2 校验失败（SHA 不匹配）

可能原因：
- 包被篡改
- checksum 配置错误

处置：
- 立即终止激活
- 重新获取官方包与 checksum

## 5.3 解压失败（安全拦截）

可能原因：
- zip-slip
- 非法符号链接
- 文件数量/大小超限

处置：
- 使用可信来源重新打包
- 保留审计记录，禁止强制绕过

## 5.4 激活失败

可能原因：
- Caddy `/adapt` 失败
- include 路径不完整
- 规则语法错误

处置：
- 自动回滚 `last_good`
- 查看 Caddy admin 返回信息与 job 日志
- 修正后重试

## 6. 巡检与告警建议

## 6.1 每日巡检

- 最近 24h 是否有 `failed` 的更新任务
- 当前 `current` 是否与预期版本一致
- `waf_audit.log` 是否持续写入
- 磁盘空间是否低于阈值

## 6.2 每周巡检

- release 保留数量是否超阈值
- 是否有过期 source 凭证
- 是否存在长期未处理失败任务

## 6.3 告警建议

- 连续 3 次 `sync` 失败告警
- 激活失败且自动回滚告警（高优先）
- 工作目录磁盘使用率 > 80%
- 建议至少订阅：`security.waf_source_sync_failed`、`security.waf_release_activate_failed`、`security.waf_release_rollback_failed`

## 7. 运维检查命令（容器内）

```bash
# 1) 查看 current/last_good 指向
ls -l /config/security/current /config/security/last_good

# 2) 查看 release 目录
ls -lah /config/security/releases

# 3) 查看审计日志
tail -n 200 /var/log/caddy/waf_audit.log

# 4) 查看 Caddy 配置热加载日志
tail -n 200 /var/log/caddy/runtime.log
```

## 8. 发布后复盘模板（简版）

- 更新时间：
- 操作人：
- 源类型：`remote` / `upload`
- 目标版本：
- 是否激活成功：
- 回归结果：
- 指标变化（错误率/P95）：
- 是否回滚：
- 后续动作：

## 9. 发布前演练（H04）与 30 分钟观察（H05）模板

## 9.1 发布前演练记录（手动上传 + 激活失败自动回滚）

- 演练时间：
- 演练环境：
- 演练包版本：
- 演练步骤：
1. 上传规则包（`/api/caddy/waf/upload`）
2. 人为注入错误规则并触发激活
3. 验证 `current` 回到 `last_good`
4. 校验 `waf_update_jobs` 中存在失败审计与回滚记录
- 演练结果：通过 / 未通过
- 失败原因与整改：

## 9.2 发布后 30 分钟观察记录

- 观察窗口开始时间：
- 观察窗口结束时间：
- 关键指标：
1. 5xx 错误率
2. 请求 P95
3. WAF 拦截总数/误报反馈数
4. `security.waf_*_failed` 告警次数
- 观察结论：稳定 / 需回滚

