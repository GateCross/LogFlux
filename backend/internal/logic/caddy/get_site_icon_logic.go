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
// 抓不到不是错误：站点离线、没声明 favicon 都属正常，前端回落字母头像即可。
func (l *GetSiteIconLogic) GetSiteIcon(req *types.SiteIconReq) (resp *types.SiteIconResp, err error) {
	return &types.SiteIconResp{Icon: resolveSiteIcon(l.ctx, req.Host)}, nil
}
