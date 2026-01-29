package user

import (
	"context"

	"logflux/common/result"
	"logflux/internal/svc"
	"logflux/internal/types"
	"logflux/model"

	"github.com/zeromicro/go-zero/core/logx"
	"golang.org/x/crypto/bcrypt"
)

type ChangePasswordLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewChangePasswordLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ChangePasswordLogic {
	return &ChangePasswordLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ChangePasswordLogic) ChangePassword(req *types.ChangePasswordReq) (resp *types.BaseResp, err error) {
	userId := l.ctx.Value("userId")
	if userId == nil {
		return nil, result.NewErrMsg("未认证")
	}

	var u model.User
	if err := l.svcCtx.DB.First(&u, userId).Error; err != nil {
		return nil, result.NewErrMsg("用户不存在")
	}

	// 验证旧密码
	if err := bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(req.OldPassword)); err != nil {
		return nil, result.NewErrMsg("旧密码错误")
	}

	// 加密新密码
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		return nil, result.NewErrMsg("密码加密失败")
	}

	// 更新密码
	if err := l.svcCtx.DB.Model(&u).Update("password", string(hashedPassword)).Error; err != nil {
		return nil, result.NewErrMsg("更新失败")
	}

	return &types.BaseResp{
		Code: 200,
		Msg:  "修改成功",
	}, nil
}
