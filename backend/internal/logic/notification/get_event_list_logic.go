package notification

import (
	"context"

	"logflux/internal/notification"
	"logflux/internal/svc"
	"logflux/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetEventListLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetEventListLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetEventListLogic {
	return &GetEventListLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetEventListLogic) GetEventList() (resp *types.EventListResp, err error) {
	resp = &types.EventListResp{
		List: make([]types.EventItem, 0),
	}

	// 1. 事件级别
	resp.List = append(resp.List,
		types.EventItem{Value: notification.LevelCritical, Label: "严重", Group: "事件级别"},
		types.EventItem{Value: notification.LevelError, Label: "错误", Group: "事件级别"},
		types.EventItem{Value: notification.LevelWarning, Label: "警告", Group: "事件级别"},
		types.EventItem{Value: notification.LevelInfo, Label: "信息", Group: "事件级别"},
		types.EventItem{Value: "debug", Label: "调试", Group: "事件级别"},
	)

	// 2. 系统事件
	resp.List = append(resp.List,
		types.EventItem{Value: notification.EventSystemStartup, Label: "系统启动", Group: "系统事件"},
		types.EventItem{Value: notification.EventSystemShutdown, Label: "系统关闭", Group: "系统事件"},
		types.EventItem{Value: notification.EventSystemError, Label: "系统错误", Group: "系统事件"},
		types.EventItem{Value: notification.EventRedisConnectionFailed, Label: "Redis连接失败", Group: "系统事件"},
		types.EventItem{Value: notification.EventDatabaseConnectionFailed, Label: "数据库连接失败", Group: "系统事件"},
	)

	// 3. 日志采集事件
	resp.List = append(resp.List,
		types.EventItem{Value: notification.EventLogParseError, Label: "日志解析错误", Group: "日志事件"},
		types.EventItem{Value: notification.EventLogIngestFailed, Label: "日志写入失败", Group: "日志事件"},
		types.EventItem{Value: notification.EventLogHighErrorRate, Label: "高错误率", Group: "日志事件"},
		types.EventItem{Value: notification.EventLogSuspiciousIP, Label: "可疑IP访问", Group: "日志事件"},
		types.EventItem{Value: notification.EventLogCollectionStopped, Label: "采集停止", Group: "日志事件"},
	)

	// 4. 归档事件
	resp.List = append(resp.List,
		types.EventItem{Value: notification.EventArchiveFailed, Label: "归档失败", Group: "归档事件"},
		types.EventItem{Value: notification.EventArchiveCompleted, Label: "归档完成", Group: "归档事件"},
		types.EventItem{Value: notification.EventArchiveSlow, Label: "归档慢", Group: "归档事件"},
		types.EventItem{Value: notification.EventArchiveAnomaly, Label: "归档异常", Group: "归档事件"},
	)

	// 5. Caddy 配置事件
	resp.List = append(resp.List,
		types.EventItem{Value: notification.EventCaddyConfigUpdateFailed, Label: "Caddy配置更新失败", Group: "Caddy事件"},
		types.EventItem{Value: notification.EventCaddyConfigUpdateSuccess, Label: "Caddy配置更新成功", Group: "Caddy事件"},
		types.EventItem{Value: notification.EventCaddyLogSourceDiscovered, Label: "发现Caddy日志源", Group: "Caddy事件"},
	)

	// 6. 安全事件
	resp.List = append(resp.List,
		types.EventItem{Value: notification.EventSecurityLoginFailed, Label: "登录失败", Group: "安全事件"},
		types.EventItem{Value: notification.EventSecurityBruteForce, Label: "暴力破解", Group: "安全事件"},
		types.EventItem{Value: notification.EventSecurityAdminLogin, Label: "管理员登录", Group: "安全事件"},
		types.EventItem{Value: notification.EventSecurityPermissionDenied, Label: "权限拒绝", Group: "安全事件"},

	)

	return resp, nil
}
