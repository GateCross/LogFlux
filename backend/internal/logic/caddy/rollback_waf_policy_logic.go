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

type RollbackWafPolicyLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewRollbackWafPolicyLogic(ctx context.Context, svcCtx *svc.ServiceContext) *RollbackWafPolicyLogic {
	return &RollbackWafPolicyLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *RollbackWafPolicyLogic) RollbackWafPolicy(req *types.WafPolicyRollbackReq) (resp *types.BaseResp, err error) {
	if req == nil || req.RevisionId == 0 {
		return nil, fmt.Errorf("revisionId is required")
	}

	var revision model.WafPolicyRevision
	if err := l.svcCtx.DB.First(&revision, req.RevisionId).Error; err != nil {
		return nil, fmt.Errorf("policy revision not found")
	}

	if revision.PolicyID == 0 {
		return nil, fmt.Errorf("invalid policy revision")
	}

	var policy model.WafPolicy
	if err := l.svcCtx.DB.First(&policy, revision.PolicyID).Error; err != nil {
		return nil, fmt.Errorf("policy not found")
	}

	directives := revision.DirectivesSnapshot
	if directives == "" {
		builtDirectives, buildErr := buildWafPolicyDirectives(&policy)
		if buildErr != nil {
			return nil, buildErr
		}
		directives = builtDirectives
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
		return nil, fmt.Errorf("policy rollback validate failed: %w", err)
	}
	if err := loadCaddyfile(server, candidateConfig); err != nil {
		return nil, fmt.Errorf("policy rollback load failed: %w", err)
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
			Action:   "policy_rollback",
			Hash:     hashConfig(candidateConfig),
			Config:   candidateConfig,
			Modules:  modules,
		}
		if err := tx.Create(history).Error; err != nil {
			return fmt.Errorf("create caddy config history failed: %w", err)
		}

		if err := markPolicyRevisionsRolledBack(tx, revision.PolicyID, revision.ID); err != nil {
			return err
		}

		if err := tx.Model(&model.WafPolicyRevision{}).
			Where("id = ?", revision.ID).
			Updates(map[string]interface{}{
				"status":   wafPolicyStatusPublished,
				"operator": operator,
				"message":  "rollback policy",
			}).Error; err != nil {
			return fmt.Errorf("update revision status failed: %w", err)
		}

		if _, err := createPolicyRevision(tx, &policy, wafPolicyStatusRolledBack, directives, "rollback policy", operator); err != nil {
			return err
		}

		return nil
	}); err != nil {
		return nil, err
	}

	go syncCaddyLogSources(l.svcCtx, server, l.Logger)

	return &types.BaseResp{Code: 200, Msg: "success"}, nil
}
