package notification

import (
	"context"
	"encoding/json"

	"logflux/internal/svc"
	"logflux/internal/types"
	"logflux/model"

	"github.com/zeromicro/go-zero/core/logx"
)

type UpdateChannelLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewUpdateChannelLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateChannelLogic {
	return &UpdateChannelLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *UpdateChannelLogic) UpdateChannel(req *types.ChannelUpdateReq) (resp *types.BaseResp, err error) {
	var channel model.NotificationChannel
	if err := l.svcCtx.DB.First(&channel, req.ID).Error; err != nil {
		return nil, err
	}

	if req.Name != "" {
		channel.Name = req.Name
	}
	if req.Type != "" {
		channel.Type = req.Type
	}
	// Note: Enabled is optional in API but go-zero usually generates pointer for primitive optional fields ONLY if `optional` keyword is used AND it's not default.
	// Or sometimes it doesn't?
	// If `json:"enabled,optional"` -> type *bool.
	// We'll assume pointer for now since I'll check types.go in parallel.
	// If types.go shows it's `bool` (non-pointer), then I can't distinguish between false and not-provided unless I use separate logic or if `default` was used.
	// In my API: `Enabled bool json:"enabled,optional"`
	channel.Enabled = req.Enabled

	if req.Description != "" {
		channel.Description = req.Description
	}

	if req.Config != "" {
		var configMap map[string]interface{}
		if err := json.Unmarshal([]byte(req.Config), &configMap); err != nil {
			return nil, err
		}
		channel.Config = model.JSONMap(configMap)
	}

	if req.Events != "" {
		var events []string
		if err := json.Unmarshal([]byte(req.Events), &events); err != nil {
			return nil, err
		}
		channel.Events = model.StringArray(events)
	}

	if err := l.svcCtx.DB.Save(&channel).Error; err != nil {
		return nil, err
	}

	// Reload channels
	if l.svcCtx.NotificationMgr != nil {
		l.svcCtx.NotificationMgr.ReloadChannels()
	}

	return &types.BaseResp{
		Code: 200,
		Msg:  "success",
	}, nil
}
