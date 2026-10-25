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

type ResetCaddyEntryBlockLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewResetCaddyEntryBlockLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ResetCaddyEntryBlockLogic {
	return &ResetCaddyEntryBlockLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// ResetCaddyEntryBlock 把内置默认模板写回存储键，同样只落库不触发 Apply。
func (l *ResetCaddyEntryBlockLogic) ResetCaddyEntryBlock() (resp *types.CaddyEntryBlockResp, err error) {
	raw := defaultCaddyEntryBlockNormalized()
	if err := saveCaddyEntryBlock(l.svcCtx.DB, raw); err != nil {
		return nil, fmt.Errorf("重置入口块模板失败: %w", err)
	}
	l.Logger.Info("Caddy 入口块模板已重置为默认")
	return &types.CaddyEntryBlockResp{Raw: raw}, nil
}
