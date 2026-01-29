package auth

import (
	"context"
	"errors"
	"logflux/common/cryptx"

	"logflux/internal/svc"
	"logflux/internal/types"
	"logflux/model"

	"time"

	"github.com/zeromicro/go-zero/core/logx"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"github.com/golang-jwt/jwt/v4"
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

	// 密码解密
	password, err := cryptx.Decrypt(req.Password, l.svcCtx.Config.Auth.AESKey)
	if err != nil {
		// l.Logger.Errorf("Decrypt error: %v, password: %s", err, req.Password)
		return nil, errors.New("用户名或密码错误")
	}
	l.Logger.Infof("Decrypted password: %s", password)

	// 密码验证
	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password))
	if err != nil {
		return nil, errors.New("用户名或密码错误")
	}

	// 生成 JWT Token
	now := time.Now().Unix()
	accessSecret := l.svcCtx.Config.Auth.AccessSecret
	accessExpire := l.svcCtx.Config.Auth.AccessExpire

	claims := make(jwt.MapClaims)
	claims["exp"] = now + accessExpire
	claims["iat"] = now
	claims["userId"] = user.ID
	claims["role"] = "user" // Default role
	if len(user.Roles) > 0 {
		claims["role"] = user.Roles[0] // Use first role as primary
	}

	token := jwt.New(jwt.SigningMethodHS256)
	token.Claims = claims

	accessToken, err := token.SignedString([]byte(accessSecret))
	if err != nil {
		return nil, err
	}

	return &types.LoginResp{
		Token:        accessToken,
		RefreshToken: accessToken, // TODO: Implement real refresh token
	}, nil
}
