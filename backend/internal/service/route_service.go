package service

import (
	"context"
	"encoding/json"
	menumodel "logflux/model/menu"
	rolemodel "logflux/model/role"
	"strings"

	"logflux/internal/svc"
	"logflux/internal/types"
	"logflux/internal/utils/logger"
	"logflux/internal/xerr"
)

// RouteService 负责前端路由树业务。
type RouteService struct {
	logger.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// NewRouteService 创建路由服务。
func NewRouteService(ctx context.Context, svcCtx *svc.ServiceContext) *RouteService {
	return &RouteService{
		Logger: logger.New(logger.ModuleSystem).WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (s *RouteService) GetUserRoutes() (*types.UserRouteResp, error) {
	userID, err := userIDFromContext(s.ctx)
	if err != nil {
		return nil, xerr.NewBusinessErrorWith(err.Error())
	}

	user, err := s.svcCtx.UserModel.FindByID(s.ctx, userID)
	if err != nil {
		return nil, xerr.NewCodeErrorWithCause(xerr.ServerCommonError, "查询用户失败", err)
	}

	roles, err := s.svcCtx.RoleModel.FindByNames(s.ctx, user.Roles)
	if err != nil {
		return nil, xerr.NewCodeErrorWithCause(xerr.ServerCommonError, "查询角色权限失败", err)
	}
	permissions := make(map[string]bool)
	for _, role := range roles {
		for _, permission := range role.Permissions {
			permissions[permission] = true
		}
	}

	menus, err := s.svcCtx.MenuModel.FindAll(s.ctx)
	if err != nil {
		return nil, xerr.NewCodeErrorWithCause(xerr.ServerCommonError, "查询菜单失败", err)
	}

	return &types.UserRouteResp{
		Home:   "dashboard",
		Routes: buildRouteTree(menus, nil, roles, permissions),
	}, nil
}

func buildRouteTree(allMenus []menumodel.Menu, parentID *uint, userRoles []rolemodel.Role, permissions map[string]bool) []types.MenuRoute {
	routes := make([]types.MenuRoute, 0)
	for _, menu := range allMenus {
		if isRetiredMenu(menu) || !matchMenuParent(menu, parentID) || !hasMenuPermission(menu, userRoles, permissions) {
			continue
		}

		meta := parseRouteMeta(menu)
		if meta.HideInMenu {
			continue
		}

		route := types.MenuRoute{
			Name:      menu.Name,
			Path:      menu.Path,
			Component: menu.Component,
			Meta:      meta,
		}
		if children := buildRouteTree(allMenus, &menu.ID, userRoles, permissions); len(children) > 0 {
			route.Children = children
		}
		routes = append(routes, route)
	}
	return routes
}

func isRetiredMenu(menu menumodel.Menu) bool {
	normalizedName := strings.ToLower(strings.TrimSpace(menu.Name))
	normalizedPath := strings.ToLower(strings.TrimSpace(menu.Path))
	return normalizedName == "security" ||
		normalizedName == "waf" ||
		normalizedName == "waf_security" ||
		normalizedPath == "/security" ||
		strings.HasPrefix(normalizedPath, "/security/")
}

func parseRouteMeta(menu menumodel.Menu) types.RouteMeta {
	var meta types.RouteMeta
	if menu.Meta != "" {
		_ = json.Unmarshal([]byte(menu.Meta), &meta)
	}
	if meta.Order == 0 && menu.Order != 0 {
		meta.Order = menu.Order
	}
	return meta
}

func hasMenuPermission(menu menumodel.Menu, userRoles []rolemodel.Role, permissions map[string]bool) bool {
	for _, role := range userRoles {
		if role.Name == "admin" {
			return true
		}
	}
	for _, requiredRole := range menu.RequiredRoles {
		for _, role := range userRoles {
			if role.Name == requiredRole {
				return true
			}
		}
	}
	if key := menuPermissionKey(menu.Name); key != "" && permissions[key] {
		return true
	}
	return len(menu.RequiredRoles) == 0 && menuPermissionKey(menu.Name) == ""
}

func menuPermissionKey(name string) string {
	normalized := strings.ToLower(strings.TrimSpace(name))
	switch normalized {
	case "dashboard":
		return "dashboard"
	case "caddy":
		return "caddy"
	case "caddy_config", "caddy-config":
		return "caddy_config"
	case "caddy_access", "caddy-access":
		return "caddy_access"
	case "caddy_source", "caddy-source":
		return "caddy_source"
	case "caddy_log", "caddy-log":
		return "logs_caddy"
	case "caddy_system_log", "caddy_system-log", "system_log", "system-log":
		return "logs"
	case "cron":
		return "cron"
	case "user":
		return ""
	default:
		return normalized
	}
}
