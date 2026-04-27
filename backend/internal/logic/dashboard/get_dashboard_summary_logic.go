package dashboard

import (
	"context"

	"logflux/internal/service"
	"logflux/internal/svc"
	"logflux/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetDashboardSummaryLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetDashboardSummaryLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetDashboardSummaryLogic {
	return &GetDashboardSummaryLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetDashboardSummaryLogic) GetDashboardSummary(req *types.DashboardSummaryReq) (resp *types.DashboardSummaryResp, err error) {
	return service.NewDashboardService(l.ctx, l.svcCtx).GetSummary(req)
}
