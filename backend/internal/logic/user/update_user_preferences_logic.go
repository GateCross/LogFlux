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

type UpdateUserPreferencesLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewUpdateUserPreferencesLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateUserPreferencesLogic {
	return &UpdateUserPreferencesLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *UpdateUserPreferencesLogic) UpdateUserPreferences(req *types.UserPreferencesReq) (resp *types.BaseResp, err error) {
	// Get userId from context
	userIdVal := l.ctx.Value("userId")
	var uid int64
	if jsonUid, ok := userIdVal.(json.Number); ok {
		if id, err := jsonUid.Int64(); err == nil {
			uid = id
		} else {
			return nil, errors.New("invalid userId format")
		}
	} else if floatUid, ok := userIdVal.(float64); ok {
		uid = int64(floatUid)
	} else if intUid, ok := userIdVal.(int); ok {
		uid = int64(intUid)
	} else {
		return nil, errors.New("invalid userId type")
	}

	// Parse preferences JSON
	var prefs map[string]interface{}
	if err := json.Unmarshal([]byte(req.Preferences), &prefs); err != nil {
		return nil, errors.New("invalid preferences JSON")
	}

	// Update user in database
	var user model.User
	result := l.svcCtx.DB.First(&user, uid)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, errors.New("user not found")
		}
		return nil, errors.New("failed to query user")
	}

	// Update preferences
	user.Preferences = prefs
	if err := l.svcCtx.DB.Save(&user).Error; err != nil {
		return nil, errors.New("failed to update preferences")
	}

	return &types.BaseResp{
		Code: 0,
		Msg:  "success",
	}, nil
}
