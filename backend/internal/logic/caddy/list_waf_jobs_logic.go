package caddy

import (
	"context"
	"fmt"
	"strconv"
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

	operatorNameMap := l.buildJobOperatorNameMap(jobs)
	items := make([]types.WafJobItem, 0, len(jobs))
	for _, job := range jobs {
		items = append(items, types.WafJobItem{
			ID:          job.ID,
			SourceId:    job.SourceID,
			ReleaseId:   job.ReleaseID,
			Action:      job.Action,
			TriggerMode: job.TriggerMode,
			Operator:    mapJobOperator(job.Operator, operatorNameMap),
			Status:      job.Status,
			Message:     job.Message,
			StartedAt:   formatNullableTime(job.StartedAt),
			FinishedAt:  formatNullableTime(job.FinishedAt),
			CreatedAt:   formatTime(job.CreatedAt),
		})
	}

	return &types.WafJobListResp{List: items, Total: total}, nil
}

func (l *ListWafJobsLogic) buildJobOperatorNameMap(jobs []model.WafUpdateJob) map[string]string {
	operatorNameMap := map[string]string{}
	if len(jobs) == 0 || l == nil || l.svcCtx == nil || l.svcCtx.DB == nil {
		return operatorNameMap
	}

	userIDs := make([]uint, 0, len(jobs))
	seenUserID := make(map[uint]struct{}, len(jobs))
	for _, job := range jobs {
		userID, ok := parseJobOperatorUserID(job.Operator)
		if !ok {
			continue
		}
		if _, exists := seenUserID[userID]; exists {
			continue
		}
		seenUserID[userID] = struct{}{}
		userIDs = append(userIDs, userID)
	}
	if len(userIDs) == 0 {
		return operatorNameMap
	}

	var users []model.User
	if err := l.svcCtx.DB.Model(&model.User{}).Select("id", "username").Where("id IN ?", userIDs).Find(&users).Error; err != nil {
		l.Logger.Errorf("query operator usernames failed: userIDs=%v err=%v", userIDs, err)
		return operatorNameMap
	}

	for _, user := range users {
		username := strings.TrimSpace(user.Username)
		if username == "" {
			continue
		}
		operatorNameMap[strconv.FormatUint(uint64(user.ID), 10)] = username
	}
	return operatorNameMap
}

func parseJobOperatorUserID(operator string) (uint, bool) {
	trimmed := strings.TrimSpace(operator)
	if trimmed == "" || strings.EqualFold(trimmed, "system") {
		return 0, false
	}

	userID, err := strconv.ParseUint(trimmed, 10, 64)
	if err != nil || userID == 0 {
		return 0, false
	}

	return uint(userID), true
}

func mapJobOperator(rawOperator string, operatorNameMap map[string]string) string {
	operator := strings.TrimSpace(rawOperator)
	if operator == "" {
		return "system"
	}

	if username, ok := operatorNameMap[operator]; ok {
		trimmedUsername := strings.TrimSpace(username)
		if trimmedUsername != "" {
			return trimmedUsername
		}
	}

	return operator
}
