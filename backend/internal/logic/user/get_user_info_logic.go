package user

import (
	"context"
	"errors"

	"logflux/internal/svc"
	"logflux/internal/types"
	"logflux/model"

	"gorm.io/gorm"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetUserInfoLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetUserInfoLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetUserInfoLogic {
	return &GetUserInfoLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetUserInfoLogic) GetUserInfo(req *types.UserInfoReq) (resp *types.UserInfoResp, err error) {
	// 从 context 中获取 userId (由 JWT 中间件注入)
	userId := l.ctx.Value("userId")
	// userId from jwt is json.Number or float64 depending on parser, safe cast needed
	// Here assuming standard jwt parser which might return float64 or string
	// For simplicity in this demo, we'll try to cast. In production use robust casting.

	var user model.User
	result := l.svcCtx.DB.First(&user, userId)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, errors.New("用户不存在")
		}
		return nil, errors.New("查询用户失败")
	}

	return &types.UserInfoResp{
		UserId:   int64(user.ID),
		Username: user.Username,
		Roles:    user.Roles,
	}, nil
}
