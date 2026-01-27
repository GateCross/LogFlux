package route

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"logflux/internal/svc"
	"logflux/internal/types"
	"logflux/model"

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
	// 获取当前用户信息
	userId := l.ctx.Value("userId")

	var user model.User
	if err := l.svcCtx.DB.First(&user, userId).Error; err != nil {
		return nil, err
	}

	// 获取用户角色的权限列表
	userPermissions := make(map[string]bool)
	var roles []model.Role

	// 修复：正确处理 PostgreSQL 数组查询
	// user.Roles 是 pq.StringArray 类型
	if len(user.Roles) > 0 {
		// 使用 ANY 操作符处理 PostgreSQL 数组
		l.svcCtx.DB.Where("name = ANY(?)", user.Roles).Find(&roles)
	}

	for _, role := range roles {
		for _, perm := range role.Permissions {
			userPermissions[perm] = true
		}
	}

	// 构建基于权限的路由
	routes := l.buildRoutes(userPermissions)

	// DEBUG: Write to file
	debugMsg := fmt.Sprintf("Time: %s\nUser ID: %v\nRoles: %v\nUser Permissions Map: %v\nBuilt Routes: %d\nFetched Roles Dump: %+v\n\n", time.Now().Format(time.RFC3339), userId, user.Roles, userPermissions, len(routes), roles)
	f, _ := os.OpenFile("debug.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if f != nil {
		f.WriteString(debugMsg)
		f.Close()
	}

	return &types.UserRouteResp{
		Home:   "dashboard",
		Routes: routes,
	}, nil
}

// buildRoutes 根据用户权限构建路由
func (l *GetUserRoutesLogic) buildRoutes(permissions map[string]bool) []types.MenuRoute {
	routes := []types.MenuRoute{}

	// 自动添加父级权限：如果有任何子权限，就认为拥有父权限
	if permissions["logs_caddy"] {
		permissions["logs"] = true
	}
	if permissions["manage_user"] || permissions["manage_role"] {
		permissions["manage"] = true
	}

	// Dashboard 路由（所有角色都可访问）
	if permissions["dashboard"] {
		routes = append(routes, types.MenuRoute{
			Name:      "dashboard",
			Path:      "/dashboard",
			Component: "layout.base$view.dashboard",
			Meta: types.RouteMeta{
				Title:   "dashboard",
				I18nKey: "route.dashboard",
				Icon:    "mdi:monitor-dashboard",
				Order:   1,
			},
		})
	}

	// Logs 路由（仅 admin 和 analyst）
	if permissions["logs"] {
		logsChildren := []types.MenuRoute{}

		if permissions["logs_caddy"] {
			logsChildren = append(logsChildren, types.MenuRoute{
				Name:      "logs_caddy",
				Path:      "/logs/caddy",
				Component: "view.logs_caddy",
				Meta: types.RouteMeta{
					Title:   "logs_caddy",
					I18nKey: "route.logs_caddy",
					Icon:    "mdi:file-document",
				},
			})
		}

		if len(logsChildren) > 0 {
			routes = append(routes, types.MenuRoute{
				Name:      "logs",
				Path:      "/logs",
				Component: "layout.base",
				Meta: types.RouteMeta{
					Title:   "logs",
					I18nKey: "route.logs",
					Icon:    "mdi:file-document-multiple",
					Order:   5,
				},
				Children: logsChildren,
			})
		}
	}

	// Manage 路由（仅 admin）
	if permissions["manage"] {
		manageChildren := []types.MenuRoute{}

		if permissions["manage_user"] {
			manageChildren = append(manageChildren, types.MenuRoute{
				Name:      "manage_user",
				Path:      "/manage/user",
				Component: "view.manage_user",
				Meta: types.RouteMeta{
					Title:   "manage_user",
					I18nKey: "route.manage_user",
					Icon:    "ic:round-manage-accounts",
					Roles:   []string{"admin"},
				},
			})
		}

		if permissions["manage_role"] {
			manageChildren = append(manageChildren, types.MenuRoute{
				Name:      "manage_role",
				Path:      "/manage/role",
				Component: "view.manage_role",
				Meta: types.RouteMeta{
					Title:   "manage_role",
					I18nKey: "route.manage_role",
					Icon:    "carbon:user-role",
					Roles:   []string{"admin"},
				},
			})
		}

		if len(manageChildren) > 0 {
			routes = append(routes, types.MenuRoute{
				Name:      "manage",
				Path:      "/manage",
				Component: "layout.base",
				Meta: types.RouteMeta{
					Title:   "manage",
					I18nKey: "route.manage",
					Icon:    "carbon:cloud-service-management",
					Order:   9,
					Roles:   []string{"admin"},
				},
				Children: manageChildren,
			})
		}
	}

	return routes
}

// parseMenuMeta 解析菜单元数据
func parseMenuMeta(metaJSON string) types.RouteMeta {
	var meta types.RouteMeta
	json.Unmarshal([]byte(metaJSON), &meta)
	return meta
}
