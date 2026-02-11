package auth

import (
	"context"
	"errors"
	"logflux/common/cryptx"
	"logflux/internal/svc"
	"logflux/internal/types"
	"logflux/model"

	"github.com/zeromicro/go-zero/core/logx"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type LoginLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewLoginLogic(ctx context.Context, svcCtx *svc.ServiceContext) *LoginLogic {
	return &LoginLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *LoginLogic) Login(req *types.LoginReq) (resp *types.LoginResp, err error) {
	var user model.User
	result := l.svcCtx.DB.Where("username = ?", req.Username).First(&user)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, errors.New("用户不存在")
		}

		return nil, errors.New("查询用户失败")
	}

	password, err := cryptx.Decrypt(req.Password, l.svcCtx.Config.Auth.AESKey)
	if err != nil {
		return nil, errors.New("用户名或密码错误")
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password))
	if err != nil {
		return nil, errors.New("用户名或密码错误")
	}

	accessSecret := l.svcCtx.Config.Auth.AccessSecret
	accessExpire := l.svcCtx.Config.Auth.AccessExpire
	accessToken, refreshToken, err := buildTokenPair(int64(user.ID), user.Roles, accessSecret, accessExpire)
	if err != nil {
		return nil, err
	}

	return &types.LoginResp{
		Token:        accessToken,
		RefreshToken: refreshToken,
	}, nil
}
