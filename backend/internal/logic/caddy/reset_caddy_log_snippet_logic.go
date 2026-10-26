// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.1

package caddy

import (
	"context"
	"fmt"

	"logflux/internal/svc"
	"logflux/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type ResetCaddyLogSnippetLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewResetCaddyLogSnippetLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ResetCaddyLogSnippetLogic {
	return &ResetCaddyLogSnippetLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// ResetCaddyLogSnippet 把内置默认定义写回存储键，同样只落库不触发 Apply。
func (l *ResetCaddyLogSnippetLogic) ResetCaddyLogSnippet() (resp *types.CaddyLogSnippetResp, err error) {
	raw := defaultCaddyLogSnippetNormalized()
	if err := saveCaddyLogSnippet(l.svcCtx.DB, raw); err != nil {
		return nil, fmt.Errorf("重置日志 snippet 定义失败: %w", err)
	}
	l.Logger.Info("Caddy 日志 snippet 定义已重置为默认")
	return &types.CaddyLogSnippetResp{Raw: raw}, nil
}
