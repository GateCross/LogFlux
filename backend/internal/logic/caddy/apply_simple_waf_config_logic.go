package caddy

import (
	"context"

	"logflux/internal/svc"
	"logflux/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type ApplySimpleWafConfigLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewApplySimpleWafConfigLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ApplySimpleWafConfigLogic {
	return &ApplySimpleWafConfigLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ApplySimpleWafConfigLogic) ApplySimpleWafConfig(req *types.SimpleWafConfigUpdateReq) (resp *types.SimpleWafConfigResp, err error) {
	defer func() {
		err = localizeWafPolicyError(err)
	}()

	service := newSimpleWafConfigService(l.ctx, l.svcCtx, l.Logger)
	return service.Apply(req)
}
