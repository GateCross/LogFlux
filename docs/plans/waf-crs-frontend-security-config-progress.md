# WAF/CRS 前端安全配置开发进度

更新时间：2026-02-13

## 当前已完成

- P0-A01 ~ P0-A05：完成策略模型、策略版本模型、AutoMigrate、默认策略初始化与结构化配置字段。
- P0-B01 ~ P0-B04：完成 `manage.api` 策略接口定义，并执行 `goctl api go -api api/logflux.api -dir . -style go_zero` 校验路由与类型生成一致性。
- P0-C01 ~ P0-C08：完成策略构建、预览、校验、发布、回滚、失败自动回滚（`last_good` 思路）与策略错误消息本地化。
- P0-D01 ~ P0-D08：完成前端策略 API 封装、`运行模式` Tab、策略表单、预览/校验/发布/回滚、发布记录表格与表单校验。
- P0-E01 ~ P0-E04：完成策略 builder 稳定性、字段越界/枚举校验、发布失败回滚分支单测，以及 preview/validate/publish/rollback 链路 smoke tests（新增 `waf_policy_builder_test.go`、`publish_waf_policy_logic_test.go`、`waf_policy_flow_smoke_test.go`）。
- P0-E05 ~ P0-E06：完成前端自测基础验证（`pnpm --dir frontend typecheck` + `pnpm --dir frontend build:test` 通过），并补齐 source/release/job 列表逻辑回归测试（新增 `list_waf_sources_logic_test.go`、`list_waf_releases_logic_test.go`、`list_waf_jobs_logic_test.go`）。
- P0-F01 ~ P0-F03：完成 `docs/README.md` 与 `docker/README.md` 的策略入口、发布/回滚操作和常见失败 FAQ 更新。
- P1-A01 ~ P1-A03：完成策略模型 CRS 调优字段扩展（模板/PL/入站阈值/出站阈值）、`manage.api` 与 `types` 同步、默认策略初始化补齐。
- P1-B01 ~ P1-B03：完成 CRS 调优参数校验与 directives 渲染（`tx.paranoia_level` / `tx.inbound_anomaly_score_threshold` / `tx.outbound_anomaly_score_threshold`），并接入策略错误本地化。
- P1-C01 ~ P1-C04：完成前端 `CRS 调优` 页签（模板切换、参数编辑、预览/校验/发布、PL>=3 风险提示、调优 revision 列表）。
- P1-D01 ~ P1-D03：补齐后端单测覆盖（builder + publish/policy flow 的 CRS 字段回归）并通过全量回归。
- P2-A01 ~ P2-A04：完成 `waf_rule_exclusions` / `waf_policy_bindings` 模型、AutoMigrate、`manage.api` 新接口定义与 goctl 生成接入。
- P2-B01 ~ P2-B03：完成规则例外与策略绑定后端 CRUD、字段校验（scope/removeType/method/priority）与中文错误本地化。
- P2-B04：完成发布链路冲突门禁（同作用域+同优先级绑定冲突阻断发布）与带例外指令的 preview/validate/publish 渲染。
- P2-C01 ~ P2-C04：完成前端 `规则例外` + `策略绑定` 页签（查询/分页/新增/编辑/删除、作用域条件字段、风险提示）。
- P2-C05：完成策略绑定冲突可视化与“策略叠加执行顺序（当前列表）”预览，便于发布前自检。
- P2-D01：补充 `waf_policy_scope_helpers_test.go`，覆盖例外指令渲染与绑定冲突检测分支。
- P3-A01 ~ P3-A03：新增策略观测接口 `GET /api/caddy/waf/policy/stats`，实现按策略绑定作用域统计命中/拦截/放行/疑似误报，并返回趋势序列。
- P3-A04：策略观测接口新增 `topN` 与维度下钻返回（`topHosts/topPaths/topMethods`），支持按主机/路径/方法快速定位高风险流量入口。
- P3-A05：策略观测接口新增 `host/path/method` 过滤参数，支持前端点击维度值后联动下钻（同口径刷新 summary/list/trend/top 维度）。
- P3-A06：策略观测接口下钻过滤与 Top 维度统计完成同口径耦合，支持在下钻态下继续查看/导出聚合结果。
- P3-B01 ~ P3-B03：增强策略 revision 审计返回（`policyName`、`changeSummary`），支持在发布记录中展示“谁在何时改了什么”。
- P3-C01 ~ P3-C03：接入通知联动，新增 WAF 策略发布/回滚成功与失败、自动回滚事件，并在 publish/rollback 链路中发出通知事件。
- P3-C04：前端新增 `策略观测` 页签（统计筛选、总览指标、趋势表、策略统计表）并接入后端统计接口。
- P3-C05：前端策略观测新增 Top Host/Path/Method 三组下钻视图与 CSV 导出按钮，支撑离线复盘与排障协作。
- P3-C06：前端 Top Host/Path/Method 行点击下钻已打通，页面展示当前过滤条件并支持一键清空下钻。
- P3-C07：前端下钻体验增强：增加层级提示（Host→Path→Method）、当前下钻标签（可按层级关闭）、维度表行选中高亮与禁点态提示。
- P3-C08：前端 Top 维度卡片新增锁定状态图标与悬浮说明（已解锁/待解锁），并在禁点行增加 hover 提示，降低误操作成本。
- P3-C09：前端策略观测页接入 URL query 状态同步（`activeTab/policyId/window/intervalSec/topN/host/path/method`），支持刷新与回退后恢复筛选/下钻上下文，并增加循环保护避免 watcher 相互触发。
- P3-C10：前端策略观测页新增“复制筛选链接”按钮，支持一键分享当前观测条件与下钻上下文，便于排障协作。
- P4-A01：新增误报反馈闭环基础能力：`waf_policy_false_positive_feedbacks` 模型、`GET/POST /api/caddy/waf/policy/false-positive-feedback` 接口、后端字段归一化校验与本地化错误映射。
- P4-A02：前端策略观测页接入“标记误报”提交流程与“人工误报反馈（当前筛选口径）”列表，支持按当前策略/下钻条件联动查询与分页查看。
- P4-A03：误报反馈新增状态流转（`pending/confirmed/resolved`）与处理备注能力，支持后端 `PUT /api/caddy/waf/policy/false-positive-feedback/:id/status` 更新、前端状态筛选与“处理反馈”弹窗闭环。
- P4-A04：误报反馈新增责任归属与 SLA 字段（`assignee/dueAt/isOverdue`）及后端筛选能力（`assignee/slaStatus`），支持按“正常/超时/已解决”定位待处理项。
- P4-A05：前端策略观测页误报反馈表补充责任人、截止时间、SLA 标签与筛选条件，处理弹窗支持更新责任人与截止时间，形成可执行的 SLA 管理闭环。
- P4-B01：前端策略观测页新增“对比基线快照 + 导出对比 CSV”能力，支持对比当前与上一次查询口径的 summary/policy 统计差异，提升复盘效率。
- P4-B02：误报反馈批量处理闭环已落地：反馈表支持多选、批量处理弹窗、批量状态流转与统一处理备注，减少逐条处理成本。
- P4-B03：新增后端批量处理接口 `PUT /api/caddy/waf/policy/false-positive-feedback/batch-status`，返回 `affectedCount/processedBy/processedAt` 审计信息并支持前端展示处理结果。
- P4-C01：误报反馈与规则例外联动已落地：反馈列表支持“生成例外草稿”，可自动带入策略/Host/Path/Method/建议内容并一键跳转到例外表单。
- P4-C02：策略观测对比导出已扩展 Top Host/Path/Method 差异块，支持维度级当前值/基线值/变化量离线复盘。
- P5-A01：补齐批量处理与例外草稿链路回归能力：新增批量处理后端单测（ID 去重/越界、无记录、非法截止时间、pending/confirmed 审计分支），并在前端支持跨页保留多选，观测口径变化时自动清空选择，降低误批量风险。
- P5-A02：增强“建议动作 -> 例外草稿”解析策略，支持 `removeById/removeByTag`、`ruleRemoveById/ruleRemoveByTag`、中文“移除规则/移除标签”等自然语言模板识别。
- P5-B01：新增“生成例外草稿”确认向导，先展示策略/作用域/移除类型与移除值草稿，再确认落入例外表单，降低误操作风险。
- P5-B02：确认向导支持“多候选 remove 值”选择（当建议文本匹配多个 ID/Tag 时可手工择一），减少解析歧义导致的二次编辑。
- P5-C01：确认向导升级为“可编辑确认”：支持在弹窗中直接修改策略/作用域/Host/Path/Method/规则名称/removeType/removeValue/描述后再生成草稿，降低二次返工。
- P5-C02：补充前端自动化回归用例：新增 `policy-feedback-draft.ts` 纯函数模块与 `policy-feedback-draft.test.ts`，覆盖向导候选值解析/选择、可编辑确认默认解析、跨页多选合并，以及引号/中文标点/异常 key 等边界样例，并提供 `pnpm --dir frontend test:regression` 脚本。
- P5-C03：当草稿缺少 `removeValue` 时，确认生成后自动聚焦例外表单“移除值”输入框，减少人工定位成本。
- 文档更新：补充 `docs/README.md` 与 `docker/README.md` 的 CRS 调优入口与操作说明。
- 工程验证：通过 `backend` 目录 `go test ./...`，通过 `frontend` 目录 `pnpm --dir frontend typecheck`、`pnpm --dir frontend build:test` 与 `pnpm --dir frontend test:regression`。
- P0-H02（补充）：新增 `check_waf_source_logic_test.go` 与 `sync_waf_source_logic_test.go`，覆盖 source check/sync 关键失败分支及作业审计写入路径回归。

## 未完成任务（重点）

- 无（P5 主线范围已落地）。

## 未完成任务（后续阶段）

- 无（草稿差异对比提示与异常文本边界样例已补齐）。

## 下一步建议

1. 继续保持当前 P5 主线功能冻结，以回归测试兜底稳定性。
2. 若进入下一轮迭代，优先补充更复杂的误报建议模板语料库（多语言与规则片段组合）。
