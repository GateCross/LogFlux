package caddy

import (
	"context"
	"fmt"

	"logflux/internal/notification"
	"logflux/internal/svc"
	"logflux/internal/types"
	"logflux/internal/utils/safego"

	"github.com/zeromicro/go-zero/core/logx"
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
	helper := NewWafPolicyNotifyAuditHelper(l.svcCtx, l.Logger)

	defer func() {
		if err != nil {
			err = helper.NotifyFailure(notification.EventSecurityWafPolicyRollbackFailed, "WAF 策略回滚失败", policyID, policyName, operator, err)
		}
	}()

	if req == nil || req.RevisionId == 0 {
		return nil, fmt.Errorf("revisionId is required")
	}

	publishService := NewPolicyPublishService(l.ctx, l.svcCtx)
	candidate, revision, err := publishService.BuildRollbackCandidate(req.RevisionId)
	if err != nil {
		return nil, err
	}
	policyID = candidate.Policy.ID
	policyName = candidate.Policy.Name

	if err := publishService.ValidateCandidate(candidate, "rollback"); err != nil {
		return nil, fmt.Errorf("policy rollback validate failed: %w", err)
	}
	if err := publishService.LoadCandidate(candidate, "rollback"); err != nil {
		helper.NotifyAutoRollback(fmt.Sprintf("WAF 策略回滚加载失败，已自动回滚到 last_good：policy=%s", policyName), policyID, policyName, operator)
		return nil, err
	}

	if err := publishService.PersistRolledBackCandidate(candidate, revision, operator); err != nil {
		helper.NotifyAutoRollback(fmt.Sprintf("WAF 策略回滚落库失败，已自动回滚到 last_good：policy=%s", policyName), policyID, policyName, operator)
		return nil, err
	}

	helper.NotifySuccess(
		notification.EventSecurityWafPolicyRollback,
		"WAF 策略已回滚",
		fmt.Sprintf("WAF 策略回滚成功：policy=%s revision=%d", policyName, req.RevisionId),
		policyID,
		policyName,
		operator,
	)

	safego.New(context.Background(), "回滚 WAF 策略后同步日志源").Go(func() {
		syncCaddyLogSources(l.svcCtx, candidate.Server, l.Logger)
	})

	return &types.BaseResp{Code: 200, Msg: "success"}, nil
}
