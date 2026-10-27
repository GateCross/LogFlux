package caddy

import (
	"context"
	caddymodel "logflux/model/caddy"

	"logflux/internal/svc"
	"logflux/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetCaddyConfigHistoryLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetCaddyConfigHistoryLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetCaddyConfigHistoryLogic {
	return &GetCaddyConfigHistoryLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// GetCaddyConfigHistory 返回唯一的配置历史；server_id 列保留但恒为 0，不再参与过滤。
func (l *GetCaddyConfigHistoryLogic) GetCaddyConfigHistory(req *types.CaddyConfigHistoryListReq) (resp *types.CaddyConfigHistoryListResp, err error) {
	var history []caddymodel.CaddyConfigHistory
	var total int64

	db := l.svcCtx.DB.WithContext(l.ctx).Model(&caddymodel.CaddyConfigHistory{})

	if err := db.Count(&total).Error; err != nil {
		return nil, err
	}

	offset := (req.Page - 1) * req.PageSize
	if err := db.Order("id desc").Limit(req.PageSize).Offset(offset).Find(&history).Error; err != nil {
		return nil, err
	}

	list := make([]types.CaddyConfigHistoryItem, 0, len(history))
	for _, item := range history {
		list = append(list, types.CaddyConfigHistoryItem{
			ID:        item.ID,
			ServerId:  item.ServerID,
			Action:    item.Action,
			Hash:      item.Hash,
			CreatedAt: item.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}

	return &types.CaddyConfigHistoryListResp{
		List:  list,
		Total: total,
	}, nil
}
