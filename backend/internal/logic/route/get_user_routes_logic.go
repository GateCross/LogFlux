package route

import (
	"context"

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
	// 暂时硬编码路由，后续可从数据库加载
	routes := []types.MenuRoute{
		{
			Name:      "dashboard",
			Path:      "/dashboard",
			Component: "layout.base",
			Meta: types.RouteMeta{
				Title: "dashboard",
				Icon:  "mdi:monitor-dashboard",
				Order: 1,
			},
			Children: []types.MenuRoute{
				{
					Name:      "dashboard_analysis",
					Path:      "/dashboard/analysis",
					Component: "view.dashboard_analysis",
					Meta: types.RouteMeta{
						Title: "analysis",
						Icon:  "icon-park-outline:analysis",
					},
				},
				{
					Name:      "dashboard_workbench",
					Path:      "/dashboard/workbench",
					Component: "view.dashboard_workbench",
					Meta: types.RouteMeta{
						Title: "workbench",
						Icon:  "icon-park-outline:workbench",
					},
				},
			},
		},
		{
			Name:      "manage",
			Path:      "/manage",
			Component: "layout.base",
			Meta: types.RouteMeta{
				Title: "manage",
				Icon:  "carbon:cloud-service-management",
				Order: 9,
				Roles: []string{"admin"}, // 仅 admin 可见
			},
			Children: []types.MenuRoute{
				{
					Name:      "manage_user",
					Path:      "/manage/user",
					Component: "view.manage_user",
					Meta: types.RouteMeta{
						Title: "user",
						Icon:  "ic:round-manage-accounts",
						Roles: []string{"admin"},
					},
				},
				// Add log source management here if needed, keeping it simple for now
			},
		},
	}

	return &types.UserRouteResp{
		Home:   "dashboard_analysis",
		Routes: routes,
	}, nil
}
