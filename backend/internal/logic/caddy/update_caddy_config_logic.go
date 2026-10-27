package caddy

import (
	"context"
	caddymodel "logflux/model/caddy"

	"logflux/internal/svc"
	"logflux/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/gorm"
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
	db := l.svcCtx.DB.WithContext(l.ctx)
	// 配置写的是 system_configs 里的唯一副本，caddy_servers 只提供 Admin API 地址。
	server, err := findPreferredCaddyServer(db)
	if err != nil {
		return nil, err
	}
	// 提前校验一次避免无谓的文本处理，写入器内还会再校验兜底。
	if err := checkBaseHash(db, req.BaseHash); err != nil {
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
	// 方案 A 提示：用户在配置里改了 :80 块但模板未同步时，Apply 会用存储模板
	// 覆盖该改动；这里在 Apply 前检测差异，返回中文提示引导去设置里同步模板。
	msg := "成功"
	if hint, differs := checkEntryBlockDiffers(l.svcCtx.DB, candidate.Config); differs {
		l.Logger.Infof("检测到 :80 入口块与模板不一致，将在应用时被模板覆盖: %q", hint)
		msg = "已保存。:80 为系统入口块，此修改将在下次全局合并时被模板覆盖，可在设置中同步为模板"
	}

	modules := resolveCaddyConfigModules(req.Mode, req.Modules)
	if err := applyService.Apply(server, candidate.Config, modules, configWriteOptions{
		HistoryAction: caddymodel.CaddyHistoryActionUpdate,
		BaseHash:      req.BaseHash,
	}); err != nil {
		return nil, err
	}

	l.Logger.Info("Caddy 配置更新成功")
	return &types.BaseResp{
		Code: 200,
		Msg:  msg,
	}, nil
}

// checkEntryBlockDiffers 判断传入配置里的 :80 块与存储模板是否不一致。
//
// 配置中没有 :80 块（Apply 会追加模板）或内容与模板相同时不提示。
func checkEntryBlockDiffers(db *gorm.DB, config string) (string, bool) {
	extracted, err := extractCaddyEntryBlock(config)
	if err != nil {
		return "", false
	}
	if extracted == loadCaddyEntryBlock(db) {
		return "", false
	}
	return extracted, true
}
