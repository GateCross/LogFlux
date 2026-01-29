package user

import (
	"context"
	"encoding/json"
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

	// Parse userId
	var uid int64
	if jsonUid, ok := userId.(json.Number); ok {
		if id, err := jsonUid.Int64(); err == nil {
			uid = id
		} else {
			return nil, errors.New("invalid userId format")
		}
	} else if floatUid, ok := userId.(float64); ok {
		uid = int64(floatUid)
	} else if intUid, ok := userId.(int); ok {
		uid = int64(intUid)
	} else {
		return nil, errors.New("invalid userId type")
	}

	var user model.User
	result := l.svcCtx.DB.First(&user, uid)
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
