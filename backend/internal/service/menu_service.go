package service

import (
	"context"
	"encoding/json"

	"logflux/internal/svc"
	"logflux/internal/types"
	"logflux/internal/utils/logger"
	"logflux/internal/xerr"
	"logflux/model"

	"github.com/lib/pq"
)

// MenuService 负责菜单管理业务。
type MenuService struct {
	logger.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// NewMenuService 创建菜单服务。
func NewMenuService(ctx context.Context, svcCtx *svc.ServiceContext) *MenuService {
	return &MenuService{
		Logger: logger.New(logger.ModuleSystem).WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (s *MenuService) CreateMenu(req *types.CreateMenuReq) (*types.BaseResp, error) {
	count, err := s.svcCtx.MenuModel.CountByName(s.ctx, req.Name)
	if err != nil {
		return nil, xerr.NewCodeErrorWithCause(xerr.ServerCommonError, "检查菜单失败", err)
	}
	if count > 0 {
		return nil, xerr.NewBusinessErrorWith("菜单标识已存在")
	}

	metaBytes, _ := json.Marshal(req.Meta)
	menu := &model.Menu{
		Name:          req.Name,
		Path:          req.Path,
		Component:     req.Component,
		Order:         req.Order,
		Meta:          string(metaBytes),
		RequiredRoles: pq.StringArray(req.RequiredRoles),
	}
	if req.ParentID > 0 {
		parentID := req.ParentID
		menu.ParentID = &parentID
	}
	if err := s.svcCtx.MenuModel.Create(s.ctx, menu); err != nil {
		return nil, xerr.NewCodeErrorWithCause(xerr.ServerCommonError, "创建菜单失败", err)
	}
	return baseResp("创建成功"), nil
}

func (s *MenuService) DeleteMenu(req *types.IDReq) (*types.BaseResp, error) {
	count, err := s.svcCtx.MenuModel.CountChildren(s.ctx, req.ID)
	if err != nil {
		return nil, xerr.NewCodeErrorWithCause(xerr.ServerCommonError, "检查子菜单失败", err)
	}
	if count > 0 {
		return nil, xerr.NewBusinessErrorWith("存在子菜单，无法删除")
	}
	if err := s.svcCtx.MenuModel.DeleteByID(s.ctx, req.ID); err != nil {
		return nil, xerr.NewCodeErrorWithCause(xerr.ServerCommonError, "删除菜单失败", err)
	}
	return baseResp("删除成功"), nil
}

func (s *MenuService) GetMenuList() (*types.MenuListResp, error) {
	menus, err := s.svcCtx.MenuModel.FindAll(s.ctx)
	if err != nil {
		return nil, xerr.NewCodeErrorWithCause(xerr.ServerCommonError, "查询菜单失败", err)
	}
	return &types.MenuListResp{List: buildMenuTree(menus, nil)}, nil
}

func (s *MenuService) UpdateMenu(req *types.UpdateMenuReq) (*types.BaseResp, error) {
	menu, err := s.svcCtx.MenuModel.FindByID(s.ctx, req.ID)
	if err != nil {
		return nil, xerr.NewBusinessErrorWith("菜单不存在")
	}

	updates := map[string]interface{}{}
	if req.Name != "" {
		updates["name"] = req.Name
	}
	if req.Path != "" {
		updates["path"] = req.Path
	}
	if req.Component != "" {
		updates["component"] = req.Component
	}
	updates["order"] = req.Order
	if req.Meta.Title != "" {
		metaBytes, _ := json.Marshal(req.Meta)
		updates["meta"] = string(metaBytes)
	}
	if req.RequiredRoles != nil {
		updates["required_roles"] = pq.StringArray(req.RequiredRoles)
	}
	if req.ParentID > 0 {
		updates["parent_id"] = req.ParentID
	} else {
		updates["parent_id"] = nil
	}

	if err := s.svcCtx.MenuModel.UpdateFields(s.ctx, menu, updates); err != nil {
		return nil, xerr.NewCodeErrorWithCause(xerr.ServerCommonError, "更新菜单失败", err)
	}
	return baseResp("更新成功"), nil
}

func buildMenuTree(allMenus []model.Menu, parentID *uint) []types.MenuItem {
	items := make([]types.MenuItem, 0)
	for _, menu := range allMenus {
		if !matchMenuParent(menu, parentID) {
			continue
		}

		var meta types.RouteMeta
		if menu.Meta != "" {
			_ = json.Unmarshal([]byte(menu.Meta), &meta)
		}

		item := types.MenuItem{
			ID:            menu.ID,
			Name:          menu.Name,
			Path:          menu.Path,
			Component:     menu.Component,
			Order:         menu.Order,
			Meta:          meta,
			RequiredRoles: menu.RequiredRoles,
			CreatedAt:     menu.CreatedAt.Format("2006-01-02 15:04:05"),
		}
		if item.Order == 0 && meta.Order != 0 {
			item.Order = meta.Order
		}
		if menu.ParentID != nil {
			item.ParentID = *menu.ParentID
		}
		if children := buildMenuTree(allMenus, &menu.ID); len(children) > 0 {
			item.Children = children
		}
		items = append(items, item)
	}
	return items
}

func matchMenuParent(menu model.Menu, parentID *uint) bool {
	if parentID == nil {
		return menu.ParentID == nil
	}
	return menu.ParentID != nil && *menu.ParentID == *parentID
}
