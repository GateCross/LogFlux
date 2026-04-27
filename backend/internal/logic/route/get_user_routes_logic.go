package route

import (
	"context"

	"logflux/internal/service"
	"logflux/internal/svc"
	"logflux/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetUserRoutesLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetUserRoutesLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetUserRoutesLogic {
	return &GetUserRoutesLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetUserRoutesLogic) GetUserRoutes() (resp *types.UserRouteResp, err error) {
	return service.NewRouteService(l.ctx, l.svcCtx).GetUserRoutes()
}

func menuPermissionKey(menuName string) string {
	switch menuName {
	case "dashboard":
		return "dashboard"
	case "manage":
		return "manage"
	case "manage_user":
		return "manage_user"
	case "manage_role":
		return "manage_role"
	case "manage_menu":
		return "manage_menu"
	case "notification":
		return "notification"
	case "notification_channel":
		return "notification_channel"
	case "notification_rule":
		return "notification_rule"
	case "notification_template":
		return "notification_template"
	case "notification_log":
		return "notification_log"
	case "caddy_system_log":
		fallthrough
	case "caddy_system-log":
		return "logs"
	case "caddy_log":
		return "logs_caddy"
	case "security":
		return "security"
	case "user_center":
		return "user_center"
	default:
		return ""
	}
}
