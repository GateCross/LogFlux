package user

import (
	"context"

	"logflux/internal/svc"
	"logflux/internal/types"
	"logflux/model"

	"github.com/zeromicro/go-zero/core/logx"
)

type ToggleUserStatusLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewToggleUserStatusLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ToggleUserStatusLogic {
	return &ToggleUserStatusLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ToggleUserStatusLogic) ToggleUserStatus(req *types.ToggleUserStatusReq) (resp *types.BaseResp, err error) {
	// 查找用户
	var user model.User
	if err := l.svcCtx.DB.First(&user, req.ID).Error; err != nil {
		return nil, err
	}

	// 禁止冻结 admin 用户
	for _, role := range user.Roles {
		if role == "admin" {
			return &types.BaseResp{
				Code: 403,
				Msg:  "禁止冻结管理员用户",
			}, nil
		}
	}

	// 切换状态
	newStatus := 1 - user.Status // 0 变 1, 1 变 0
	if err := l.svcCtx.DB.Model(&user).Update("status", newStatus).Error; err != nil {
		return nil, err
	}

	statusText := "启用"
	if newStatus == 0 {
		statusText = "禁用"
	}

	return &types.BaseResp{
		Code: 200,
		Msg:  "用户已" + statusText,
	}, nil
}
