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

type CreateMenuLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewCreateMenuLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CreateMenuLogic {
	return &CreateMenuLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *CreateMenuLogic) CreateMenu(req *types.CreateMenuReq) (resp *types.BaseResp, err error) {
	// 检查名称是否存在
	var count int64
	l.svcCtx.DB.Model(&model.Menu{}).Where("name = ?", req.Name).Count(&count)
	if count > 0 {
		return nil, result.NewErrMsg("菜单标识已存在")
	}

	metaBytes, _ := json.Marshal(req.Meta)
	menu := model.Menu{
		Name:          req.Name,
		Path:          req.Path,
		Component:     req.Component,
		Order:         req.Order,
		Meta:          string(metaBytes),
		RequiredRoles: pq.StringArray(req.RequiredRoles),
	}

	if req.ParentID > 0 {
		pid := req.ParentID
		menu.ParentID = &pid
	}

	if err := l.svcCtx.DB.Create(&menu).Error; err != nil {
		return nil, result.NewErrMsg("创建失败: " + err.Error())
	}

	return &types.BaseResp{
		Code: 200,
		Msg:  "创建成功",
	}, nil
}
