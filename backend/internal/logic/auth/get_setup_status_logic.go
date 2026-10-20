package auth

import (
	"context"

	"logflux/internal/service"
	"logflux/internal/svc"
	"logflux/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetSetupStatusLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetSetupStatusLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetSetupStatusLogic {
	return &GetSetupStatusLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// GetSetupStatus 返回系统是否已完成初始化，前端据此决定是否引导创建首个账号。
func (l *GetSetupStatusLogic) GetSetupStatus() (resp *types.SetupStatusResp, err error) {
	return service.NewAuthService(l.ctx, l.svcCtx).GetSetupStatus()
}
