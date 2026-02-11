package caddy

import (
	"context"
	"fmt"

	"logflux/internal/svc"
	"logflux/internal/types"
	"logflux/model"

	"github.com/zeromicro/go-zero/core/logx"
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

	items := make([]types.WafPolicyRevisionItem, 0, len(revisions))
	for _, revision := range revisions {
		items = append(items, types.WafPolicyRevisionItem{
			ID:        revision.ID,
			PolicyId:  revision.PolicyID,
			Version:   revision.Version,
			Status:    revision.Status,
			Operator:  revision.Operator,
			Message:   revision.Message,
			CreatedAt: formatTime(revision.CreatedAt),
			UpdatedAt: formatTime(revision.UpdatedAt),
		})
	}

	return &types.WafPolicyRevisionListResp{List: items, Total: total}, nil
}
