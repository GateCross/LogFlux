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
// 优先直接访问上游地址抓取图标，抓不到时不视为错误，由前端回落字母头像。
func (l *GetSiteIconLogic) GetSiteIcon(req *types.SiteIconReq) (resp *types.SiteIconResp, err error) {
	return &types.SiteIconResp{Icon: resolveSiteIcon(l.ctx, req.Upstream, req.Host)}, nil
}
