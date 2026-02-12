package caddy

import (
	"context"
	"fmt"

	"logflux/internal/notification"
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
	policyID := uint(0)
	policyName := ""
	operator := currentOperatorFromContext(l.ctx)

	defer func() {
		if err != nil {
			notifyData := buildWafPolicyNotifyData(policyID, policyName, operator)
			notifyData["error"] = localizeWafPolicyMessage(err.Error())
			notifyWafPolicyEventAsync(
				l.svcCtx,
				l.Logger,
				notification.EventSecurityWafPolicyRollbackFailed,
				notification.LevelError,
				"WAF 策略回滚失败",
				fmt.Sprintf("WAF 策略回滚失败：%s", localizeWafPolicyMessage(err.Error())),
				notifyData,
			)
		}
		err = localizeWafPolicyError(err)
	}()

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
	policyID = policy.ID
	policyName = policy.Name

	directives := revision.DirectivesSnapshot
	if directives == "" {
		builtDirectives, buildErr := buildPolicyDirectivesWithExclusions(l.svcCtx.DB, &policy)
		if buildErr != nil {
			return nil, buildErr
		}
		directives = builtDirectives
	}

	server, err := findPrimaryCaddyServer(l.svcCtx.DB)
	if err != nil {
		return nil, err
	}

	lastGoodConfig := server.Config
	lastGoodModules := normalizeCaddyModulesJSON(server.Modules)

	candidateConfig, err := applyWafPolicyToCaddyConfig(server.Config, directives)
	if err != nil {
		return nil, err
	}

	if err := adaptCaddyfile(server, candidateConfig); err != nil {
		return nil, fmt.Errorf("policy rollback validate failed: %w", err)
	}
	if err := loadCaddyfile(server, candidateConfig); err != nil {
		if rollbackErr := rollbackPolicyConfigToLastGood(server, lastGoodConfig); rollbackErr != nil {
			return nil, fmt.Errorf("policy rollback load failed: %v, rollback to last_good failed: %v", err, rollbackErr)
		}
		notifyWafPolicyEventAsync(
			l.svcCtx,
			l.Logger,
			notification.EventSecurityWafPolicyAutoRollback,
			notification.LevelWarning,
			"WAF 策略自动回滚",
			fmt.Sprintf("WAF 策略回滚加载失败，已自动回滚到 last_good：policy=%s", policyName),
			buildWafPolicyNotifyData(policyID, policyName, operator),
		)
		return nil, fmt.Errorf("policy rollback load failed: %w", err)
	}

	modules := normalizeCaddyModulesJSON(server.Modules)

	if err := l.svcCtx.DB.Transaction(func(tx *gorm.DB) error {
		if err := createCaddyPolicyHistory(tx, server.ID, "policy_last_good", lastGoodConfig, lastGoodModules); err != nil {
			return err
		}

		if err := tx.Model(&model.CaddyServer{}).
			Where("id = ?", server.ID).
			Updates(map[string]interface{}{
				"config":  candidateConfig,
				"modules": modules,
			}).Error; err != nil {
			return fmt.Errorf("save caddy server config failed: %w", err)
		}

		if err := createCaddyPolicyHistory(tx, server.ID, "policy_rollback", candidateConfig, modules); err != nil {
			return err
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
		if rollbackErr := rollbackPolicyConfigToLastGood(server, lastGoodConfig); rollbackErr != nil {
			return nil, fmt.Errorf("policy rollback persist failed: %v, rollback to last_good failed: %v", err, rollbackErr)
		}
		notifyWafPolicyEventAsync(
			l.svcCtx,
			l.Logger,
			notification.EventSecurityWafPolicyAutoRollback,
			notification.LevelWarning,
			"WAF 策略自动回滚",
			fmt.Sprintf("WAF 策略回滚落库失败，已自动回滚到 last_good：policy=%s", policyName),
			buildWafPolicyNotifyData(policyID, policyName, operator),
		)
		return nil, fmt.Errorf("policy rollback persist failed: %w", err)
	}

	notifyWafPolicyEventAsync(
		l.svcCtx,
		l.Logger,
		notification.EventSecurityWafPolicyRollback,
		notification.LevelInfo,
		"WAF 策略已回滚",
		fmt.Sprintf("WAF 策略回滚成功：policy=%s revision=%d", policyName, req.RevisionId),
		buildWafPolicyNotifyData(policyID, policyName, operator),
	)

	go syncCaddyLogSources(l.svcCtx, server, l.Logger)

	return &types.BaseResp{Code: 200, Msg: "success"}, nil
}
