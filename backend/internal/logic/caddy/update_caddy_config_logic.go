package caddy

import (
	"context"
	"fmt"

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
	configService := newCaddyConfigService()
	candidate, err := configService.Prepare(caddyConfigPrepareInput{
		Mode:    req.Mode,
		Config:  req.Config,
		Modules: req.Modules,
	})
	if err != nil {
		return nil, err
	}
	modules := resolveCaddyConfigModulesForUpdate(req.Mode, req.Modules)
	if err := applyService.apply(&server, candidate.Config, modules, "update"); err != nil {
		return nil, err
	}

	l.Logger.Info("Caddy 配置更新成功")
	return &types.BaseResp{
		Code: 200,
		Msg:  "成功",
	}, nil
}
