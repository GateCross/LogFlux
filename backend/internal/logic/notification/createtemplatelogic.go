package notification

import (
	"context"

	"logflux/internal/svc"
	"logflux/internal/types"
	"logflux/model"

	"github.com/zeromicro/go-zero/core/logx"
)

type CreateTemplateLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewCreateTemplateLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CreateTemplateLogic {
	return &CreateTemplateLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *CreateTemplateLogic) CreateTemplate(req *types.TemplateReq) (resp *types.BaseResp, err error) {
	t := &model.NotificationTemplate{
		Name:    req.Name,
		Format:  req.Format,
		Content: req.Content,
		Type:    req.Type,
	}

	if err := l.svcCtx.DB.Create(t).Error; err != nil {
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
