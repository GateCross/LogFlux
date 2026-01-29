package route

import (
	"context"
	"encoding/json"
	"errors"

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

	// Parse userId
	var uid int64
	if jsonUid, ok := userId.(json.Number); ok {
		if id, err := jsonUid.Int64(); err == nil {
			uid = id
		} else {
			return nil, errors.New("invalid userId format")
		}
	} else if floatUid, ok := userId.(float64); ok {
		uid = int64(floatUid)
	} else if intUid, ok := userId.(int); ok {
		uid = int64(intUid)
	} else {
		return nil, errors.New("invalid userId type")
	}

	var user model.User
	if err := l.svcCtx.DB.First(&user, uid).Error; err != nil {
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
	routes := l.buildRoutesFromDB(userPermissions, roles)

	// 调试日志
	// l.Logger.Infof("User %v Permissions: %v, Roles: %v, Routes Count: %d", userId, userPermissions, user.Roles, len(routes))

	return &types.UserRouteResp{
		Home:   "dashboard",
		Routes: routes,
	}, nil
}

// buildRoutesFromDB 从数据库构建路由树
func (l *GetUserRoutesLogic) buildRoutesFromDB(permissions map[string]bool, userRoles []model.Role) []types.MenuRoute {
	var allMenus []model.Menu
	// 获取所有菜单，按 Order 排序
	l.svcCtx.DB.Order("\"order\" asc").Find(&allMenus)

	// 重新构建：使用递归方法
	// 让我们使用 ID 索引所有原始 model，然后递归构建。

	return l.buildTree(allMenus, nil, userRoles)
}

func (l *GetUserRoutesLogic) buildTree(allMenus []model.Menu, parentID *uint, userRoles []model.Role) []types.MenuRoute {
	var routes []types.MenuRoute

	for _, m := range allMenus {
		// 检查父节点匹配
		isMatch := false
		if parentID == nil {
			if m.ParentID == nil {
				isMatch = true
			}
		} else {
			if m.ParentID != nil && *m.ParentID == *parentID {
				isMatch = true
			}
		}

		if isMatch {
			// 权限检查
			if !l.hasPermission(m, userRoles) {
				continue
			}

			children := l.buildTree(allMenus, &m.ID, userRoles)

			route := types.MenuRoute{
				Name:      m.Name,
				Path:      m.Path,
				Component: m.Component,
				Meta:      l.parseMenuMeta(m.Meta),
			}

			if len(children) > 0 {
				route.Children = children
			}

			routes = append(routes, route)
		}
	}
	return routes
}

// hasPermission 检查用户是否拥有菜单所需的角色
func (l *GetUserRoutesLogic) hasPermission(menu model.Menu, userRoles []model.Role) bool {
	// 如果没有定义 RequiredRoles，则默认允许访问 (public)
	if len(menu.RequiredRoles) == 0 {
		return true
	}

	// 检查用户角色是否在 RequiredRoles 中
	for _, userRole := range userRoles {
		for _, requiredRole := range menu.RequiredRoles {
			if userRole.Name == requiredRole {
				return true
			}
		}
	}

	return false
}

// parseMenuMeta 解析菜单元数据
func (l *GetUserRoutesLogic) parseMenuMeta(metaJSON string) types.RouteMeta {
	var meta types.RouteMeta
	if metaJSON == "" {
		return meta
	}
	json.Unmarshal([]byte(metaJSON), &meta)
	return meta
}
