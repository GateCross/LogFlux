package notification

import (
	"context"

	"logflux/internal/svc"
	"logflux/internal/types"
	"logflux/model"

	"github.com/zeromicro/go-zero/core/logx"
)

type DeleteRuleLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewDeleteRuleLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeleteRuleLogic {
	return &DeleteRuleLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *DeleteRuleLogic) DeleteRule(req *types.IDReq) (resp *types.BaseResp, err error) {
	if err := l.svcCtx.DB.Delete(&model.NotificationRule{}, req.ID).Error; err != nil {
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
