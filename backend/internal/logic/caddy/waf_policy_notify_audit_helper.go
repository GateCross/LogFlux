package caddy

import (
	"fmt"

	"logflux/internal/notification"
	"logflux/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type WafPolicyNotifyAuditHelper struct {
	svcCtx *svc.ServiceContext
	logger logx.Logger
}

func NewWafPolicyNotifyAuditHelper(svcCtx *svc.ServiceContext, logger logx.Logger) *WafPolicyNotifyAuditHelper {
	return &WafPolicyNotifyAuditHelper{svcCtx: svcCtx, logger: logger}
}

func (h *WafPolicyNotifyAuditHelper) NotifyFailure(eventType, title string, policyID uint, policyName, operator string, err error) error {
	if err == nil {
		return nil
	}
	notifyData := buildWafPolicyNotifyData(policyID, policyName, operator)
	notifyData["error"] = localizeWafPolicyMessage(err.Error())
	notifyWafPolicyEventAsync(
		h.svcCtx,
		h.logger,
		eventType,
		notification.LevelError,
		title,
		fmt.Sprintf("%s：%s", title, localizeWafPolicyMessage(err.Error())),
		notifyData,
	)
	return localizeWafPolicyError(err)
}

func (h *WafPolicyNotifyAuditHelper) NotifyAutoRollback(message string, policyID uint, policyName, operator string) {
	notifyWafPolicyEventAsync(
		h.svcCtx,
		h.logger,
		notification.EventSecurityWafPolicyAutoRollback,
		notification.LevelWarning,
		"WAF 策略自动回滚",
		message,
		buildWafPolicyNotifyData(policyID, policyName, operator),
	)
}

func (h *WafPolicyNotifyAuditHelper) NotifySuccess(eventType, title, message string, policyID uint, policyName, operator string) {
	notifyWafPolicyEventAsync(
		h.svcCtx,
		h.logger,
		eventType,
		notification.LevelInfo,
		title,
		message,
		buildWafPolicyNotifyData(policyID, policyName, operator),
	)
}
