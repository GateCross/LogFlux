// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.1

package caddy

import (
	"context"
	"fmt"

	"logflux/internal/svc"
	"logflux/internal/types"
	"logflux/internal/xerr"
	systemmodel "logflux/model/system"

	"github.com/zeromicro/go-zero/core/logx"
)

type UpdateCaddyLogRetentionLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewUpdateCaddyLogRetentionLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateCaddyLogRetentionLogic {
	return &UpdateCaddyLogRetentionLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// UpdateCaddyLogRetention 保存日志保留策略；下一次清理任务执行时生效。
func (l *UpdateCaddyLogRetentionLogic) UpdateCaddyLogRetention(req *types.CaddyLogRetentionUpdateReq) (resp *types.CaddyLogRetentionResp, err error) {
	if req.RetentionDays < 1 || req.RetentionDays > systemmodel.MaxLogRetentionDays {
		return nil, xerr.NewBusinessErrorWith(fmt.Sprintf("保留天数需为 1~%d 的整数", systemmodel.MaxLogRetentionDays))
	}
	retention := systemmodel.LogRetention{Enabled: req.Enabled, Days: req.RetentionDays}
	if err := systemmodel.SaveLogRetention(l.svcCtx.DB, retention); err != nil {
		return nil, xerr.NewCodeErrorWithCause(xerr.ServerCommonError, "保存日志保留策略失败", err)
	}
	l.Logger.Infof("日志保留策略已更新: enabled=%v, days=%d", req.Enabled, req.RetentionDays)
	return &types.CaddyLogRetentionResp{Enabled: req.Enabled, RetentionDays: req.RetentionDays}, nil
}
