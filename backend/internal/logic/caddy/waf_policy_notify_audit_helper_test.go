package caddy

import (
	"context"
	"errors"
	"testing"
	"time"

	"logflux/internal/notification"
	"logflux/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type notifyRecorder struct {
	events chan *notification.Event
}

func (r *notifyRecorder) Notify(_ context.Context, event *notification.Event) error {
	r.events <- event
	return nil
}
func (r *notifyRecorder) RegisterProvider(notification.NotificationProvider) error { return nil }
func (r *notifyRecorder) Start(context.Context) error                              { return nil }
func (r *notifyRecorder) Stop() error                                              { return nil }
func (r *notifyRecorder) ReloadChannels() error                                    { return nil }
func (r *notifyRecorder) ReloadRules() error                                       { return nil }
func (r *notifyRecorder) ReloadTemplates() error                                   { return nil }
func (r *notifyRecorder) SendToChannel(context.Context, uint, *notification.Event) error {
	return nil
}

func TestWafPolicyNotifyAuditHelperNotifyFailure(t *testing.T) {
	recorder := &notifyRecorder{events: make(chan *notification.Event, 1)}
	helper := NewWafPolicyNotifyAuditHelper(&svc.ServiceContext{NotificationMgr: recorder}, logx.WithContext(context.Background()))

	err := helper.NotifyFailure(notification.EventSecurityWafPolicyPublishFailed, "WAF 策略发布失败", 12, "baseline", "tester", errors.New("policy publish validate failed"))
	if err == nil {
		t.Fatalf("expected localized error")
	}
	select {
	case event := <-recorder.events:
		if event.Type != notification.EventSecurityWafPolicyPublishFailed {
			t.Fatalf("unexpected event type: %s", event.Type)
		}
		if event.Data["policyId"] != uint(12) {
			t.Fatalf("unexpected policyId: %#v", event.Data["policyId"])
		}
	case <-time.After(time.Second):
		t.Fatal("expected notify event")
	}
}
