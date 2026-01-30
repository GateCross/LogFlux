package user

import (
	"context"

	"logflux/internal/svc"
	"logflux/internal/types"
	"logflux/model"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetUserListLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetUserListLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetUserListLogic {
	return &GetUserListLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetUserListLogic) GetUserList(req *types.UserListReq) (resp *types.UserListResp, err error) {
	var users []model.User
	var total int64

	db := l.svcCtx.DB.Model(&model.User{})

	if req.Username != "" {
		db = db.Where("username LIKE ?", "%"+req.Username+"%")
	}

	if err := db.Count(&total).Error; err != nil {
		l.Error("Count error: ", err)
		return nil, err
	}

	offset := (req.Page - 1) * req.PageSize
	if err := db.Limit(req.PageSize).Offset(offset).Find(&users).Error; err != nil {
		l.Error("Find error: ", err)
		return nil, err
	}

	l.Infof("Found %d users, total: %d", len(users), total)

	var list []types.UserItem
	for _, u := range users {
		list = append(list, types.UserItem{
			ID:        u.ID,
			Username:  u.Username,
			Roles:     u.Roles,
			Status:    u.Status,
			CreatedAt: u.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}

	return &types.UserListResp{
		List:  list,
		Total: total,
	}, nil
}
