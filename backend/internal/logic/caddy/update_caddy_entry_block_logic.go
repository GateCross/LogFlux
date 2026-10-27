// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.1

package caddy

import (
	"context"
	"fmt"
	"strings"

	"logflux/internal/svc"
	"logflux/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type UpdateCaddyEntryBlockLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewUpdateCaddyEntryBlockLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateCaddyEntryBlockLogic {
	return &UpdateCaddyEntryBlockLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// UpdateCaddyEntryBlock 保存入口块模板；只落库不触发 Apply，整体应用时统一合并。
//
// syncFromConfig 为 true 时忽略请求体 raw，改从当前生效配置提取 :80 块
// 存为模板——用于用户在配置编辑器里改好入口块后"同步为模板"。
func (l *UpdateCaddyEntryBlockLogic) UpdateCaddyEntryBlock(req *types.CaddyEntryBlockUpdateReq) (resp *types.CaddyEntryBlockResp, err error) {
	var raw string
	if req.SyncFromConfig {
		current, _, err := loadCurrentCaddyConfig(l.svcCtx.DB.WithContext(l.ctx))
		if err != nil {
			return nil, err
		}
		extracted, err := extractCaddyEntryBlock(current)
		if err != nil {
			return nil, err
		}
		raw = extracted
	} else {
		raw = strings.TrimSpace(strings.ReplaceAll(req.Raw, "\r\n", "\n"))
		if raw == "" {
			return nil, fmt.Errorf("入口块模板不能为空")
		}
	}
	if _, err := mergeEntryBlock("", raw); err != nil {
		return nil, err
	}
	if err := saveCaddyEntryBlock(l.svcCtx.DB, raw); err != nil {
		return nil, fmt.Errorf("保存入口块模板失败: %w", err)
	}
	l.Logger.Info("Caddy 入口块模板已保存，等待整体应用时合并")
	return &types.CaddyEntryBlockResp{Raw: raw}, nil
}
