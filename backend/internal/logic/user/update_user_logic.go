package user

import (
	"context"

	"logflux/internal/svc"
	"logflux/internal/types"
	"logflux/model"

	"github.com/zeromicro/go-zero/core/logx"
	"golang.org/x/crypto/bcrypt"
)

type UpdateUserLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewUpdateUserLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateUserLogic {
	return &UpdateUserLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *UpdateUserLogic) UpdateUser(req *types.UpdateUserReq) (resp *types.BaseResp, err error) {
	var user model.User
	if err := l.svcCtx.DB.First(&user, req.ID).Error; err != nil {
		return nil, err
	}

	updates := map[string]interface{}{}
	if req.Password != "" {
		hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
		if err != nil {
			return nil, err
		}
		updates["password"] = string(hash)
	}
	if req.Roles != nil {
		// GORM might need special handling for array types or we update the struct field
		user.Roles = req.Roles
		// For simple updates map is safer for partial, but roles is struct field.
		// Let's save the model directly for roles.
	}

	if err := l.svcCtx.DB.Model(&user).Updates(updates).Error; err != nil {
		return nil, err
	}

	// Explicitly save roles if changed (Updates with map ignores zero values and complex types depending on driver)
	if req.Roles != nil {
		user.Roles = req.Roles
		if err := l.svcCtx.DB.Save(&user).Error; err != nil {
			return nil, err
		}
	}

	return &types.BaseResp{
		Code: 200,
		Msg:  "success",
	}, nil
}
