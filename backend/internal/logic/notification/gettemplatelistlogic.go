package notification

import (
	"context"

	"logflux/internal/svc"
	"logflux/internal/types"
	"logflux/model"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetTemplateListLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetTemplateListLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetTemplateListLogic {
	return &GetTemplateListLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetTemplateListLogic) GetTemplateList() (resp *types.TemplateListResp, err error) {
	var templates []model.NotificationTemplate
	if err := l.svcCtx.DB.Find(&templates).Error; err != nil {
		return nil, err
	}

	list := make([]types.TemplateItem, 0, len(templates))
	for _, t := range templates {
		list = append(list, types.TemplateItem{
			ID:        uint(t.ID),
			Name:      t.Name,
			Format:    t.Format,
			Content:   t.Content,
			Type:      t.Type,
			CreatedAt: t.CreatedAt.Format("2006-01-02 15:04:05"),
			UpdatedAt: t.UpdatedAt.Format("2006-01-02 15:04:05"),
		})
	}

	return &types.TemplateListResp{
		List: list,
	}, nil
}
