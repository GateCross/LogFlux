// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.1

package caddy

import (
	"context"
	"fmt"

	"logflux/internal/svc"
	"logflux/internal/types"
	"logflux/internal/utils"
	"logflux/internal/xerr"
	wafmodel "logflux/model/waf"

	"github.com/zeromicro/go-zero/core/logx"
)

type ListWafEventsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewListWafEventsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListWafEventsLogic {
	return &ListWafEventsLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ListWafEventsLogic) ListWafEvents(req *types.WafEventListReq) (resp *types.WafEventListResp, err error) {
	start, err := utils.ParseOptionalTime(req.StartTime)
	if err != nil {
		return nil, xerr.NewBusinessErrorWith(fmt.Sprintf("开始时间格式无效: %v", err))
	}
	end, err := utils.ParseOptionalTime(req.EndTime)
	if err != nil {
		return nil, xerr.NewBusinessErrorWith(fmt.Sprintf("结束时间格式无效: %v", err))
	}

	events, total, err := l.svcCtx.WafEventModel.List(l.ctx, wafmodel.WafEventQuery{
		Page:      req.Page,
		PageSize:  req.PageSize,
		StartTime: start,
		EndTime:   end,
		ClientIP:  req.ClientIP,
		Host:      req.Host,
		RuleID:    req.RuleID,
		Severity:  req.Severity,
		Action:    req.Action,
		Keyword:   req.Keyword,
	})
	if err != nil {
		l.Logger.Errorf("查询 WAF 事件列表失败: %v", err)
		return nil, xerr.NewCodeErrorWithCause(xerr.ServerCommonError, "查询 WAF 事件列表失败", err)
	}

	items := make([]types.WafEventItem, 0, len(events))
	for _, event := range events {
		items = append(items, types.WafEventItem{
			ID:            event.ID,
			EventTime:     formatTime(event.EventTime),
			TransactionID: event.TransactionID,
			RuleID:        event.RuleID,
			RuleMsg:       event.RuleMsg,
			Severity:      event.Severity,
			Interrupted:   event.Interrupted,
			Status:        event.Status,
			ClientIP:      event.ClientIP,
			Country:       event.Country,
			Province:      event.Province,
			City:          event.City,
			Host:          event.Host,
			Method:        event.Method,
			Uri:           event.Uri,
			UserAgent:     event.UserAgent,
		})
	}
	return &types.WafEventListResp{List: items, Total: total}, nil
}
