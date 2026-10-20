package auth

import (
	"context"

	"logflux/internal/service"
	"logflux/internal/svc"
	"logflux/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type SetupLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewSetupLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SetupLogic {
	return &SetupLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// Setup 创建首个管理员账号，仅当系统还没有任何账号时允许调用。
func (l *SetupLogic) Setup(req *types.SetupReq) (resp *types.LoginResp, err error) {
	return service.NewAuthService(l.ctx, l.svcCtx).Setup(req)
}
