package caddy

import (
	"context"
	"strings"

	"logflux/internal/notification"
	"logflux/internal/svc"
	"logflux/internal/utils/safego"

	"github.com/zeromicro/go-zero/core/logx"
)

func notifyWafPolicyEventAsync(
	svcCtx *svc.ServiceContext,
	logger logx.Logger,
	eventType string,
	level string,
	title string,
	message string,
	data map[string]interface{},
) {
	if svcCtx == nil || svcCtx.NotificationMgr == nil {
		return
	}

	event := notification.NewEvent(eventType, level, strings.TrimSpace(title), strings.TrimSpace(message))
	if len(data) > 0 {
		event.WithDataMap(data)
	}

	safego.New(context.Background(), "WAF 策略通知").Go(func() {
		if err := svcCtx.NotificationMgr.Notify(context.Background(), event); err != nil {
			logger.Errorf("发送 WAF 策略通知失败: type=%s err=%v", eventType, err)
		}
	})
}

func buildWafPolicyNotifyData(policyID uint, policyName, operator string) map[string]interface{} {
	payload := map[string]interface{}{
		"policyId": policyID,
	}
	if name := strings.TrimSpace(policyName); name != "" {
		payload["policyName"] = name
	}
	if user := strings.TrimSpace(operator); user != "" {
		payload["operator"] = user
	}
	return payload
}
