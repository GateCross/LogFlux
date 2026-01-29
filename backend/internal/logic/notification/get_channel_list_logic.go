package notification

import (
	"context"
	"encoding/json"
	"time"

	"logflux/internal/svc"
	"logflux/internal/types"
	"logflux/model"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetChannelListLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetChannelListLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetChannelListLogic {
	return &GetChannelListLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetChannelListLogic) GetChannelList() (resp *types.ChannelListResp, err error) {
	var channels []model.NotificationChannel
	if err := l.svcCtx.DB.Find(&channels).Error; err != nil {
		return nil, err
	}

	list := make([]types.ChannelItem, 0, len(channels))
	for _, ch := range channels {
		configBytes, _ := json.Marshal(ch.Config)
		eventsBytes, _ := json.Marshal(ch.Events)

		list = append(list, types.ChannelItem{
			ID:          ch.ID,
			Name:        ch.Name,
			Type:        ch.Type,
			Enabled:     ch.Enabled,
			Config:      string(configBytes),
			Events:      string(eventsBytes),
			Description: ch.Description,
			CreatedAt:   ch.CreatedAt.Format(time.DateTime),
			UpdatedAt:   ch.UpdatedAt.Format(time.DateTime),
		})
	}

	return &types.ChannelListResp{
		List: list,
	}, nil
}
