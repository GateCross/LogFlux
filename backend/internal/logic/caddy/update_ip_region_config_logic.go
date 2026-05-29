package caddy

import (
	"context"
	"encoding/json"

	"logflux/internal/svc"
	"logflux/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type UpdateIpRegionConfigLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewUpdateIpRegionConfigLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateIpRegionConfigLogic {
	return &UpdateIpRegionConfigLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *UpdateIpRegionConfigLogic) UpdateIpRegionConfig(req *types.IPRegionConfigUpdateReq) (resp *types.IPRegionConfigResp, err error) {
	if req.Enabled && len(req.AllowList) == 0 {
		req.AllowList = []string{"中国"}
	}

	value, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	if err := l.svcCtx.SystemConfigModel.SetByKey("ip_region", string(value)); err != nil {
		return nil, err
	}

	l.svcCtx.IPRegionMgr.Reload(req.Enabled, req.AllowList)

	return &types.IPRegionConfigResp{
		Enabled:   req.Enabled,
		AllowList: req.AllowList,
	}, nil
}
