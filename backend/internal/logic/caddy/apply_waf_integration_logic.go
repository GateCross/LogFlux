package caddy

import (
	"context"
	"fmt"
	"strings"

	"logflux/internal/svc"
	"logflux/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type ApplyWafIntegrationLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewApplyWafIntegrationLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ApplyWafIntegrationLogic {
	return &ApplyWafIntegrationLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ApplyWafIntegrationLogic) ApplyWafIntegration(req *types.WafIntegrationApplyReq) (resp *types.WafIntegrationApplyResp, err error) {
	if req == nil {
		return nil, fmt.Errorf("invalid waf integration payload")
	}

	server, err := findPreferredCaddyServer(l.svcCtx.DB.WithContext(l.ctx), req.ServerId)
	if err != nil {
		return nil, err
	}

	applyService := newCaddyConfigApplyService(l.svcCtx, l.Logger)
	config, modules, err := applyService.loadCurrent(server)
	if err != nil {
		return nil, err
	}

	snapshot, err := inspectWafIntegration(config)
	if err != nil {
		return nil, err
	}

	targetSites := buildWafIntegrationTargetSites(snapshot.AvailableSites, req.ApplyAll, req.SiteAddresses)
	if req.Enabled && len(targetSites) == 0 {
		return nil, fmt.Errorf("site addresses is empty")
	}

	nextConfig := config
	actions := make([]string, 0)
	changed := false

	if req.Enabled {
		var changedStep bool
		nextConfig, changedStep, err = ensureCorazaOrder(nextConfig)
		if err != nil {
			return nil, err
		}
		if changedStep {
			changed = true
			actions = append(actions, "注入全局 order coraza_waf first")
		}

		nextConfig, changedStep, err = ensureWafProtectSnippet(nextConfig)
		if err != nil {
			return nil, err
		}
		if changedStep {
			changed = true
			actions = append(actions, "注入 waf_protect 统一片段")
		}

		for _, siteAddress := range targetSites {
			nextConfig, changedStep, err = ensureSiteImport(nextConfig, siteAddress)
			if err != nil {
				return nil, err
			}
			if changedStep {
				changed = true
				actions = append(actions, fmt.Sprintf("为站点 %s 挂载 waf_protect", siteAddress))
			}
		}
	} else {
		for _, siteAddress := range targetSites {
			var changedStep bool
			nextConfig, changedStep, err = removeSiteImport(nextConfig, siteAddress)
			if err != nil {
				return nil, err
			}
			if changedStep {
				changed = true
				actions = append(actions, fmt.Sprintf("取消站点 %s 的 waf_protect 挂载", siteAddress))
			}
		}
	}

	updatedSnapshot, err := inspectWafIntegration(nextConfig)
	if err != nil {
		return nil, err
	}

	message := buildWafIntegrationApplyMessage(req.Enabled, changed, req.DryRun, updatedSnapshot)
	resp = &types.WafIntegrationApplyResp{
		ServerId:      server.ID,
		Enabled:       req.Enabled,
		Changed:       changed,
		ImportedSites: updatedSnapshot.ImportedSites,
		Actions:       actions,
		Config:        nextConfig,
		Message:       message,
	}

	if req.DryRun || !changed {
		return resp, nil
	}

	historyAction := "waf_integration_disable"
	if req.Enabled {
		historyAction = "waf_integration_enable"
	}
	if err := applyService.apply(server, nextConfig, modules, historyAction); err != nil {
		return nil, err
	}
	return resp, nil
}

func buildWafIntegrationTargetSites(availableSites []string, applyAll bool, requested []string) []string {
	if len(availableSites) == 0 {
		return nil
	}
	if applyAll {
		return append([]string{}, availableSites...)
	}
	requestedSet := make(map[string]struct{}, len(requested))
	for _, item := range requested {
		trimmed := strings.TrimSpace(item)
		if trimmed == "" {
			continue
		}
		requestedSet[trimmed] = struct{}{}
	}
	if len(requestedSet) == 0 {
		return nil
	}
	targets := make([]string, 0, len(requestedSet))
	for _, address := range availableSites {
		if _, ok := requestedSet[address]; ok {
			targets = append(targets, address)
		}
	}
	return targets
}

func buildWafIntegrationApplyMessage(enabled, changed, dryRun bool, snapshot *wafIntegrationSnapshot) string {
	if dryRun {
		if changed {
			return "已生成 WAF 接入配置预览，尚未实际推送到 Caddy"
		}
		return "当前配置无需变更，预览结果与现状一致"
	}
	if enabled {
		if changed {
			return "WAF 接入配置已应用，可继续在策略中心切换运行模式"
		}
		return buildWafIntegrationStatusMessage(snapshot)
	}
	if changed {
		return "已取消目标站点的 WAF 挂载，保留全局片段供后续再次接入"
	}
	return "目标站点当前未挂载 WAF，无需变更"
}
