package notification

import (
	"context"

	"encoding/json"
	"logflux/internal/svc"
	"logflux/internal/types"
	"logflux/model"

	"github.com/zeromicro/go-zero/core/logx"
)

type CreateChannelLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewCreateChannelLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CreateChannelLogic {
	return &CreateChannelLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *CreateChannelLogic) CreateChannel(req *types.ChannelReq) (resp *types.BaseResp, err error) {
	var configMap map[string]interface{}
	if req.Config != "" {
		if err := json.Unmarshal([]byte(req.Config), &configMap); err != nil {
			return nil, err
		}
	}

	var events []string
	if req.Events != "" {
		if err := json.Unmarshal([]byte(req.Events), &events); err != nil {
			return nil, err
		}
	}

	channel := &model.NotificationChannel{
		Name:        req.Name,
		Type:        req.Type,
		Enabled:     req.Enabled,
		Description: req.Description,
		Config:      model.JSONMap(configMap),
		Events:      model.StringArray(events),
	}

	if err := l.svcCtx.DB.Create(channel).Error; err != nil {
		return nil, err
	}

	// Reload channels in NotificationManager
	if l.svcCtx.NotificationMgr != nil {
		if err := l.svcCtx.NotificationMgr.ReloadChannels(); err != nil {
			l.Logger.Errorf("Failed to reload channels: %v", err)
		}
	}

	return &types.BaseResp{
		Code: 200,
		Msg:  "success",
	}, nil
}
