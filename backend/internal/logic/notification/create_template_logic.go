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
	template := &model.NotificationTemplate{
		Name:    req.Name,
		Format:  req.Format,
		Content: req.Content,
		Type:    req.Type,
	}

	if err := l.svcCtx.DB.Create(template).Error; err != nil {
		return nil, err
	}

	return &types.BaseResp{
		Code: 200,
		Msg:  "success",
	}, nil
}
