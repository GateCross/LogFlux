package menu

import (
	"context"

	"logflux/common/result"
	"logflux/internal/svc"
	"logflux/internal/types"
	"logflux/model"

	"github.com/lib/pq"
	"github.com/zeromicro/go-zero/core/logx"
)

type UpdateMenuLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewUpdateMenuLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateMenuLogic {
	return &UpdateMenuLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *UpdateMenuLogic) UpdateMenu(req *types.UpdateMenuReq) (resp *types.BaseResp, err error) {
	var menu model.Menu
	if err := l.svcCtx.DB.First(&menu, req.ID).Error; err != nil {
		return nil, result.NewErrMsg("菜单不存在")
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
	if req.Order != 0 {
		updates["order"] = req.Order
	}
	if req.Meta != "" {
		updates["meta"] = req.Meta
	}
	if req.RequiredRoles != nil {
		updates["required_roles"] = pq.StringArray(req.RequiredRoles)
	}

	if req.ParentID > 0 {
		updates["parent_id"] = req.ParentID
	} else {
		updates["parent_id"] = nil // 设为顶级菜单
	}

	if err := l.svcCtx.DB.Model(&menu).Updates(updates).Error; err != nil {
		return nil, result.NewErrMsg("更新失败: " + err.Error())
	}

	return &types.BaseResp{
		Code: 200,
		Msg:  "更新成功",
	}, nil
}
