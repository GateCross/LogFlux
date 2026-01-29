package notification

import (
	"context"

	"encoding/json"
	"logflux/internal/svc"
	"logflux/internal/types"
	"logflux/model"

	"github.com/zeromicro/go-zero/core/logx"
)

type CreateRuleLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewCreateRuleLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CreateRuleLogic {
	return &CreateRuleLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *CreateRuleLogic) CreateRule(req *types.RuleReq) (resp *types.BaseResp, err error) {
	var conditionMap map[string]interface{}
	if req.Condition != "" {
		if err := json.Unmarshal([]byte(req.Condition), &conditionMap); err != nil {
			return nil, err
		}
	}

	rule := &model.NotificationRule{
		Name:            req.Name,
		Enabled:         req.Enabled,
		RuleType:        req.RuleType,
		EventType:       req.EventType,
		Condition:       model.JSONMap(conditionMap),
		ChannelIDs:      model.Int64Array(req.ChannelIDs),
		Template:        req.Template,
		SilenceDuration: req.SilenceDuration,
		Description:     req.Description,
	}

	if err := l.svcCtx.DB.Create(rule).Error; err != nil {
		return nil, err
	}

	// Reload rules
	if l.svcCtx.NotificationMgr != nil {
		if err := l.svcCtx.NotificationMgr.ReloadRules(); err != nil {
			l.Logger.Errorf("Failed to reload rules: %v", err)
		}
	}

	return &types.BaseResp{
		Code: 200,
		Msg:  "success",
	}, nil
}
