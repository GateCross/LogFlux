package caddy

import (
	"context"
	"fmt"
	"strings"

	"logflux/internal/svc"
	"logflux/internal/types"
	"logflux/model"

	"github.com/zeromicro/go-zero/core/logx"
)

type PreviewCaddyConfigLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewPreviewCaddyConfigLogic(ctx context.Context, svcCtx *svc.ServiceContext) *PreviewCaddyConfigLogic {
	return &PreviewCaddyConfigLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *PreviewCaddyConfigLogic) PreviewCaddyConfig(req *types.CaddyConfigPreviewReq) (resp *types.CaddyConfigPreviewResp, err error) {
	var server model.CaddyServer
	if err := l.svcCtx.DB.WithContext(l.ctx).First(&server, req.ServerId).Error; err != nil {
		return nil, fmt.Errorf("服务器不存在")
	}

	configService := newCaddyConfigService()
	candidate, err := configService.Prepare(caddyConfigPrepareInput{
		Mode:    req.Mode,
		Config:  req.Config,
		Modules: req.Modules,
	})
	if err != nil {
		return &types.CaddyConfigPreviewResp{
			Valid:   false,
			Config:  strings.TrimSpace(req.Config),
			Errors:  []string{err.Error()},
			Actions: []string{"生成配置失败"},
		}, nil
	}

	resp = &types.CaddyConfigPreviewResp{
		Valid:   true,
		Config:  candidate.Config,
		Errors:  []string{},
		Actions: append([]string{}, candidate.Actions...),
	}
	if err := adaptCaddyfile(&server, candidate.Config); err != nil {
		resp.Valid = false
		resp.Errors = append(resp.Errors, fmt.Sprintf("Caddy /adapt 校验失败: %v", err))
		resp.Actions = append(resp.Actions, "未执行 Caddy /load")
		return resp, nil
	}
	resp.Actions = append(resp.Actions, "Caddy /adapt 校验通过", "预览未执行 Caddy /load")

	return resp, nil
}
