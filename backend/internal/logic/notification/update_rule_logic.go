package notification

import (
	"context"
	"encoding/json"
	commonmodel "logflux/model/common"
	notificationmodel "logflux/model/notification"

	"logflux/internal/svc"
	"logflux/internal/types"

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
	var rule notificationmodel.NotificationRule
	if err := l.svcCtx.DB.WithContext(l.ctx).First(&rule, req.ID).Error; err != nil {
		return nil, err
	}

	if req.Name != "" {
		rule.Name = req.Name
	}
	rule.Enabled = req.Enabled
	if req.RuleType != "" {
		rule.RuleType = req.RuleType
	}
	if req.EventLevel != "" {
		rule.EventLevel = req.EventLevel
	}
	if req.EventTypes != nil {
		rule.EventTypes = notificationmodel.StringArray(req.EventTypes)
	}
	if req.Condition != "" {
		var conditionMap map[string]interface{}
		if err := json.Unmarshal([]byte(req.Condition), &conditionMap); err != nil {
			return nil, err
		}
		rule.Condition = commonmodel.JSONMap(conditionMap)
	}
	if req.ChannelIDs != nil {
		rule.ChannelIDs = notificationmodel.Int64Array(req.ChannelIDs)
	}
	if req.Template != "" {
		rule.Template = req.Template
	}
	rule.SilenceDuration = req.SilenceDuration
	if req.Description != "" {
		rule.Description = req.Description
	}

	if err := l.svcCtx.DB.WithContext(l.ctx).Save(&rule).Error; err != nil {
		return nil, err
	}

	if l.svcCtx.NotificationMgr != nil {
		l.svcCtx.NotificationMgr.ReloadRules()
	}

	return &types.BaseResp{
		Code: 200,
		Msg:  "成功",
	}, nil
}
