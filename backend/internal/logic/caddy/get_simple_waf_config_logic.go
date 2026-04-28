package caddy

import (
	"context"

	"logflux/internal/svc"
	"logflux/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetSimpleWafConfigLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetSimpleWafConfigLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetSimpleWafConfigLogic {
	return &GetSimpleWafConfigLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetSimpleWafConfigLogic) GetSimpleWafConfig(req *types.SimpleWafConfigReq) (resp *types.SimpleWafConfigResp, err error) {
	defer func() {
		err = localizeWafPolicyError(err)
	}()

	service := newSimpleWafConfigService(l.ctx, l.svcCtx, l.Logger)
	return service.Get(req)
}
