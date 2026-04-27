package user

import (
	"context"

	"logflux/internal/service"
	"logflux/internal/svc"
	"logflux/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type UpdateUserPreferencesLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewUpdateUserPreferencesLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateUserPreferencesLogic {
	return &UpdateUserPreferencesLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *UpdateUserPreferencesLogic) UpdateUserPreferences(req *types.UserPreferencesReq) (resp *types.BaseResp, err error) {
	return service.NewUserService(l.ctx, l.svcCtx).UpdateUserPreferences(req)
}
