package auth

import (
	"context"
	"errors"
	"logflux/common/result"
	"logflux/internal/svc"
	"logflux/internal/types"
	"logflux/model"

	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/gorm"
)

type RefreshTokenLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewRefreshTokenLogic(ctx context.Context, svcCtx *svc.ServiceContext) *RefreshTokenLogic {
	return &RefreshTokenLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *RefreshTokenLogic) RefreshToken(req *types.RefreshTokenReq) (resp *types.LoginResp, err error) {
	if req.RefreshToken == "" {
		return nil, result.NewCodeError(3000, "刷新令牌不能为空")
	}

	secret := l.svcCtx.Config.Auth.AccessSecret
	claims, err := parseTokenClaims(req.RefreshToken, secret)
	if err != nil {
		return nil, result.NewCodeError(3000, "刷新令牌无效或已过期")
	}

	if getTokenType(claims) != refreshTokenType {
		return nil, result.NewCodeError(3000, "刷新令牌类型错误")
	}

	userId, err := getUserIdFromClaims(claims)
	if err != nil {
		return nil, result.NewCodeError(3000, "刷新令牌无效")
	}

	var user model.User
	query := l.svcCtx.DB.First(&user, userId)
	if query.Error != nil {
		if errors.Is(query.Error, gorm.ErrRecordNotFound) {
			return nil, result.NewCodeError(3000, "用户不存在")
		}

		return nil, result.NewCodeError(500, "查询用户失败")
	}

	if user.Status == 0 {
		return nil, result.NewCodeError(3000, "用户已被禁用")
	}

	accessExpire := l.svcCtx.Config.Auth.AccessExpire
	accessToken, refreshToken, err := buildTokenPair(int64(user.ID), user.Roles, secret, accessExpire)
	if err != nil {
		return nil, result.NewCodeError(500, "刷新令牌失败")
	}

	return &types.LoginResp{
		Token:        accessToken,
		RefreshToken: refreshToken,
	}, nil
}
