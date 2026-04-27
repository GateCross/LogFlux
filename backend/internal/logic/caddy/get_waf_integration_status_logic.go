package caddy

import (
	"context"
	"fmt"

	"logflux/internal/svc"
	"logflux/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetWafIntegrationStatusLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetWafIntegrationStatusLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetWafIntegrationStatusLogic {
	return &GetWafIntegrationStatusLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetWafIntegrationStatusLogic) GetWafIntegrationStatus() (resp *types.WafIntegrationStatusResp, err error) {
	server, err := findPreferredCaddyServer(l.svcCtx.DB.WithContext(l.ctx), 0)
	if err != nil {
		return nil, fmt.Errorf("query caddy server failed: %w", err)
	}

	applyService := newCaddyConfigApplyService(l.svcCtx, l.Logger)
	config, _, err := applyService.loadCurrent(server)
	if err != nil {
		return &types.WafIntegrationStatusResp{
			ServerId: server.ID,
			Message:  localizeWafPolicyMessage(err.Error()),
		}, nil
	}

	snapshot, err := inspectWafIntegration(config)
	if err != nil {
		return nil, err
	}

	return &types.WafIntegrationStatusResp{
		ServerId:       server.ID,
		Integrated:     snapshot.OrderReady && snapshot.SnippetReady && snapshot.DirectiveReady && len(snapshot.ImportedSites) > 0,
		OrderReady:     snapshot.OrderReady,
		SnippetReady:   snapshot.SnippetReady,
		DirectiveReady: snapshot.DirectiveReady,
		ImportedSites:  snapshot.ImportedSites,
		AvailableSites: snapshot.AvailableSites,
		Message:        buildWafIntegrationStatusMessage(snapshot),
	}, nil
}
