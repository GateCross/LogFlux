package user

import (
	"context"

	"logflux/common/result"
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

func (l *ToggleUserStatusLogic) ToggleUserStatus(req *types.IDReq) (resp *types.BaseResp, err error) {
	var user model.User
	if err := l.svcCtx.DB.First(&user, req.ID).Error; err != nil {
		return nil, err
	}

	if user.Status == 1 && hasRole(user.Roles, "admin") {
		var activeUsers []model.User
		if err := l.svcCtx.DB.Select("roles").Where("status = ? AND id <> ?", 1, req.ID).Find(&activeUsers).Error; err != nil {
			return nil, err
		}

		hasOtherAdmin := false
		for _, activeUser := range activeUsers {
			if hasRole(activeUser.Roles, "admin") {
				hasOtherAdmin = true
				break
			}
		}

		if !hasOtherAdmin {
			return nil, result.NewErrMsg("至少保留一个启用的管理员用户")
		}
	}

	newStatus := 1
	msg := "用户已解冻"
	if user.Status == 1 {
		newStatus = 0
		msg = "用户已冻结"
	}

	if err := l.svcCtx.DB.Model(&user).Update("status", newStatus).Error; err != nil {
		return nil, err
	}

	return &types.BaseResp{
		Code: 200,
		Msg:  msg,
	}, nil
}

func hasRole(roles []string, target string) bool {
	for _, role := range roles {
		if role == target {
			return true
		}
	}

	return false
}
