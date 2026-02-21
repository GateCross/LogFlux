package caddy

import (
	"context"
	"strings"

	"logflux/internal/notification"
	"logflux/model"

	"github.com/zeromicro/go-zero/core/logx"
)

func (helper *wafLogicHelper) notifyWafUpdateJobEvent(job *model.WafUpdateJob, status, message string, releaseID uint) {
	if helper == nil || helper.svcCtx == nil || helper.svcCtx.NotificationMgr == nil || job == nil {
		return
	}

	normalizedStatus := strings.ToLower(strings.TrimSpace(status))
	normalizedAction := strings.ToLower(strings.TrimSpace(job.Action))
	eventType, eventLevel, eventTitle := resolveWafUpdateEventMeta(normalizedAction, normalizedStatus)
	if eventType == "" {
		return
	}

	effectiveReleaseID := releaseID
	if effectiveReleaseID == 0 {
		effectiveReleaseID = job.ReleaseID
	}

	payload := map[string]interface{}{
		"jobId":       job.ID,
		"action":      normalizedAction,
		"status":      normalizedStatus,
		"triggerMode": strings.ToLower(strings.TrimSpace(job.TriggerMode)),
		"operator":    strings.TrimSpace(job.Operator),
		"sourceId":    job.SourceID,
		"releaseId":   effectiveReleaseID,
		"message":     strings.TrimSpace(message),
	}

	if sourceName := helper.queryWafSourceName(job.SourceID); sourceName != "" {
		payload["sourceName"] = sourceName
	}
	if releaseVersion := helper.queryWafReleaseVersion(effectiveReleaseID); releaseVersion != "" {
		payload["releaseVersion"] = releaseVersion
	}

	notifyWafEventAsync(helper.svcCtx.NotificationMgr, helper.logger, eventType, eventLevel, eventTitle, buildWafUpdateEventMessage(normalizedAction, message), payload)
}

func resolveWafUpdateEventMeta(action, status string) (eventType, level, title string) {
	switch action {
	case "check":
		if status == wafJobStatusSuccess {
			return notification.EventSecurityWafSourceCheckSuccess, notification.LevelInfo, "WAF 更新源检查成功"
		}
		if status == wafJobStatusFailed {
			return notification.EventSecurityWafSourceCheckFailed, notification.LevelError, "WAF 更新源检查失败"
		}
	case "download":
		if status == wafJobStatusSuccess {
			return notification.EventSecurityWafSourceSyncSuccess, notification.LevelInfo, "WAF 更新源同步成功"
		}
		if status == wafJobStatusFailed {
			return notification.EventSecurityWafSourceSyncFailed, notification.LevelError, "WAF 更新源同步失败"
		}
	case "activate":
		if status == wafJobStatusSuccess {
			return notification.EventSecurityWafReleaseActivateSuccess, notification.LevelInfo, "WAF 版本激活成功"
		}
		if status == wafJobStatusFailed {
			return notification.EventSecurityWafReleaseActivateFailed, notification.LevelError, "WAF 版本激活失败"
		}
	case "rollback":
		if status == wafJobStatusSuccess {
			return notification.EventSecurityWafReleaseRollbackSuccess, notification.LevelWarning, "WAF 版本回滚成功"
		}
		if status == wafJobStatusFailed {
			return notification.EventSecurityWafReleaseRollbackFailed, notification.LevelError, "WAF 版本回滚失败"
		}
	}
	return "", "", ""
}

func buildWafUpdateEventMessage(action, message string) string {
	actionName := map[string]string{
		"check":    "检查",
		"download": "同步",
		"activate": "激活",
		"rollback": "回滚",
	}[action]
	if actionName == "" {
		actionName = "任务"
	}

	trimmedMessage := strings.TrimSpace(message)
	if trimmedMessage == "" {
		return "WAF " + actionName + "任务已完成"
	}
	return "WAF " + actionName + "结果：" + trimmedMessage
}

func (helper *wafLogicHelper) queryWafSourceName(sourceID uint) string {
	if helper == nil || helper.svcCtx == nil || helper.svcCtx.DB == nil || sourceID == 0 {
		return ""
	}

	var source model.WafSource
	if err := helper.svcCtx.DB.Select("name").First(&source, sourceID).Error; err != nil {
		return ""
	}
	return strings.TrimSpace(source.Name)
}

func (helper *wafLogicHelper) queryWafReleaseVersion(releaseID uint) string {
	if helper == nil || helper.svcCtx == nil || helper.svcCtx.DB == nil || releaseID == 0 {
		return ""
	}

	var release model.WafRelease
	if err := helper.svcCtx.DB.Select("version").First(&release, releaseID).Error; err != nil {
		return ""
	}
	return strings.TrimSpace(release.Version)
}

func notifyWafEventAsync(
	manager notification.NotificationManager,
	logger logx.Logger,
	eventType string,
	level string,
	title string,
	message string,
	data map[string]interface{},
) {
	if manager == nil {
		return
	}

	event := notification.NewEvent(eventType, level, strings.TrimSpace(title), strings.TrimSpace(message))
	if len(data) > 0 {
		event.WithDataMap(data)
	}

	go func() {
		if err := manager.Notify(context.Background(), event); err != nil {
			logger.Errorf("notify waf event failed, type=%s err=%v", eventType, err)
		}
	}()
}
