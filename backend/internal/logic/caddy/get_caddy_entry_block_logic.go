// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.1

package caddy

import (
	"context"

	"logflux/internal/svc"
	"logflux/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetCaddyEntryBlockLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetCaddyEntryBlockLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetCaddyEntryBlockLogic {
	return &GetCaddyEntryBlockLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetCaddyEntryBlockLogic) GetCaddyEntryBlock() (resp *types.CaddyEntryBlockResp, err error) {
	return &types.CaddyEntryBlockResp{Raw: loadCaddyEntryBlock(l.svcCtx.DB)}, nil
}
