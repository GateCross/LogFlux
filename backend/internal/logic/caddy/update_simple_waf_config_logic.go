package caddy

import (
	"context"

	"logflux/internal/svc"
	"logflux/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type UpdateSimpleWafConfigLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewUpdateSimpleWafConfigLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateSimpleWafConfigLogic {
	return &UpdateSimpleWafConfigLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *UpdateSimpleWafConfigLogic) UpdateSimpleWafConfig(req *types.SimpleWafConfigUpdateReq) (resp *types.BaseResp, err error) {
	defer func() {
		err = localizeWafPolicyError(err)
	}()

	service := newSimpleWafConfigService(l.ctx, l.svcCtx, l.Logger)
	if err := service.Save(req); err != nil {
		return nil, err
	}
	return &types.BaseResp{Code: 200, Msg: "成功"}, nil
}
