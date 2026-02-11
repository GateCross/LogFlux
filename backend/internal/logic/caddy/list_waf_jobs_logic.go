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

type ListWafJobsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewListWafJobsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListWafJobsLogic {
	return &ListWafJobsLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ListWafJobsLogic) ListWafJobs(req *types.WafJobListReq) (resp *types.WafJobListResp, err error) {
	helper := newWafLogicHelper(l.ctx, l.svcCtx, l.Logger)

	page := req.Page
	if page <= 0 {
		page = 1
	}
	pageSize := req.PageSize
	if pageSize <= 0 {
		pageSize = 20
	}

	db := helper.svcCtx.DB.Model(&model.WafUpdateJob{})
	if status := strings.TrimSpace(req.Status); status != "" {
		db = db.Where("status = ?", strings.ToLower(status))
	}
	if action := strings.TrimSpace(req.Action); action != "" {
		db = db.Where("action = ?", strings.ToLower(action))
	}

	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, fmt.Errorf("count jobs failed: %w", err)
	}

	var jobs []model.WafUpdateJob
	offset := (page - 1) * pageSize
	if err := db.Order("created_at desc, id desc").Limit(pageSize).Offset(offset).Find(&jobs).Error; err != nil {
		return nil, fmt.Errorf("query jobs failed: %w", err)
	}

	items := make([]types.WafJobItem, 0, len(jobs))
	for _, job := range jobs {
		items = append(items, types.WafJobItem{
			ID:          job.ID,
			SourceId:    job.SourceID,
			ReleaseId:   job.ReleaseID,
			Action:      job.Action,
			TriggerMode: job.TriggerMode,
			Operator:    job.Operator,
			Status:      job.Status,
			Message:     job.Message,
			StartedAt:   formatNullableTime(job.StartedAt),
			FinishedAt:  formatNullableTime(job.FinishedAt),
			CreatedAt:   formatTime(job.CreatedAt),
		})
	}

	return &types.WafJobListResp{List: items, Total: total}, nil
}
