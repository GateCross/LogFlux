package role

import (
	"context"

	"logflux/internal/svc"
	"logflux/internal/types"
	"logflux/model"

	"github.com/lib/pq"
	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/gorm"
)

type UpdateRolePermissionsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewUpdateRolePermissionsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateRolePermissionsLogic {
	return &UpdateRolePermissionsLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *UpdateRolePermissionsLogic) UpdateRolePermissions(req *types.UpdateRolePermissionsReq) (resp *types.BaseResp, err error) {
	var role model.Role
	// 检查角色是否存在
	if err := l.svcCtx.DB.First(&role, req.ID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return &types.BaseResp{
				Code: 404,
				Msg:  "Role not found",
			}, nil
		}
		return nil, err
	}

	// 必须转换为 pq.StringArray
	permissions := pq.StringArray(req.Permissions)

	// 更新权限
	if err := l.svcCtx.DB.Model(&role).Update("permissions", permissions).Error; err != nil {
		l.Logger.Errorf("Failed to update role permissions: %v", err)
		return nil, err
	}

	return &types.BaseResp{
		Code: 200,
		Msg:  "Success",
	}, nil
}
