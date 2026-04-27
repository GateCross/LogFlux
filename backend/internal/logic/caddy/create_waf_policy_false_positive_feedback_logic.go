package caddy

import (
	"context"
	"fmt"
	"strings"

	"logflux/internal/svc"
	"logflux/internal/types"
	"logflux/model"

	"github.com/zeromicro/go-zero/core/logx"
)

type CreateWafPolicyFalsePositiveFeedbackLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewCreateWafPolicyFalsePositiveFeedbackLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CreateWafPolicyFalsePositiveFeedbackLogic {
	return &CreateWafPolicyFalsePositiveFeedbackLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *CreateWafPolicyFalsePositiveFeedbackLogic) CreateWafPolicyFalsePositiveFeedback(req *types.WafPolicyFalsePositiveFeedbackReq) (resp *types.BaseResp, err error) {
	defer func() {
		err = localizeWafPolicyError(err)
	}()

	if req == nil {
		return nil, fmt.Errorf("误报反馈参数不合法")
	}
	if req.PolicyId > 0 {
		if err := validatePolicyIDExists(l.svcCtx.DB.WithContext(l.ctx), req.PolicyId); err != nil {
			return nil, err
		}
	}

	host := normalizePolicyScopeHost(req.Host)
	path := strings.TrimSpace(req.Path)
	if path != "" {
		path = normalizePolicyScopePath(path)
	}
	method := normalizePolicyHTTPMethod(req.Method)
	if err := validatePolicyHTTPMethod(method); err != nil {
		return nil, err
	}

	reason := strings.TrimSpace(req.Reason)
	if reason == "" {
		return nil, fmt.Errorf("反馈原因不能为空")
	}

	status := req.Status
	if status <= 0 {
		status = 403
	}
	dueAt, err := parsePolicyFeedbackDueAt(req.DueAt)
	if err != nil {
		return nil, err
	}

	feedback := &model.WafPolicyFalsePositiveFeedback{
		PolicyID:       req.PolicyId,
		Host:           host,
		Path:           path,
		Method:         method,
		Status:         status,
		FeedbackStatus: wafFeedbackStatusPending,
		Assignee:       strings.TrimSpace(req.Assignee),
		DueAt:          dueAt,
		SampleURI:      strings.TrimSpace(req.SampleURI),
		Reason:         reason,
		Suggestion:     strings.TrimSpace(req.Suggestion),
		Operator:       currentOperatorFromContext(l.ctx),
		ProcessNote:    "",
		ProcessedBy:    "",
		ProcessedAt:    nil,
	}
	if err := l.svcCtx.DB.WithContext(l.ctx).Create(feedback).Error; err != nil {
		return nil, fmt.Errorf("创建误报反馈失败: %w", err)
	}

	return &types.BaseResp{
		Code: 200,
		Msg:  "成功",
	}, nil
}
