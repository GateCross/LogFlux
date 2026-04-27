package caddy

import (
	"context"
	"fmt"

	"logflux/internal/svc"
	"logflux/model"

	"gorm.io/gorm"
)

type PolicyPublishCandidate struct {
	Policy          *model.WafPolicy
	Directives      string
	Server          *model.CaddyServer
	CandidateConfig string
	LastGoodConfig  string
	LastGoodModules string
}

type PolicyPublishService struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewPolicyPublishService(ctx context.Context, svcCtx *svc.ServiceContext) *PolicyPublishService {
	return &PolicyPublishService{ctx: ctx, svcCtx: svcCtx}
}

func (s *PolicyPublishService) BuildPublishCandidate(policyID uint) (*PolicyPublishCandidate, error) {
	if s == nil || s.svcCtx == nil || s.svcCtx.DB == nil {
		return nil, fmt.Errorf("数据库为空")
	}
	if policyID == 0 {
		return nil, fmt.Errorf("策略 ID 不能为空")
	}

	var policy model.WafPolicy
	if err := s.svcCtx.DB.WithContext(s.ctx).First(&policy, policyID).Error; err != nil {
		return nil, fmt.Errorf("策略不存在")
	}

	if err := ensureNoPolicyBindingConflicts(s.svcCtx.DB.WithContext(s.ctx)); err != nil {
		return nil, err
	}

	directives, err := buildPolicyDirectivesWithExclusions(s.svcCtx.DB.WithContext(s.ctx), &policy)
	if err != nil {
		return nil, err
	}

	server, err := findPrimaryCaddyServer(s.svcCtx.DB.WithContext(s.ctx))
	if err != nil {
		return nil, err
	}

	candidateConfig, err := applyWafPolicyToCaddyConfig(server.Config, directives)
	if err != nil {
		return nil, err
	}

	return &PolicyPublishCandidate{
		Policy:          &policy,
		Directives:      directives,
		Server:          server,
		CandidateConfig: candidateConfig,
		LastGoodConfig:  server.Config,
		LastGoodModules: normalizeCaddyModulesJSON(server.Modules),
	}, nil
}

func (s *PolicyPublishService) BuildRollbackCandidate(revisionID uint) (*PolicyPublishCandidate, *model.WafPolicyRevision, error) {
	if s == nil || s.svcCtx == nil || s.svcCtx.DB == nil {
		return nil, nil, fmt.Errorf("数据库为空")
	}
	if revisionID == 0 {
		return nil, nil, fmt.Errorf("策略版本 ID 不能为空")
	}

	var revision model.WafPolicyRevision
	if err := s.svcCtx.DB.WithContext(s.ctx).First(&revision, revisionID).Error; err != nil {
		return nil, nil, fmt.Errorf("策略版本不存在")
	}
	if revision.PolicyID == 0 {
		return nil, nil, fmt.Errorf("策略版本无效")
	}

	var policy model.WafPolicy
	if err := s.svcCtx.DB.WithContext(s.ctx).First(&policy, revision.PolicyID).Error; err != nil {
		return nil, nil, fmt.Errorf("策略不存在")
	}

	directives := revision.DirectivesSnapshot
	if directives == "" {
		builtDirectives, err := buildPolicyDirectivesWithExclusions(s.svcCtx.DB.WithContext(s.ctx), &policy)
		if err != nil {
			return nil, nil, err
		}
		directives = builtDirectives
	}

	server, err := findPrimaryCaddyServer(s.svcCtx.DB.WithContext(s.ctx))
	if err != nil {
		return nil, nil, err
	}

	candidateConfig, err := applyWafPolicyToCaddyConfig(server.Config, directives)
	if err != nil {
		return nil, nil, err
	}

	return &PolicyPublishCandidate{
		Policy:          &policy,
		Directives:      directives,
		Server:          server,
		CandidateConfig: candidateConfig,
		LastGoodConfig:  server.Config,
		LastGoodModules: normalizeCaddyModulesJSON(server.Modules),
	}, &revision, nil
}

func (s *PolicyPublishService) ValidateCandidate(candidate *PolicyPublishCandidate, action string) error {
	if candidate == nil || candidate.Server == nil {
		return fmt.Errorf("Caddy 服务器不存在")
	}
	if err := adaptCaddyfile(candidate.Server, candidate.CandidateConfig); err != nil {
		return fmt.Errorf("策略 %s 校验失败: %w", action, err)
	}
	return nil
}

func (s *PolicyPublishService) LoadCandidate(candidate *PolicyPublishCandidate, action string) error {
	if candidate == nil || candidate.Server == nil {
		return fmt.Errorf("Caddy 服务器不存在")
	}
	if err := loadCaddyfile(candidate.Server, candidate.CandidateConfig); err != nil {
		if rollbackErr := rollbackPolicyConfigToLastGood(candidate.Server, candidate.LastGoodConfig); rollbackErr != nil {
			return fmt.Errorf("策略 %s 加载失败: %v，回滚到 last_good 失败: %v", action, err, rollbackErr)
		}
		return fmt.Errorf("策略 %s 加载失败: %w", action, err)
	}
	return nil
}

func (s *PolicyPublishService) PersistPublishedCandidate(candidate *PolicyPublishCandidate, operator string) error {
	if candidate == nil || candidate.Policy == nil || candidate.Server == nil {
		return fmt.Errorf("发布候选配置无效")
	}
	modules := normalizeCaddyModulesJSON(candidate.Server.Modules)

	if err := s.svcCtx.DB.WithContext(s.ctx).Transaction(func(tx *gorm.DB) error {
		if err := createCaddyPolicyHistory(tx, candidate.Server.ID, "policy_last_good", candidate.LastGoodConfig, candidate.LastGoodModules); err != nil {
			return err
		}
		if err := tx.Model(&model.CaddyServer{}).
			Where("id = ?", candidate.Server.ID).
			Updates(map[string]interface{}{
				"config":  candidate.CandidateConfig,
				"modules": modules,
			}).Error; err != nil {
			return fmt.Errorf("保存 Caddy 服务器配置失败: %w", err)
		}
		if err := createCaddyPolicyHistory(tx, candidate.Server.ID, "policy_publish", candidate.CandidateConfig, modules); err != nil {
			return err
		}
		revision, err := createPolicyRevision(tx, candidate.Policy, wafPolicyStatusPublished, candidate.Directives, "publish policy", operator)
		if err != nil {
			return err
		}
		return markPolicyRevisionsRolledBack(tx, candidate.Policy.ID, revision.ID)
	}); err != nil {
		if rollbackErr := rollbackPolicyConfigToLastGood(candidate.Server, candidate.LastGoodConfig); rollbackErr != nil {
			return fmt.Errorf("策略发布持久化失败: %v，回滚到 last_good 失败: %v", err, rollbackErr)
		}
		return fmt.Errorf("策略发布持久化失败: %w", err)
	}

	return nil
}

func (s *PolicyPublishService) PersistRolledBackCandidate(candidate *PolicyPublishCandidate, revision *model.WafPolicyRevision, operator string) error {
	if candidate == nil || candidate.Policy == nil || candidate.Server == nil || revision == nil {
		return fmt.Errorf("回滚候选配置无效")
	}
	modules := normalizeCaddyModulesJSON(candidate.Server.Modules)

	if err := s.svcCtx.DB.WithContext(s.ctx).Transaction(func(tx *gorm.DB) error {
		if err := createCaddyPolicyHistory(tx, candidate.Server.ID, "policy_last_good", candidate.LastGoodConfig, candidate.LastGoodModules); err != nil {
			return err
		}
		if err := tx.Model(&model.CaddyServer{}).
			Where("id = ?", candidate.Server.ID).
			Updates(map[string]interface{}{
				"config":  candidate.CandidateConfig,
				"modules": modules,
			}).Error; err != nil {
			return fmt.Errorf("保存 Caddy 服务器配置失败: %w", err)
		}
		if err := createCaddyPolicyHistory(tx, candidate.Server.ID, "policy_rollback", candidate.CandidateConfig, modules); err != nil {
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
			return fmt.Errorf("更新版本状态失败: %w", err)
		}
		_, err := createPolicyRevision(tx, candidate.Policy, wafPolicyStatusRolledBack, candidate.Directives, "rollback policy", operator)
		return err
	}); err != nil {
		if rollbackErr := rollbackPolicyConfigToLastGood(candidate.Server, candidate.LastGoodConfig); rollbackErr != nil {
			return fmt.Errorf("策略回滚持久化失败: %v，回滚到 last_good 失败: %v", err, rollbackErr)
		}
		return fmt.Errorf("策略回滚持久化失败: %w", err)
	}

	return nil
}
