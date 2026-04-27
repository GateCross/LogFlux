package service

import (
	"context"
	"errors"

	"logflux/internal/svc"
	"logflux/internal/types"
	"logflux/internal/utils/logger"
	"logflux/internal/xerr"

	"github.com/lib/pq"
	"gorm.io/gorm"
)

// RoleService 负责角色管理业务。
type RoleService struct {
	logger.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// NewRoleService 创建角色服务。
func NewRoleService(ctx context.Context, svcCtx *svc.ServiceContext) *RoleService {
	return &RoleService{
		Logger: logger.New(logger.ModuleUser).WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (s *RoleService) GetRoleList() (*types.RoleListResp, error) {
	roles, err := s.svcCtx.RoleModel.FindAll(s.ctx)
	if err != nil {
		return nil, xerr.NewCodeErrorWithCause(xerr.ServerCommonError, "查询角色列表失败", err)
	}

	list := make([]types.RoleItem, 0, len(roles))
	for _, role := range roles {
		list = append(list, types.RoleItem{
			ID:          role.ID,
			Name:        role.Name,
			DisplayName: role.DisplayName,
			Description: role.Description,
			Permissions: role.Permissions,
			CreatedAt:   role.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}
	return &types.RoleListResp{List: list}, nil
}

func (s *RoleService) UpdateRolePermissions(req *types.UpdateRolePermissionsReq) (*types.BaseResp, error) {
	role, err := s.svcCtx.RoleModel.FindByID(s.ctx, req.ID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, xerr.NewBusinessErrorWith("角色不存在")
		}
		return nil, xerr.NewCodeErrorWithCause(xerr.ServerCommonError, "查询角色失败", err)
	}
	if err := s.svcCtx.RoleModel.UpdatePermissions(s.ctx, role, pq.StringArray(req.Permissions)); err != nil {
		return nil, xerr.NewCodeErrorWithCause(xerr.ServerCommonError, "更新角色权限失败", err)
	}
	return baseResp("更新成功"), nil
}
