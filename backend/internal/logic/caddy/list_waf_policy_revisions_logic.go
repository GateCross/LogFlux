package caddy

import (
	"context"
	"fmt"
	"strings"

	"logflux/internal/svc"
	"logflux/internal/types"
	"logflux/model"

	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/gorm"
)

type ListWafPolicyRevisionsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewListWafPolicyRevisionsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListWafPolicyRevisionsLogic {
	return &ListWafPolicyRevisionsLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ListWafPolicyRevisionsLogic) ListWafPolicyRevisions(req *types.WafPolicyRevisionListReq) (resp *types.WafPolicyRevisionListResp, err error) {
	defer func() {
		err = localizeWafPolicyError(err)
	}()

	page := req.Page
	if page <= 0 {
		page = 1
	}
	pageSize := req.PageSize
	if pageSize <= 0 {
		pageSize = 20
	}

	db := l.svcCtx.DB.Model(&model.WafPolicyRevision{})
	if req.PolicyId > 0 {
		db = db.Where("policy_id = ?", req.PolicyId)
	}

	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, fmt.Errorf("count policy revisions failed: %w", err)
	}

	var revisions []model.WafPolicyRevision
	offset := (page - 1) * pageSize
	if err := db.Order("created_at desc, id desc").Limit(pageSize).Offset(offset).Find(&revisions).Error; err != nil {
		return nil, fmt.Errorf("query policy revisions failed: %w", err)
	}

	policyNameMap := make(map[uint]string, len(revisions))
	if len(revisions) > 0 {
		policyIDs := make([]uint, 0, len(revisions))
		seen := make(map[uint]struct{}, len(revisions))
		for _, revision := range revisions {
			if revision.PolicyID == 0 {
				continue
			}
			if _, ok := seen[revision.PolicyID]; ok {
				continue
			}
			seen[revision.PolicyID] = struct{}{}
			policyIDs = append(policyIDs, revision.PolicyID)
		}
		if len(policyIDs) > 0 {
			var policies []model.WafPolicy
			if err := l.svcCtx.DB.Model(&model.WafPolicy{}).Where("id IN ?", policyIDs).Find(&policies).Error; err != nil {
				return nil, fmt.Errorf("query policy names failed: %w", err)
			}
			for _, policy := range policies {
				policyNameMap[policy.ID] = strings.TrimSpace(policy.Name)
			}
		}
	}

	items := make([]types.WafPolicyRevisionItem, 0, len(revisions))
	for _, revision := range revisions {
		var previousRevision *model.WafPolicyRevision
		if revision.Version > 1 {
			var prev model.WafPolicyRevision
			prevErr := l.svcCtx.DB.
				Where("policy_id = ? AND version < ?", revision.PolicyID, revision.Version).
				Order("version desc").
				First(&prev).Error
			if prevErr == nil {
				previousRevision = &prev
			} else if prevErr != gorm.ErrRecordNotFound {
				return nil, fmt.Errorf("query previous policy revision failed: %w", prevErr)
			}
		}

		policyName := strings.TrimSpace(policyNameMap[revision.PolicyID])
		if policyName == "" {
			policyName = fmt.Sprintf("#%d", revision.PolicyID)
		}

		items = append(items, types.WafPolicyRevisionItem{
			ID:            revision.ID,
			PolicyId:      revision.PolicyID,
			PolicyName:    policyName,
			Version:       revision.Version,
			Status:        revision.Status,
			Operator:      revision.Operator,
			Message:       revision.Message,
			ChangeSummary: buildWafPolicyRevisionChangeSummary(&revision, previousRevision),
			CreatedAt:     formatTime(revision.CreatedAt),
			UpdatedAt:     formatTime(revision.UpdatedAt),
		})
	}

	return &types.WafPolicyRevisionListResp{List: items, Total: total}, nil
}
