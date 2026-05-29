package caddy

import (
	"context"
	"encoding/json"

	"logflux/internal/svc"
	"logflux/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetIpRegionConfigLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetIpRegionConfigLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetIpRegionConfigLogic {
	return &GetIpRegionConfigLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetIpRegionConfigLogic) GetIpRegionConfig() (resp *types.IPRegionConfigResp, err error) {
	cfg, err := l.svcCtx.SystemConfigModel.GetByKey("ip_region")
	if err != nil {
		return &types.IPRegionConfigResp{
			Enabled:   l.svcCtx.Config.IPRegion.Enabled,
			AllowList: l.svcCtx.Config.IPRegion.AllowList,
		}, nil
	}

	var ipCfg struct {
		Enabled   bool     `json:"enabled"`
		AllowList []string `json:"allowList"`
	}
	if err := json.Unmarshal([]byte(cfg.Value), &ipCfg); err != nil {
		return nil, err
	}

	return &types.IPRegionConfigResp{
		Enabled:   ipCfg.Enabled,
		AllowList: ipCfg.AllowList,
	}, nil
}
