// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.1

package caddy

import (
	"context"

	"logflux/internal/svc"
	"logflux/internal/types"
	systemmodel "logflux/model/system"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetCaddyLogRetentionLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetCaddyLogRetentionLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetCaddyLogRetentionLogic {
	return &GetCaddyLogRetentionLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// GetCaddyLogRetention 返回日志保留策略，未配置时为默认开启并保留 7 天。
func (l *GetCaddyLogRetentionLogic) GetCaddyLogRetention() (resp *types.CaddyLogRetentionResp, err error) {
	retention := systemmodel.LoadLogRetention(l.svcCtx.DB)
	return &types.CaddyLogRetentionResp{Enabled: retention.Enabled, RetentionDays: retention.Days}, nil
}
