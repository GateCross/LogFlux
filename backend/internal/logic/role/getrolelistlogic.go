package role

import (
	"context"

	"logflux/internal/svc"
	"logflux/internal/types"
	"logflux/model"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetRoleListLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetRoleListLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetRoleListLogic {
	return &GetRoleListLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetRoleListLogic) GetRoleList() (resp *types.RoleListResp, err error) {
	var roles []model.Role
	// 查询所有角色，按 ID 排序
	result := l.svcCtx.DB.Order("id asc").Find(&roles)
	if result.Error != nil {
		l.Logger.Errorf("Failed to query role list: %v", result.Error)
		return nil, result.Error
	}

	roleList := make([]types.RoleItem, 0, len(roles))
	for _, role := range roles {
		roleList = append(roleList, types.RoleItem{
			ID:          role.ID,
			Name:        role.Name,
			DisplayName: role.DisplayName,
			Description: role.Description,
			Permissions: role.Permissions, // pq.StringArray 可以直接赋值给 []string
			CreatedAt:   role.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}

	return &types.RoleListResp{
		List: roleList,
	}, nil
}
