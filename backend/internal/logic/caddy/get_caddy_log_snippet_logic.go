// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.1

package caddy

import (
	"context"

	"logflux/internal/svc"
	"logflux/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetCaddyLogSnippetLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetCaddyLogSnippetLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetCaddyLogSnippetLogic {
	return &GetCaddyLogSnippetLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetCaddyLogSnippetLogic) GetCaddyLogSnippet() (resp *types.CaddyLogSnippetResp, err error) {
	return &types.CaddyLogSnippetResp{Raw: loadCaddyLogSnippet(l.svcCtx.DB)}, nil
}
