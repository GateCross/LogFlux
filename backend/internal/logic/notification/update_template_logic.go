package notification

import (
	"context"

	"logflux/internal/svc"
	"logflux/internal/types"
	"logflux/model"

	"github.com/zeromicro/go-zero/core/logx"
)

type UpdateTemplateLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewUpdateTemplateLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateTemplateLogic {
	return &UpdateTemplateLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *UpdateTemplateLogic) UpdateTemplate(req *types.TemplateUpdateReq) (resp *types.BaseResp, err error) {
	var template model.NotificationTemplate
	if err := l.svcCtx.DB.First(&template, req.ID).Error; err != nil {
		return nil, err
	}

	if req.Name != "" {
		template.Name = req.Name
	}
	if req.Format != "" {
		template.Format = req.Format
	}
	if req.Content != "" {
		template.Content = req.Content
	}
	if req.Type != "" {
		template.Type = req.Type
	}

	if err := l.svcCtx.DB.Save(&template).Error; err != nil {
		return nil, err
	}

	return &types.BaseResp{
		Code: 200,
		Msg:  "success",
	}, nil
}
