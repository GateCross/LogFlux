package caddy

import (
	"context"

	"logflux/internal/svc"
	"logflux/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetSiteIconLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetSiteIconLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetSiteIconLogic {
	return &GetSiteIconLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// GetSiteIcon 返回站点自带图标（data URL）。
// 优先直接访问上游地址抓取图标：若未显式传上游地址，则根据 host 自动从配置中解析该站点的反代上游。
func (l *GetSiteIconLogic) GetSiteIcon(req *types.SiteIconReq) (resp *types.SiteIconResp, err error) {
	upstream := req.Upstream
	if upstream == "" && req.Host != "" && l.svcCtx.DB != nil {
		upstream = findUpstreamByHost(l.svcCtx.DB.WithContext(l.ctx), req.Host)
	}
	return &types.SiteIconResp{Icon: resolveSiteIcon(l.ctx, upstream, req.Host)}, nil
}
