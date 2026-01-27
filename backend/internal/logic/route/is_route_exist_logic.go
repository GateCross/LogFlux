package route

import (
	"context"

	"logflux/internal/svc"
	"logflux/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type IsRouteExistLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewIsRouteExistLogic(ctx context.Context, svcCtx *svc.ServiceContext) *IsRouteExistLogic {
	return &IsRouteExistLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *IsRouteExistLogic) IsRouteExist(req *types.IsRouteExistReq) (resp bool, err error) {
	// For now, always return true or check against a constant list if needed.
	// Returning true to allow frontend navigation.
	return true, nil
}
