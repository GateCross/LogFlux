package notification

import (
	"context"

	"encoding/json"
	"logflux/internal/svc"
	"logflux/internal/types"
	"logflux/model"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetRuleListLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetRuleListLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetRuleListLogic {
	return &GetRuleListLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetRuleListLogic) GetRuleList() (resp *types.RuleListResp, err error) {
	var rules []model.NotificationRule
	if err := l.svcCtx.DB.Find(&rules).Error; err != nil {
		return nil, err
	}

	list := make([]types.RuleItem, 0, len(rules))
	for _, r := range rules {
		conditionBytes, _ := json.Marshal(r.Condition)

		list = append(list, types.RuleItem{
			ID:              r.ID,
			Name:            r.Name,
			Enabled:         r.Enabled,
			RuleType:        r.RuleType,
			EventType:       r.EventType,
			Condition:       string(conditionBytes),
			ChannelIDs:      []int64(r.ChannelIDs),
			Template:        r.Template,
			SilenceDuration: r.SilenceDuration,
			Description:     r.Description,
			CreatedAt:       r.CreatedAt.Format("2006-01-02 15:04:05"),
			UpdatedAt:       r.UpdatedAt.Format("2006-01-02 15:04:05"),
		})
	}

	return &types.RuleListResp{
		List: list,
	}, nil
}
