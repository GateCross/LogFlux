# LogFlux 安全管理功能优化任务清单（执行版）

> 对应方案文档：`docs/plans/security-management-optimization-execution-plan.md`

## 1. 里程碑与范围

- 目标：在不回退现有安全管理能力的前提下，完成信息架构收束、前端结构重构、策略工作区化、Observe 拆分、前端 API 按域拆分，以及后端发布链路服务化。
- 范围：`/security` 安全管理模块前后端优化。
- 不含：新增高风险安全能力、大规模 API 协议重构、权限体系重构、视觉系统重做。

## 2. 任务优先级说明

- `P0`：必须完成，缺失会影响整体方案成立或后续阶段推进
- `P1`：建议优先完成，直接影响可维护性与用户理解成本
- `P2`：增强项，用于提升稳定性、测试覆盖和长期演进效率

## 3. 阶段任务清单

## 阶段 A：方案冻结与基线确认（P0）

- [ ] `P0-A01` 评审并确认 `docs/plans/security-management-optimization-execution-plan.md`
- [ ] `P0-A02` 确认新的一级领域划分：规则来源 / 策略中心 / 观测与处置 / 发布运维
- [ ] `P0-A03` 梳理旧路由与新领域的兼容映射关系
- [ ] `P0-A04` 列出需要保留的关键用户路径（source / policy / observe / release / job）
- [ ] `P0-A05` 产出回归验证清单并作为后续阶段统一验收基线

**验收标准**
- [ ] 团队对重构范围、阶段边界、兼容策略达成一致
- [ ] 旧入口与新结构之间的映射关系明确
- [ ] 回归基线可用于各阶段验收

## 阶段 B：一级信息架构收束（P0/P1）

- [ ] `P0-B01` 收敛 `/security` 一级导航模型
- [ ] `P0-B02` 设计统一导航 schema，覆盖 menu / route / legacy tab / default child
- [ ] `P0-B03` 调整页面标题、高亮、默认跳转逻辑
- [ ] `P1-B04` 保持旧 deep link 兼容并正确落到新领域
- [ ] `P1-B05` 收敛现有 route/menu/tab 多份映射配置

**验收标准**
- [ ] 用户可通过 4 个一级领域理解安全管理结构
- [ ] 页面标题、导航高亮、浏览器前进后退行为正确
- [ ] 旧链接与历史书签仍可访问正确页面

## 阶段 C：前端壳层化与领域容器拆分（P0/P1）

- [ ] `P0-C01` 将 `frontend/src/views/security/index.vue` 降级为壳层页面
- [ ] `P0-C02` 新增 `SecurityShell` 或等价页面编排层
- [ ] `P1-C03` 新增 `SecuritySourcePage`
- [ ] `P1-C04` 新增 `SecurityPolicyPage`
- [ ] `P1-C05` 新增 `SecurityObservePage`
- [ ] `P1-C06` 新增 `SecurityOpsPage`
- [ ] `P1-C07` 下沉现有模板与状态，减少壳层业务负担
- [ ] `P1-C08` 建立统一领域刷新器，替代按 tab 分支刷新

**验收标准**
- [ ] 主壳层仅承担导航、路由兼容、共享挂载、跨域协调职责
- [ ] 领域容器边界清晰
- [ ] 原有主要交互行为保持不变

## 阶段 D：策略中心工作区化（P1）

- [ ] `P1-D01` 将 `runtime/crs/exclusion/binding` 收束为同一策略工作区
- [ ] `P1-D02` 设计策略列表 + 工作区布局
- [ ] `P1-D03` 统一基础配置、CRS、例外、绑定的页面入口
- [ ] `P1-D04` 统一预览 / 校验 / 发布操作区
- [ ] `P1-D05` 将 revision 历史与当前策略上下文绑定
- [ ] `P1-D06` 明确未保存草稿状态与提示机制

**验收标准**
- [ ] 用户可围绕单一策略完成主要配置流程
- [ ] CRS 不再以独立且割裂的流程暴露
- [ ] revision 与当前策略上下文一致

## 阶段 E：Observe 拆分优化（P1）

- [ ] `P1-E01` 将 Observe 拆为“效果分析”与“误报处置”两个子视图
- [ ] `P1-E02` 将统计、趋势、维度分析、导出归入分析视图
- [ ] `P1-E03` 将反馈列表、批量处理、状态流转、指派归入处置视图
- [ ] `P1-E04` 优化筛选、drilldown 与 URL query 同步逻辑
- [ ] `P2-E05` 保持 feedback -> exclusion 草稿链路顺畅

**验收标准**
- [ ] 分析与处置在页面结构上清晰区分
- [ ] 工具栏密度显著下降
- [ ] 批量处理、导出、drilldown 等核心能力保持可用

## 阶段 F：前端 API 与 composable 按域拆分（P1/P2）

- [ ] `P1-F01` 将 `service/api/caddy.ts` 逐步瘦身为兼容出口或过渡层
- [ ] `P1-F02` 将 source 域 API 实现与类型沉淀到独立文件
- [ ] `P1-F03` 将 policy 域 API 实现与类型沉淀到独立文件
- [ ] `P1-F04` 将 observe 域 API 实现与类型沉淀到独立文件
- [ ] `P1-F05` 将 ops 域 API 实现与类型沉淀到独立文件
- [ ] `P2-F06` 整理现有 composable，避免跨领域状态相互渗透
- [ ] `P2-F07` 为关键 composable 补充类型校验与必要测试

**验收标准**
- [ ] 新增或修改某个领域时不需要频繁改动中心 API 文件
- [ ] composable 与页面容器的职责边界清晰
- [ ] 类型导入路径简洁且领域明确

## 阶段 G：后端发布链路服务化（P1/P2）

- [ ] `P1-G01` 梳理 `publish_waf_policy_logic.go` 当前职责边界
- [ ] `P1-G02` 抽离 directives 构建服务
- [ ] `P1-G03` 抽离 candidate config 应用服务
- [ ] `P1-G04` 抽离 publish/load/rollback 执行服务
- [ ] `P1-G05` 抽离 revision/history 持久化服务
- [ ] `P2-G06` 抽离通知发送与审计事件辅助层
- [ ] `P2-G07` 为关键失败路径补充测试

**验收标准**
- [ ] publish logic 主文件显著收敛
- [ ] 配置生成、执行落地、失败回滚边界明确
- [ ] 发布失败定位与回滚验证更容易执行

## 阶段 H：回归与验收（P0）

- [ ] `P0-H01` 回归规则来源：source CRUD / upload / engine status / engine check
- [ ] `P0-H02` 回归策略中心：新增 / 编辑 / 删除 / 预览 / 校验 / 发布 / 回滚
- [ ] `P0-H03` 回归 CRS / exclusion / binding 相关流程
- [ ] `P0-H04` 回归 Observe：统计 / drilldown / feedback / 批量处理 / 导出
- [ ] `P0-H05` 回归 Release / Job：激活 / 回滚 / 列表 / 清理
- [ ] `P1-H06` 执行 `pnpm typecheck`
- [ ] `P2-H07` 补充必要的前端交互测试与后端关键路径测试

**验收标准**
- [ ] source / policy / observe / release / job 主流程回归通过
- [ ] 类型检查通过
- [ ] 关键深链与旧入口兼容可用

## 4. 建议实施顺序

1. 阶段 A：先冻结方案和回归基线
2. 阶段 B：先做信息架构收束
3. 阶段 C：再做壳层化和领域容器拆分
4. 阶段 D：把策略中心工作区化
5. 阶段 E：拆分 Observe 分析与处置
6. 阶段 F：完成 API 与 composable 按域拆分
7. 阶段 G：最后做后端发布链路服务化
8. 阶段 H：每阶段完成后都执行回归，最终做一次全链路验收

## 5. 工时预估（可并行）

- 阶段 A + B：约 1~2 人日
- 阶段 C：约 2~4 人日
- 阶段 D：约 2~4 人日
- 阶段 E：约 1~3 人日
- 阶段 F：约 2~3 人日
- 阶段 G：约 2~4 人日
- 阶段 H：约 1~2 人日
- 合计：约 11~22 人日

## 6. 上线前 DoD 清单

- [ ] 安全管理一级入口语义清晰
- [ ] `index.vue` 不再是主要业务实现载体
- [ ] 策略相关能力收束到统一工作区
- [ ] Observe 分析与处置分区明确
- [ ] 前端 API 与 composable 具备稳定领域边界
- [ ] 后端发布链路完成服务层收敛
- [ ] 主流程回归通过
- [ ] 旧路由与关键深链兼容可用

## 7. 文档任务完成记录

- [x] `DOC-01` 执行方案文档：`security-management-optimization-execution-plan.md`
- [x] `DOC-02` 任务清单文档：`security-management-optimization-task-checklist.md`
- [ ] `DOC-03` 页面结构映射文档（可选）
- [ ] `DOC-04` 路由兼容映射说明（可选）
- [ ] `DOC-05` 分阶段进度跟踪文档（可选）
