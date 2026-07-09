package log

import (
	"context"

	"logflux/internal/service"
	"logflux/internal/svc"
	"logflux/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetSiteMetricsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetSiteMetricsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetSiteMetricsLogic {
	return &GetSiteMetricsLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetSiteMetricsLogic) GetSiteMetrics(req *types.SiteMetricsReq) (resp *types.SiteMetricsResp, err error) {
	return service.NewLogService(l.ctx, l.svcCtx).GetSiteMetrics(req)
}
