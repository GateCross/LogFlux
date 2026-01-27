package route

import (
	"context"

	"logflux/internal/svc"
	"logflux/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetConstantRoutesLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetConstantRoutesLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetConstantRoutesLogic {
	return &GetConstantRoutesLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetConstantRoutesLogic) GetConstantRoutes() (resp []types.MenuRoute, err error) {
	routes := []types.MenuRoute{
		{
			Name:      "403",
			Path:      "/403",
			Component: "layout.blank$view.403",
			Meta: types.RouteMeta{
				Title:      "403",
				I18nKey:    "route.403",
				HideInMenu: true,
			},
		},
		{
			Name:      "404",
			Path:      "/404",
			Component: "layout.blank$view.404",
			Meta: types.RouteMeta{
				Title:      "404",
				I18nKey:    "route.404",
				HideInMenu: true,
			},
		},
		{
			Name:      "500",
			Path:      "/500",
			Component: "layout.blank$view.500",
			Meta: types.RouteMeta{
				Title:      "500",
				I18nKey:    "route.500",
				HideInMenu: true,
			},
		},
		// login is usually handled separately or via static import in some versions,
		// but providing it here ensures it exists in dynamic map using blank layout
	}

	return routes, nil
}
