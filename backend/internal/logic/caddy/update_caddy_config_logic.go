package caddy

import (
	"context"
	"fmt"
	caddymodel "logflux/model/caddy"

	"logflux/internal/svc"
	"logflux/internal/types"

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
	var server caddymodel.CaddyServer
	if err := l.svcCtx.DB.WithContext(l.ctx).First(&server, req.ServerId).Error; err != nil {
		return nil, fmt.Errorf("服务器不存在")
	}
	// 提前校验一次避免无谓的文本处理，写入器内还会再校验兜底。
	if err := checkBaseHash(&server, req.BaseHash); err != nil {
		return nil, err
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
	modules := resolveCaddyConfigModules(req.Mode, req.Modules)
	if err := applyService.Apply(&server, candidate.Config, modules, configWriteOptions{
		HistoryAction: caddymodel.CaddyHistoryActionUpdate,
		BaseHash:      req.BaseHash,
	}); err != nil {
		return nil, err
	}

	l.Logger.Info("Caddy 配置更新成功")
	return &types.BaseResp{
		Code: 200,
		Msg:  "成功",
	}, nil
}
