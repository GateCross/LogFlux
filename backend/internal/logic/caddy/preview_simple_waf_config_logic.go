package caddy

import (
	"context"

	"logflux/internal/svc"
	"logflux/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type PreviewSimpleWafConfigLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewPreviewSimpleWafConfigLogic(ctx context.Context, svcCtx *svc.ServiceContext) *PreviewSimpleWafConfigLogic {
	return &PreviewSimpleWafConfigLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *PreviewSimpleWafConfigLogic) PreviewSimpleWafConfig(req *types.SimpleWafConfigUpdateReq) (resp *types.SimpleWafConfigResp, err error) {
	defer func() {
		err = localizeWafPolicyError(err)
	}()

	service := newSimpleWafConfigService(l.ctx, l.svcCtx, l.Logger)
	return service.Preview(req)
}
