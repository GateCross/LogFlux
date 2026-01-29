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

	updates := make(map[string]interface{})
	if req.Name != "" {
		updates["name"] = req.Name
	}
	if req.Format != "" {
		updates["format"] = req.Format
	}
	if req.Content != "" {
		updates["content"] = req.Content
	}
	if req.Type != "" {
		updates["type"] = req.Type
	}

	if err := l.svcCtx.DB.Model(&template).Updates(updates).Error; err != nil {
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
