# LogFlux 文档中心

## 项目文档

### 核心文档
- [README.md](../README.md) - 项目介绍和快速开始

### 部署文档
- [deploy/README.md](../deploy/README.md) - Docker 部署指南
- [deploy/config.example.yaml](../deploy/config.example.yaml) - 配置文件示例

### 安全与防护
- [plans/caddy-coraza-crs-full-cutover-plan.md](./plans/caddy-coraza-crs-full-cutover-plan.md) - Caddy + Coraza WAF + OWASP CRS 一次性全面引入执行计划（阻断模式）
- [plans/waf-update-management-design.md](./plans/waf-update-management-design.md) - WAF 规则更新管理设计（源地址下载 + 手动上传，不依赖 compose 额外挂载）
- [plans/waf-update-p0-implementation-plan.md](./plans/waf-update-p0-implementation-plan.md) - WAF 更新管理 P0 开发实施细化计划（文件级）
- [plans/waf-update-task-checklist.md](./plans/waf-update-task-checklist.md) - WAF 更新管理实施任务清单（P0/P1/P2 + 里程碑）
- [plans/waf-update-operations-guide.md](./plans/waf-update-operations-guide.md) - WAF 更新管理操作指南（API 示例 + 运维 SOP）

## 功能设计文档

### 通知功能 (Notification Feature)
1. **[notification-feature-design.md](./notification-feature-design.md)** ⭐️ 完整设计文档
   - 13 个章节,涵盖设计的各个方面
   - 功能需求、架构设计、数据模型
   - API 设计、配置示例、安全考虑
   - 实施计划、技术选型、扩展性

2. **[notification-task-checklist.md](./notification-task-checklist.md)** ⭐️ 实施任务清单
   - 60 个具体任务,6 个阶段
   - 优先级标注 (P0/P1/P2)
   - 时间估算 (9 周)
   - 依赖关系、资源分配

3. **[notification-quick-reference.md](./notification-quick-reference.md)** ⭐️ 快速参考
   - 阶段规划、核心组件
   - 优先级任务、关键文件
   - 配置示例、API 端点
   - 开发流程、里程碑检查点

## 文档使用指南

### 面向产品经理/项目经理
建议阅读顺序:
1. `notification-quick-reference.md` (了解整体规划)
2. `notification-feature-design.md` 第 1-2 章 (功能需求)
3. `notification-task-checklist.md` (实施计划和里程碑)

### 面向架构师
建议阅读顺序:
1. `notification-feature-design.md` 第 3-5 章 (架构设计、数据模型、API 设计)
2. `notification-feature-design.md` 第 7-12 章 (技术选型、事件集成、安全、扩展性)
3. `notification-quick-reference.md` 第 2 节 (核心组件)

### 面向后端开发者
建议阅读顺序:
1. `notification-quick-reference.md` (快速了解)
2. `notification-feature-design.md` 第 3-8 章 (架构、数据模型、事件集成点)
3. `notification-task-checklist.md` (具体任务列表)
4. 开始编码,参考 `notification-quick-reference.md` 第 8 节 (开发流程)

### 面向前端开发者
建议阅读顺序:
1. `notification-feature-design.md` 第 5 章 (API 设计)
2. `notification-task-checklist.md` 阶段 5 (前端任务)
3. `notification-quick-reference.md` 第 7 节 (API 端点)

### 面向测试工程师
建议阅读顺序:
1. `notification-feature-design.md` 第 1-2 章 (功能需求)
2. `notification-task-checklist.md` 阶段 6 (测试任务)
3. `notification-quick-reference.md` 第 11 节 (测试清单)

## 文档维护

### 更新记录
- 2026-01-28: 创建通知功能完整文档体系
- 2026-02-10: 新增 Caddy + Coraza WAF + OWASP CRS 一次性全面引入执行计划
- 2026-02-10: 新增 WAF 更新管理设计（自动下载 + 手动上传）
- 2026-02-10: 新增 WAF 更新任务清单与运维操作指南
- 2026-02-10: 新增 WAF 更新管理 P0 实施细化计划

### 贡献指南
如需更新文档,请确保:
1. 保持三个文档的一致性
2. 更新相关的交叉引用
3. 在 README.md 中添加更新记录

### 文档结构说明

```
docs/
├── README.md                              # 文档中心 (本文件)
├── notification-feature-design.md         # 完整设计 (13章,~200行)
├── notification-task-checklist.md         # 任务清单 (60+任务,~400行)
└── notification-quick-reference.md        # 快速参考 (~200行)
```

### 文档特点

**notification-feature-design.md** (完整性)
- ✅ 全面的功能需求分析
- ✅ 详细的架构设计
- ✅ 完整的数据模型定义
- ✅ API 设计规范
- ✅ 配置示例
- ✅ 安全和扩展性考虑
- 📖 适合深度阅读和作为设计参考

**notification-task-checklist.md** (可执行性)
- ✅ 具体的任务分解
- ✅ 明确的优先级
- ✅ 时间估算
- ✅ 依赖关系
- ✅ 里程碑定义
- 📋 适合项目管理和任务跟踪

**notification-quick-reference.md** (实用性)
- ✅ 快速上手指南
- ✅ 关键信息汇总
- ✅ 代码示例
- ✅ 常见问题
- ✅ 检查清单
- 🚀 适合日常开发参考

## 相关资源

### 外部参考
- [Prometheus Alertmanager](https://prometheus.io/docs/alerting/latest/alertmanager/)
- [Grafana Alerting](https://grafana.com/docs/grafana/latest/alerting/)
- [PagerDuty Event API](https://developer.pagerduty.com/docs/events-api-v2/overview/)

### 技术文档
- [Go-Zero 框架](https://go-zero.dev/)
- [GORM 文档](https://gorm.io/)
- [gomail 文档](https://pkg.go.dev/gopkg.in/gomail.v2)
- [telegram-bot-api 文档](https://pkg.go.dev/github.com/go-telegram-bot-api/telegram-bot-api/v5)
- [expr 语言](https://expr-lang.org/)

## 快速链接

### 立即开始
想要开始实施通知功能? 请按以下步骤:

1. **阅读快速参考** (5 分钟)
   - [notification-quick-reference.md](./notification-quick-reference.md)

2. **查看任务清单** (10 分钟)
   - [notification-task-checklist.md](./notification-task-checklist.md)
   - 找到 "阶段 1" 的任务

3. **开始编码** (参考开发流程)
   - 创建数据库表
   - 定义核心接口
   - 实现 NotificationManager

### 寻求帮助
遇到问题? 请查看:
- `notification-quick-reference.md` 第 12 节 - 常见问题
- `notification-feature-design.md` 第 8 节 - 事件集成点

### 贡献代码
准备贡献? 请确保:
- 遵循 `notification-feature-design.md` 中的架构设计
- 完成 `notification-task-checklist.md` 中的对应任务
- 添加单元测试 (覆盖率 > 80%)
- 更新相关文档

---

**文档版本**: 1.1.0
**最后更新**: 2026-02-10
**维护者**: LogFlux Team
