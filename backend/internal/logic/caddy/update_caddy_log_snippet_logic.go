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

type UpdateCaddyLogSnippetLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewUpdateCaddyLogSnippetLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateCaddyLogSnippetLogic {
	return &UpdateCaddyLogSnippetLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// UpdateCaddyLogSnippet 保存日志 snippet 定义；只落库不触发 Apply，整体应用时统一合并。
//
// 保存前先用空配置试合并一次：块头不对或花括号不配对在这里就报出来，避免坏定义
// 拖到下一次应用才在 adapt 阶段失败。
func (l *UpdateCaddyLogSnippetLogic) UpdateCaddyLogSnippet(req *types.CaddyLogSnippetUpdateReq) (resp *types.CaddyLogSnippetResp, err error) {
	raw := strings.TrimSpace(strings.ReplaceAll(req.Raw, "\r\n", "\n"))
	if raw == "" {
		return nil, fmt.Errorf("日志 snippet 定义不能为空")
	}
	if _, err := mergeCaddyLogSnippet("", raw); err != nil {
		return nil, err
	}
	if err := saveCaddyLogSnippet(l.svcCtx.DB, raw); err != nil {
		return nil, fmt.Errorf("保存日志 snippet 定义失败: %w", err)
	}
	l.Logger.Info("Caddy 日志 snippet 定义已保存，等待整体应用时合并")
	return &types.CaddyLogSnippetResp{Raw: raw}, nil
}
