package user

import (
	"context"

	"logflux/internal/svc"
	"logflux/internal/types"
	"logflux/model"

	"github.com/zeromicro/go-zero/core/logx"
)

type DeleteUserLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewDeleteUserLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeleteUserLogic {
	return &DeleteUserLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *DeleteUserLogic) DeleteUser(req *types.IDReq) (resp *types.BaseResp, err error) {
	// 查找用户
	var user model.User
	if err := l.svcCtx.DB.First(&user, req.ID).Error; err != nil {
		return nil, err
	}

	// 禁止删除 admin 用户
	for _, role := range user.Roles {
		if role == "admin" {
			return &types.BaseResp{
				Code: 403,
				Msg:  "禁止删除管理员用户",
			}, nil
		}
	}

	// 物理删除用户
	if err := l.svcCtx.DB.Delete(&user).Error; err != nil {
		return nil, err
	}

	return &types.BaseResp{
		Code: 200,
		Msg:  "删除成功",
	}, nil
}
