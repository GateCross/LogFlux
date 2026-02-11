package caddy

import (
	"context"
	"fmt"

	"logflux/internal/svc"
	"logflux/internal/types"
	"logflux/model"

	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/gorm"
)

type PublishWafPolicyLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewPublishWafPolicyLogic(ctx context.Context, svcCtx *svc.ServiceContext) *PublishWafPolicyLogic {
	return &PublishWafPolicyLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *PublishWafPolicyLogic) PublishWafPolicy(req *types.WafPolicyActionReq) (resp *types.BaseResp, err error) {
	if req == nil || req.ID == 0 {
		return nil, fmt.Errorf("policy id is required")
	}

	var policy model.WafPolicy
	if err := l.svcCtx.DB.First(&policy, req.ID).Error; err != nil {
		return nil, fmt.Errorf("policy not found")
	}

	directives, err := buildWafPolicyDirectives(&policy)
	if err != nil {
		return nil, err
	}

	server, err := findPrimaryCaddyServer(l.svcCtx.DB)
	if err != nil {
		return nil, err
	}

	candidateConfig, err := applyWafPolicyToCaddyConfig(server.Config, directives)
	if err != nil {
		return nil, err
	}

	if err := adaptCaddyfile(server, candidateConfig); err != nil {
		return nil, fmt.Errorf("policy publish validate failed: %w", err)
	}
	if err := loadCaddyfile(server, candidateConfig); err != nil {
		return nil, fmt.Errorf("policy publish load failed: %w", err)
	}

	modules := server.Modules
	if modules == "" {
		modules = emptyModulesJSON
	}

	operator := currentOperatorFromContext(l.ctx)
	if err := l.svcCtx.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&model.CaddyServer{}).
			Where("id = ?", server.ID).
			Updates(map[string]interface{}{
				"config":  candidateConfig,
				"modules": modules,
			}).Error; err != nil {
			return fmt.Errorf("save caddy server config failed: %w", err)
		}

		history := &model.CaddyConfigHistory{
			ServerID: server.ID,
			Action:   "policy_publish",
			Hash:     hashConfig(candidateConfig),
			Config:   candidateConfig,
			Modules:  modules,
		}
		if err := tx.Create(history).Error; err != nil {
			return fmt.Errorf("create caddy config history failed: %w", err)
		}

		revision, err := createPolicyRevision(tx, &policy, wafPolicyStatusPublished, directives, "publish policy", operator)
		if err != nil {
			return err
		}

		if err := markPolicyRevisionsRolledBack(tx, policy.ID, revision.ID); err != nil {
			return err
		}

		return nil
	}); err != nil {
		return nil, err
	}

	go syncCaddyLogSources(l.svcCtx, server, l.Logger)

	return &types.BaseResp{Code: 200, Msg: "success"}, nil
}
