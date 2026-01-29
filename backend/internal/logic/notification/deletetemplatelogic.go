package notification

import (
	"context"

	"logflux/internal/svc"
	"logflux/internal/types"
	"logflux/model"

	"github.com/zeromicro/go-zero/core/logx"
)

type DeleteTemplateLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewDeleteTemplateLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeleteTemplateLogic {
	return &DeleteTemplateLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *DeleteTemplateLogic) DeleteTemplate(req *types.IDReq) (resp *types.BaseResp, err error) {
	if err := l.svcCtx.DB.Delete(&model.NotificationTemplate{}, req.ID).Error; err != nil {
		return nil, err
	}

	// Reload templates
	if l.svcCtx.NotificationMgr != nil {
		l.svcCtx.NotificationMgr.ReloadTemplates()
	}

	return &types.BaseResp{
		Code: 200,
		Msg:  "success",
	}, nil
}
