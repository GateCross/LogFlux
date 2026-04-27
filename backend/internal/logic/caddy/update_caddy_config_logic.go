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

const emptyModulesJSON = "{}"

type UpdateCaddyConfigLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewUpdateCaddyConfigLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateCaddyConfigLogic {
	return &UpdateCaddyConfigLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *UpdateCaddyConfigLogic) UpdateCaddyConfig(req *types.CaddyConfigUpdateReq) (resp *types.BaseResp, err error) {
	var server model.CaddyServer
	if err := l.svcCtx.DB.WithContext(l.ctx).First(&server, req.ServerId).Error; err != nil {
		return nil, fmt.Errorf("服务器不存在")
	}
	applyService := newCaddyConfigApplyService(l.svcCtx, l.Logger)

	modulesPayload := strings.TrimSpace(req.Modules)
	if modulesPayload == "" {
		if err := l.updateRawConfig(applyService, &server, req.Config); err != nil {
			return nil, err
		}
	} else {
		if err := l.updateStructuredConfig(applyService, &server, req.Config, req.Modules); err != nil {
			return nil, err
		}
	}

	l.Logger.Info("Caddy 配置更新成功")
	return &types.BaseResp{
		Code: 200,
		Msg:  "成功",
	}, nil
}

func (l *UpdateCaddyConfigLogic) updateRawConfig(applyService *caddyConfigApplyService, server *model.CaddyServer, config string) error {
	modules := strings.TrimSpace(server.Modules)
	if modules == "" {
		modules = emptyModulesJSON
	}
	return applyService.apply(server, config, modules, "update")
}

func (l *UpdateCaddyConfigLogic) updateStructuredConfig(applyService *caddyConfigApplyService, server *model.CaddyServer, config, modules string) error {
	return applyService.apply(server, config, modules, "update")
}
