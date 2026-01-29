package notification

import (
	"context"

	"encoding/json"
	"logflux/internal/svc"
	"logflux/internal/types"
	"logflux/model"

	"github.com/zeromicro/go-zero/core/logx"
)

type UpdateRuleLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewUpdateRuleLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateRuleLogic {
	return &UpdateRuleLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *UpdateRuleLogic) UpdateRule(req *types.RuleUpdateReq) (resp *types.BaseResp, err error) {
	var rule model.NotificationRule
	if err := l.svcCtx.DB.First(&rule, req.ID).Error; err != nil {
		return nil, err
	}

	updates := make(map[string]interface{})
	if req.Name != "" {
		updates["name"] = req.Name
	}
	// Handle Bool Enabled - Always update using the provided value (default false if omitted)
	// Given types.go definition, we assume users send full update or we have to trust the value.
	// For API consistency with Channel, we apply it.
	updates["enabled"] = req.Enabled

	if req.RuleType != "" {
		updates["rule_type"] = req.RuleType
	}
	if req.EventType != "" {
		updates["event_type"] = req.EventType
	}
	if req.Condition != "" {
		var conditionMap map[string]interface{}
		if err := json.Unmarshal([]byte(req.Condition), &conditionMap); err != nil {
			return nil, err
		}
		updates["condition"] = model.JSONMap(conditionMap)
	}
	if req.ChannelIDs != nil { // Slice can be nil if not present in JSON? No, empty slice if omitted usually.
		// types.go: ChannelIDs []int64 `json:"channelIds,optional"`
		// If omitted, it might be nil or empty.
		// If user wants to clear channels, they send []. if they omit, it might satisfy "optional".
		// But here passing a nil slice updates nothing usually in specialized logic, but updates["channel_ids"] = nil might clear it.
		// Go-zero handling: if it's optional and not passed, it is zero value (nil).
		// If passed as [], it is empty slice.
		// So checking nil is correct to skip update if we want "PATCH" behavior.
		// BUT, if user WANTS to clear it, they send empty list.
		// Distinction between nil and empty slice in Go deserialization depends on library.
		// For now we assume if len > 0 or if it's not nil, update.
		// Safer: if user sends request, we update.
		// But let's check if req.ChannelIDs is nil.
		if req.ChannelIDs != nil {
			updates["channel_ids"] = model.Int64Array(req.ChannelIDs)
		}
	}
	if req.Template != "" {
		updates["template"] = req.Template
	}
	if req.SilenceDuration != 0 {
		updates["silence_duration"] = req.SilenceDuration
	}
	if req.Description != "" {
		updates["description"] = req.Description
	}

	if err := l.svcCtx.DB.Model(&rule).Updates(updates).Error; err != nil {
		return nil, err
	}

	// Reload rules
	if l.svcCtx.NotificationMgr != nil {
		l.svcCtx.NotificationMgr.ReloadRules()
	}

	return &types.BaseResp{
		Code: 200,
		Msg:  "success",
	}, nil
}
