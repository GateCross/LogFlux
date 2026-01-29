package menu

import (
	"context"
	"encoding/json"

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

	// 序列化 Meta
	metaJSON, _ := json.Marshal(req.Meta)

	updates := map[string]interface{}{
		"name":           req.Name,
		"path":           req.Path,
		"component":      req.Component,
		"order":          req.Order,
		"meta":           string(metaJSON),
		"required_roles": pq.StringArray(req.RequiredRoles),
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
