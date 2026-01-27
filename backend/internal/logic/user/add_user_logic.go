package user

import (
	"context"
	"errors"
	"strings"

	"logflux/internal/svc"
	"logflux/internal/types"
	"logflux/model"

	"github.com/zeromicro/go-zero/core/logx"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type AddUserLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewAddUserLogic(ctx context.Context, svcCtx *svc.ServiceContext) *AddUserLogic {
	return &AddUserLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *AddUserLogic) AddUser(req *types.AddUserReq) (resp *types.BaseResp, err error) {
	// Check if user exists (including soft-deleted users)
	var existing model.User
	if err := l.svcCtx.DB.Unscoped().Where("username = ?", req.Username).First(&existing).Error; err == nil {
		return &types.BaseResp{
			Code: 400,
			Msg:  "用户名已存在",
		}, nil
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	// Hash password
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	newUser := model.User{
		Username: req.Username,
		Password: string(hash),
		Roles:    req.Roles,
	}

	if err := l.svcCtx.DB.Create(&newUser).Error; err != nil {
		l.Logger.Errorf("Failed to create user: %v", err)
		// Check if it's a unique constraint violation
		if strings.Contains(err.Error(), "duplicate key") || strings.Contains(err.Error(), "idx_users_username") {
			return &types.BaseResp{
				Code: 400,
				Msg:  "用户名已存在",
			}, nil
		}
		return nil, err
	}

	return &types.BaseResp{
		Code: 200,
		Msg:  "创建成功",
	}, nil
}
