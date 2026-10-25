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

type UpdateCaddyGlobalOptionsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewUpdateCaddyGlobalOptionsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateCaddyGlobalOptionsLogic {
	return &UpdateCaddyGlobalOptionsLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// UpdateCaddyGlobalOptions 只落库不触发 Apply：前端确认后会另行整体保存，
// 届时 Apply 内统一合并进配置。
func (l *UpdateCaddyGlobalOptionsLogic) UpdateCaddyGlobalOptions(req *types.CaddyGlobalOptionsUpdateReq) (resp *types.CaddyGlobalOptionsResp, err error) {
	value, err := globalOptionsReqToValue(req)
	if err != nil {
		return nil, err
	}
	if err := saveCaddyGlobalOptions(l.svcCtx.DB, value); err != nil {
		return nil, fmt.Errorf("保存全局选项失败: %w", err)
	}
	l.Logger.Info("Caddy 全局选项已保存，等待整体应用时合并")
	return globalOptionsValueToResp(value), nil
}

// globalOptionsReqToValue 把请求转为存储结构；Debug 为 true 时记录显式开启
// （存储里用指针区分"未设置"），extraRawOptions 逐行去空白拼接。
func globalOptionsReqToValue(req *types.CaddyGlobalOptionsUpdateReq) (CaddyGlobalOptionsValue, error) {
	value := CaddyGlobalOptionsValue{
		AutoHttps:      strings.TrimSpace(req.AutoHttps),
		TrustedProxies: req.TrustedProxies,
		Email:          strings.TrimSpace(req.Email),
	}
	if req.Debug {
		debugOn := true
		value.Debug = &debugOn
	}
	var rawLines []string
	for _, raw := range strings.Split(req.ExtraRawOptions, "\n") {
		trimmed := strings.TrimSpace(raw)
		if trimmed == "" {
			continue
		}
		// 逃生舱逐行原样写入全局块，不允许携带大括号破坏块结构。
		if strings.ContainsAny(trimmed, "{}") {
			return value, fmt.Errorf("额外原生选项不支持块结构: %s", trimmed)
		}
		rawLines = append(rawLines, trimmed)
	}
	value.ExtraRawOptions = strings.Join(rawLines, "\n")
	return value, nil
}
