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
				notification.EventSecurityWafPolicyPublishFailed,
				notification.LevelError,
				"WAF 策略发布失败",
				fmt.Sprintf("WAF 策略发布失败：%s", localizeWafPolicyMessage(err.Error())),
				notifyData,
			)
		}
		err = localizeWafPolicyError(err)
	}()

	if req == nil || req.ID == 0 {
		return nil, fmt.Errorf("policy id is required")
	}
	policyID = req.ID

	var policy model.WafPolicy
	if err := l.svcCtx.DB.First(&policy, req.ID).Error; err != nil {
		return nil, fmt.Errorf("policy not found")
	}
	policyName = policy.Name

	if err := ensureNoPolicyBindingConflicts(l.svcCtx.DB); err != nil {
		return nil, err
	}

	directives, err := buildPolicyDirectivesWithExclusions(l.svcCtx.DB, &policy)
	if err != nil {
		return nil, err
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
		return nil, fmt.Errorf("policy publish validate failed: %w", err)
	}
	if err := loadCaddyfile(server, candidateConfig); err != nil {
		if rollbackErr := rollbackPolicyConfigToLastGood(server, lastGoodConfig); rollbackErr != nil {
			return nil, fmt.Errorf("policy publish load failed: %v, rollback to last_good failed: %v", err, rollbackErr)
		}
		notifyWafPolicyEventAsync(
			l.svcCtx,
			l.Logger,
			notification.EventSecurityWafPolicyAutoRollback,
			notification.LevelWarning,
			"WAF 策略自动回滚",
			fmt.Sprintf("WAF 策略发布失败，已自动回滚到 last_good：policy=%s", policyName),
			buildWafPolicyNotifyData(policyID, policyName, operator),
		)
		return nil, fmt.Errorf("policy publish load failed: %w", err)
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

		if err := createCaddyPolicyHistory(tx, server.ID, "policy_publish", candidateConfig, modules); err != nil {
			return err
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
		if rollbackErr := rollbackPolicyConfigToLastGood(server, lastGoodConfig); rollbackErr != nil {
			return nil, fmt.Errorf("policy publish persist failed: %v, rollback to last_good failed: %v", err, rollbackErr)
		}
		notifyWafPolicyEventAsync(
			l.svcCtx,
			l.Logger,
			notification.EventSecurityWafPolicyAutoRollback,
			notification.LevelWarning,
			"WAF 策略自动回滚",
			fmt.Sprintf("WAF 策略发布落库失败，已自动回滚到 last_good：policy=%s", policyName),
			buildWafPolicyNotifyData(policyID, policyName, operator),
		)
		return nil, fmt.Errorf("policy publish persist failed: %w", err)
	}

	notifyWafPolicyEventAsync(
		l.svcCtx,
		l.Logger,
		notification.EventSecurityWafPolicyPublished,
		notification.LevelInfo,
		"WAF 策略已发布",
		fmt.Sprintf("WAF 策略发布成功：policy=%s", policyName),
		buildWafPolicyNotifyData(policyID, policyName, operator),
	)

	go syncCaddyLogSources(l.svcCtx, server, l.Logger)

	return &types.BaseResp{Code: 200, Msg: "success"}, nil
}
