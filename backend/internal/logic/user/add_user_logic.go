package user

import (
	"context"
	"errors"

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
	// Check if user exists
	var existing model.User
	if err := l.svcCtx.DB.Where("username = ?", req.Username).First(&existing).Error; err == nil {
		return nil, errors.New("username already exists")
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
		return nil, err
	}

	return &types.BaseResp{
		Code: 200,
		Msg:  "success",
	}, nil
}
