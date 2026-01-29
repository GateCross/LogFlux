package menu

import (
	"context"
	"encoding/json"

	"logflux/internal/svc"
	"logflux/internal/types"
	"logflux/model"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetMenuListLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetMenuListLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetMenuListLogic {
	return &GetMenuListLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetMenuListLogic) GetMenuList() (resp *types.MenuListResp, err error) {
	var allMenus []model.Menu
	// 获取所有菜单，按 Order 排序
	if err := l.svcCtx.DB.Order("\"order\" asc").Find(&allMenus).Error; err != nil {
		return nil, err
	}

	tree := l.buildTree(allMenus, nil)

	return &types.MenuListResp{
		List: tree,
	}, nil
}

func (l *GetMenuListLogic) buildTree(allMenus []model.Menu, parentID *uint) []types.MenuItem {
	var items []types.MenuItem

	for _, m := range allMenus {
		// 检查父节点匹配
		isChild := false
		if parentID == nil {
			if m.ParentID == nil {
				isChild = true
			}
		} else {
			if m.ParentID != nil && *m.ParentID == *parentID {
				isChild = true
			}
		}

		if isChild {
			children := l.buildTree(allMenus, &m.ID)

			// 解析 Meta
			var meta types.RouteMeta
			if m.Meta != "" {
				json.Unmarshal([]byte(m.Meta), &meta)
			}

			item := types.MenuItem{
				ID:            m.ID,
				Name:          m.Name,
				Path:          m.Path,
				Component:     m.Component,
				Order:         m.Order,
				Meta:          meta,
				RequiredRoles: m.RequiredRoles,
				CreatedAt:     m.CreatedAt.Format("2006-01-02 15:04:05"),
			}

			if len(children) > 0 {
				item.Children = children
			}

			items = append(items, item)
		}
	}
	return items
}
