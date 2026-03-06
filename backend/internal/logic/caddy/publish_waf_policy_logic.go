package caddy

import (
	"context"
	"fmt"

	"logflux/internal/notification"
	"logflux/internal/svc"
	"logflux/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
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
	helper := NewWafPolicyNotifyAuditHelper(l.svcCtx, l.Logger)

	defer func() {
		if err != nil {
			err = helper.NotifyFailure(notification.EventSecurityWafPolicyPublishFailed, "WAF 策略发布失败", policyID, policyName, operator, err)
		}
	}()

	if req == nil || req.ID == 0 {
		return nil, fmt.Errorf("policy id is required")
	}
	policyID = req.ID

	publishService := NewPolicyPublishService(l.ctx, l.svcCtx)
	candidate, err := publishService.BuildPublishCandidate(req.ID)
	if err != nil {
		return nil, err
	}
	policyName = candidate.Policy.Name

	if err := publishService.ValidateCandidate(candidate, "publish"); err != nil {
		return nil, fmt.Errorf("policy publish validate failed: %w", err)
	}
	if err := publishService.LoadCandidate(candidate, "publish"); err != nil {
		helper.NotifyAutoRollback(fmt.Sprintf("WAF 策略发布失败，已自动回滚到 last_good：policy=%s", policyName), policyID, policyName, operator)
		return nil, err
	}

	if err := publishService.PersistPublishedCandidate(candidate, operator); err != nil {
		helper.NotifyAutoRollback(fmt.Sprintf("WAF 策略发布落库失败，已自动回滚到 last_good：policy=%s", policyName), policyID, policyName, operator)
		return nil, err
	}

	helper.NotifySuccess(
		notification.EventSecurityWafPolicyPublished,
		"WAF 策略已发布",
		fmt.Sprintf("WAF 策略发布成功：policy=%s", policyName),
		policyID,
		policyName,
		operator,
	)

	go syncCaddyLogSources(l.svcCtx, candidate.Server, l.Logger)

	return &types.BaseResp{Code: 200, Msg: "success"}, nil
}
