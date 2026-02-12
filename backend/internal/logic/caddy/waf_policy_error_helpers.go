package caddy

import (
	"fmt"
	"regexp"
	"strings"
)

func localizeWafPolicyError(err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s", localizeWafPolicyMessage(err.Error()))
}

func localizeWafPolicyMessage(rawMessage string) string {
	messageText := strings.TrimSpace(rawMessage)
	if messageText == "" {
		return ""
	}

	exactMap := map[string]string{
		"policy id is required":                                   "策略 ID 不能为空",
		"policy not found":                                        "未找到策略",
		"invalid policy payload":                                  "策略请求参数不合法",
		"policy name is required":                                 "策略名称不能为空",
		"invalid policy transaction":                              "策略事务上下文无效",
		"invalid policy revision context":                         "策略版本上下文无效",
		"policy revision not found":                               "未找到策略发布版本",
		"invalid policy revision":                                 "策略发布版本无效",
		"revisionId is required":                                  "回滚版本 ID 不能为空",
		"policy is nil":                                           "策略对象不能为空",
		"db is nil":                                               "数据库连接不可用",
		"caddy server not found":                                  "未找到 Caddy 服务",
		"caddy config is empty":                                   "Caddy 配置为空",
		"policy directives is empty":                              "策略指令为空",
		"invalid policy exclusion payload":                        "策略例外请求参数不合法",
		"invalid policy binding payload":                          "策略绑定请求参数不合法",
		"policy exclusion id is required":                         "策略例外 ID 不能为空",
		"policy binding id is required":                           "策略绑定 ID 不能为空",
		"invalid policy false positive feedback payload":          "误报反馈请求参数不合法",
		"invalid policy false positive feedback update payload":   "误报反馈状态更新参数不合法",
		"policy false positive feedback id is required":           "误报反馈 ID 不能为空",
		"policy false positive feedback not found":                "未找到误报反馈记录",
		"invalid dueAt":                                           "截止时间格式不合法",
		"policy exclusion not found":                              "未找到策略例外",
		"policy binding not found":                                "未找到策略绑定",
		"remove value is required":                                "例外移除值不能为空",
		"feedback reason is required":                             "误报原因不能为空",
		"site scope requires host":                                "站点作用域必须填写 host",
		"route scope requires path":                               "路由作用域必须填写 path",
		"invalid policy binding context":                          "策略绑定上下文无效",
		"coraza directives block not found in caddy config":       "Caddy 配置中未找到 Coraza directives 配置块",
		"coraza directives start tick not found":                  "Coraza directives 起始标记缺失",
		"coraza directives end tick not found":                    "Coraza directives 结束标记缺失",
		"last good caddy config is empty":                         "缺少可回滚的 last_good 配置",
		"caddy config is empty, please save caddy config first":   "Caddy 配置为空，请先保存 Caddy 配置",
		"requestBodyLimit must be greater than 0":                 "请求体大小限制必须大于 0",
		"requestBodyNoFilesLimit must be greater than 0":          "无文件请求体大小限制必须大于 0",
		"requestBodyLimit is too large":                           "请求体大小限制过大",
		"requestBodyNoFilesLimit is too large":                    "无文件请求体大小限制过大",
		"crsParanoiaLevel must be between 1 and 4":                "CRS 防御等级（PL）必须在 1 到 4 之间",
		"crsInboundAnomalyThreshold must be between 1 and 20":     "CRS 入站异常阈值必须在 1 到 20 之间",
		"crsOutboundAnomalyThreshold must be between 1 and 20":    "CRS 出站异常阈值必须在 1 到 20 之间",
		"policy publish rollback to last_good succeeded":          "策略发布失败，已自动回滚到 last_good 配置",
		"policy rollback rollback to last_good succeeded":         "策略回滚失败，已自动回滚到 last_good 配置",
		"policy publish persist rollback to last_good succeeded":  "策略发布落库失败，已自动回滚到 last_good 配置",
		"policy rollback persist rollback to last_good succeeded": "策略回滚落库失败，已自动回滚到 last_good 配置",
	}
	if localized, ok := exactMap[messageText]; ok {
		return localized
	}

	replacementRules := []struct {
		pattern     *regexp.Regexp
		replacement string
	}{
		{regexp.MustCompile(`(?i)policy name already exists:`), "策略名称已存在："},
		{regexp.MustCompile(`(?i)check policy name failed:`), "校验策略名称失败："},
		{regexp.MustCompile(`(?i)create policy failed:`), "创建策略失败："},
		{regexp.MustCompile(`(?i)update policy failed:`), "更新策略失败："},
		{regexp.MustCompile(`(?i)delete policy failed:`), "删除策略失败："},
		{regexp.MustCompile(`(?i)delete policy revisions failed:`), "删除策略版本失败："},
		{regexp.MustCompile(`(?i)count policies failed:`), "查询策略总数失败："},
		{regexp.MustCompile(`(?i)query policies failed:`), "查询策略列表失败："},
		{regexp.MustCompile(`(?i)count policy revisions failed:`), "查询策略发布记录总数失败："},
		{regexp.MustCompile(`(?i)query policy revisions failed:`), "查询策略发布记录失败："},
		{regexp.MustCompile(`(?i)query policy names failed:`), "查询策略名称失败："},
		{regexp.MustCompile(`(?i)query previous policy revision failed:`), "查询上一条策略发布记录失败："},
		{regexp.MustCompile(`(?i)query policy stats policies failed:`), "查询策略统计对象失败："},
		{regexp.MustCompile(`(?i)query policy stats bindings failed:`), "查询策略统计绑定失败："},
		{regexp.MustCompile(`(?i)count policy stats hits failed:`), "统计策略命中数失败："},
		{regexp.MustCompile(`(?i)count policy stats blocked hits failed:`), "统计策略拦截数失败："},
		{regexp.MustCompile(`(?i)count policy stats suspected false positives failed:`), "统计疑似误报失败："},
		{regexp.MustCompile(`(?i)count policy stats range hits failed:`), "统计范围命中数失败："},
		{regexp.MustCompile(`(?i)count policy stats range blocked hits failed:`), "统计范围拦截数失败："},
		{regexp.MustCompile(`(?i)query policy stats trend failed:`), "查询策略趋势失败："},
		{regexp.MustCompile(`(?i)query policy stats dimensions failed:`), "查询策略维度统计失败："},
		{regexp.MustCompile(`(?i)count policy false positive feedbacks failed:`), "查询误报反馈总数失败："},
		{regexp.MustCompile(`(?i)query policy false positive feedback failed:`), "查询误报反馈详情失败："},
		{regexp.MustCompile(`(?i)query policy false positive feedbacks failed:`), "查询误报反馈列表失败："},
		{regexp.MustCompile(`(?i)create policy false positive feedback failed:`), "创建误报反馈失败："},
		{regexp.MustCompile(`(?i)update policy false positive feedback status failed:`), "更新误报反馈状态失败："},
		{regexp.MustCompile(`(?i)invalid policy false positive feedback status:`), "误报反馈状态不合法："},
		{regexp.MustCompile(`(?i)invalid policy false positive feedback sla status:`), "误报反馈 SLA 状态不合法："},
		{regexp.MustCompile(`(?i)invalid dueat:`), "截止时间格式不合法："},
		{regexp.MustCompile(`(?i)invalid policy config json:`), "策略配置 JSON 格式不合法："},
		{regexp.MustCompile(`(?i)invalid engine mode:`), "引擎模式不合法："},
		{regexp.MustCompile(`(?i)invalid audit engine:`), "审计模式不合法："},
		{regexp.MustCompile(`(?i)invalid audit log format:`), "审计日志格式不合法："},
		{regexp.MustCompile(`(?i)invalid crs template:`), "CRS 调优模板不合法："},
		{regexp.MustCompile(`(?i)invalid policy scope type:`), "策略作用域不合法："},
		{regexp.MustCompile(`(?i)invalid policy remove type:`), "策略例外类型不合法："},
		{regexp.MustCompile(`(?i)invalid policy method:`), "策略匹配方法不合法："},
		{regexp.MustCompile(`(?i)binding priority must be between`), "策略绑定优先级不合法："},
		{regexp.MustCompile(`(?i)query caddy server failed:`), "查询 Caddy 服务失败："},
		{regexp.MustCompile(`(?i)query policy failed:`), "查询策略失败："},
		{regexp.MustCompile(`(?i)count policy exclusions failed:`), "查询策略例外总数失败："},
		{regexp.MustCompile(`(?i)query policy exclusions failed:`), "查询策略例外失败："},
		{regexp.MustCompile(`(?i)create policy exclusion failed:`), "创建策略例外失败："},
		{regexp.MustCompile(`(?i)update policy exclusion failed:`), "更新策略例外失败："},
		{regexp.MustCompile(`(?i)delete policy exclusion failed:`), "删除策略例外失败："},
		{regexp.MustCompile(`(?i)count policy bindings failed:`), "查询策略绑定总数失败："},
		{regexp.MustCompile(`(?i)query policy bindings failed:`), "查询策略绑定失败："},
		{regexp.MustCompile(`(?i)create policy binding failed:`), "创建策略绑定失败："},
		{regexp.MustCompile(`(?i)update policy binding failed:`), "更新策略绑定失败："},
		{regexp.MustCompile(`(?i)delete policy binding failed:`), "删除策略绑定失败："},
		{regexp.MustCompile(`(?i)query policy binding conflict failed:`), "查询策略绑定冲突失败："},
		{regexp.MustCompile(`(?i)policy binding conflict detected`), "检测到策略绑定冲突："},
		{regexp.MustCompile(`(?i)query policy binding conflicts failed:`), "查询策略绑定冲突失败："},
		{regexp.MustCompile(`(?i)policy binding conflicts found:`), "存在策略绑定冲突："},
		{regexp.MustCompile(`(?i)save caddy server config failed:`), "保存 Caddy 配置失败："},
		{regexp.MustCompile(`(?i)create caddy config history failed:`), "写入 Caddy 配置历史失败："},
		{regexp.MustCompile(`(?i)query latest policy revision failed:`), "查询最新策略发布版本失败："},
		{regexp.MustCompile(`(?i)create policy revision failed:`), "创建策略发布版本失败："},
		{regexp.MustCompile(`(?i)mark policy revisions rolled_back failed:`), "更新策略发布版本状态失败："},
		{regexp.MustCompile(`(?i)update revision status failed:`), "更新回滚目标版本状态失败："},
		{regexp.MustCompile(`(?i)policy validate failed:`), "策略校验失败："},
		{regexp.MustCompile(`(?i)policy publish validate failed:`), "策略发布前校验失败："},
		{regexp.MustCompile(`(?i)policy publish load failed:`), "策略发布失败："},
		{regexp.MustCompile(`(?i)policy publish persist failed:`), "策略发布落库失败："},
		{regexp.MustCompile(`(?i)policy rollback validate failed:`), "策略回滚前校验失败："},
		{regexp.MustCompile(`(?i)policy rollback load failed:`), "策略回滚失败："},
		{regexp.MustCompile(`(?i)policy rollback persist failed:`), "策略回滚落库失败："},
		{regexp.MustCompile(`(?i)rollback to last_good failed:`), "回滚到 last_good 失败："},
		{regexp.MustCompile(`(?i)rollback last_good adapt failed:`), "last_good 适配失败："},
		{regexp.MustCompile(`(?i)rollback last_good load failed:`), "last_good 加载失败："},
		{regexp.MustCompile(`(?i)invalid starttime:`), "开始时间格式不合法："},
		{regexp.MustCompile(`(?i)invalid endtime:`), "结束时间格式不合法："},
		{regexp.MustCompile(`(?i)adapt failed:`), "Caddy 适配失败："},
		{regexp.MustCompile(`(?i)load failed:`), "Caddy 加载失败："},
		{regexp.MustCompile(`(?i)caddy api error:`), "Caddy API 错误："},
		{regexp.MustCompile(`(?i)connect: connection refused`), "连接被拒绝"},
		{regexp.MustCompile(`(?i)context deadline exceeded`), "请求超时"},
		{regexp.MustCompile(`(?i)i/o timeout`), "网络超时"},
	}

	localized := messageText
	for _, rule := range replacementRules {
		localized = rule.pattern.ReplaceAllString(localized, rule.replacement)
	}
	return strings.TrimSpace(localized)
}
