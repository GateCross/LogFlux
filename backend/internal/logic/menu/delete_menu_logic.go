package menu

import (
	"context"

	"logflux/common/result"
	"logflux/internal/svc"
	"logflux/internal/types"
	"logflux/model"

	"github.com/zeromicro/go-zero/core/logx"
)

type DeleteMenuLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewDeleteMenuLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeleteMenuLogic {
	return &DeleteMenuLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *DeleteMenuLogic) DeleteMenu(req *types.IDReq) (resp *types.BaseResp, err error) {
	// 检查是否有子菜单
	var count int64
	l.svcCtx.DB.Model(&model.Menu{}).Where("parent_id = ?", req.ID).Count(&count)
	if count > 0 {
		return nil, result.NewErrMsg("请先删除子菜单")
	}

	if err := l.svcCtx.DB.Delete(&model.Menu{}, req.ID).Error; err != nil {
		return nil, result.NewErrMsg("删除失败: " + err.Error())
	}

	return &types.BaseResp{
		Code: 200,
		Msg:  "删除成功",
	}, nil
}
