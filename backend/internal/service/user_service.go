package service

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"logflux/internal/svc"
	"logflux/internal/types"
	"logflux/internal/utils/logger"
	"logflux/internal/xerr"
	"logflux/model"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// UserService 负责用户管理业务。
type UserService struct {
	logger.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// NewUserService 创建用户服务。
func NewUserService(ctx context.Context, svcCtx *svc.ServiceContext) *UserService {
	return &UserService{
		Logger: logger.New(logger.ModuleUser).WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (s *UserService) AddUser(req *types.AddUserReq) (*types.BaseResp, error) {
	if _, err := s.svcCtx.UserModel.FindByUsername(s.ctx, req.Username, true); err == nil {
		return nil, xerr.NewBusinessErrorWith("用户名已存在")
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, xerr.NewCodeErrorWithCause(xerr.ServerCommonError, "检查用户失败", err)
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, xerr.NewCodeErrorWithCause(xerr.ServerCommonError, "密码加密失败", err)
	}

	newUser := &model.User{
		Username: strings.TrimSpace(req.Username),
		Password: string(hash),
		Roles:    req.Roles,
	}
	if err := s.svcCtx.UserModel.Create(s.ctx, newUser); err != nil {
		if strings.Contains(err.Error(), "duplicate key") || strings.Contains(err.Error(), "idx_users_username") {
			return nil, xerr.NewBusinessErrorWith("用户名已存在")
		}
		return nil, xerr.NewCodeErrorWithCause(xerr.ServerCommonError, "创建用户失败", err)
	}
	return baseResp("创建成功"), nil
}

func (s *UserService) ChangePassword(req *types.ChangePasswordReq) (*types.BaseResp, error) {
	userID, err := userIDFromContext(s.ctx)
	if err != nil {
		return nil, xerr.NewBusinessErrorWith(err.Error())
	}

	user, err := s.svcCtx.UserModel.FindByID(s.ctx, userID)
	if err != nil {
		return nil, xerr.NewBusinessErrorWith("用户不存在")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.OldPassword)); err != nil {
		return nil, xerr.NewBusinessErrorWith("旧密码错误")
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		return nil, xerr.NewCodeErrorWithCause(xerr.ServerCommonError, "密码加密失败", err)
	}
	if err := s.svcCtx.UserModel.UpdateFields(s.ctx, user, map[string]interface{}{"password": string(hashedPassword)}); err != nil {
		return nil, xerr.NewCodeErrorWithCause(xerr.ServerCommonError, "更新密码失败", err)
	}
	return baseResp("修改成功"), nil
}

func (s *UserService) DeleteUser(req *types.IDReq) (*types.BaseResp, error) {
	user, err := s.svcCtx.UserModel.FindByID(s.ctx, req.ID)
	if err != nil {
		return nil, err
	}

	if err := s.ensureOtherActiveAdmin(user, req.ID); err != nil {
		return nil, err
	}
	if err := s.svcCtx.UserModel.Delete(s.ctx, user); err != nil {
		return nil, xerr.NewCodeErrorWithCause(xerr.ServerCommonError, "删除用户失败", err)
	}
	return baseResp("删除成功"), nil
}

func (s *UserService) GetUserInfo() (*types.UserInfoResp, error) {
	userID, err := userIDFromContext(s.ctx)
	if err != nil {
		return nil, xerr.NewBusinessErrorWith(err.Error())
	}

	user, err := s.svcCtx.UserModel.FindByID(s.ctx, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, xerr.NewBusinessErrorWith("用户不存在")
		}
		return nil, xerr.NewCodeErrorWithCause(xerr.ServerCommonError, "查询用户失败", err)
	}

	var preferences string
	if user.Preferences != nil {
		if data, err := json.Marshal(user.Preferences); err == nil {
			preferences = string(data)
		}
	}

	return &types.UserInfoResp{
		UserId:      int64(user.ID),
		Username:    user.Username,
		Roles:       user.Roles,
		Preferences: preferences,
	}, nil
}

func (s *UserService) GetUserList(req *types.UserListReq) (*types.UserListResp, error) {
	users, total, err := s.svcCtx.UserModel.List(s.ctx, req.Username, req.Page, req.PageSize)
	if err != nil {
		return nil, xerr.NewCodeErrorWithCause(xerr.ServerCommonError, "查询用户列表失败", err)
	}

	list := make([]types.UserItem, 0, len(users))
	for _, user := range users {
		list = append(list, types.UserItem{
			ID:        user.ID,
			Username:  user.Username,
			Roles:     user.Roles,
			Status:    user.Status,
			CreatedAt: user.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}
	return &types.UserListResp{List: list, Total: total}, nil
}

func (s *UserService) ToggleUserStatus(req *types.IDReq) (*types.BaseResp, error) {
	user, err := s.svcCtx.UserModel.FindByID(s.ctx, req.ID)
	if err != nil {
		return nil, err
	}
	if err := s.ensureOtherActiveAdmin(user, req.ID); err != nil {
		return nil, err
	}

	newStatus := 1
	msg := "用户已解冻"
	if user.Status == 1 {
		newStatus = 0
		msg = "用户已冻结"
	}
	if err := s.svcCtx.UserModel.UpdateFields(s.ctx, user, map[string]interface{}{"status": newStatus}); err != nil {
		return nil, xerr.NewCodeErrorWithCause(xerr.ServerCommonError, "更新用户状态失败", err)
	}
	return baseResp(msg), nil
}

func (s *UserService) UpdateUser(req *types.UpdateUserReq) (*types.BaseResp, error) {
	user, err := s.svcCtx.UserModel.FindByID(s.ctx, req.ID)
	if err != nil {
		return nil, err
	}

	updates := map[string]interface{}{}
	if req.Password != "" {
		hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
		if err != nil {
			return nil, xerr.NewCodeErrorWithCause(xerr.ServerCommonError, "密码加密失败", err)
		}
		updates["password"] = string(hash)
	}
	if err := s.svcCtx.UserModel.UpdateFields(s.ctx, user, updates); err != nil {
		return nil, xerr.NewCodeErrorWithCause(xerr.ServerCommonError, "更新用户失败", err)
	}
	if req.Roles != nil {
		user.Roles = req.Roles
		if err := s.svcCtx.UserModel.Save(s.ctx, user); err != nil {
			return nil, xerr.NewCodeErrorWithCause(xerr.ServerCommonError, "更新用户角色失败", err)
		}
	}
	return baseResp("更新成功"), nil
}

func (s *UserService) UpdateUserPreferences(req *types.UserPreferencesReq) (*types.BaseResp, error) {
	userID, err := userIDFromContext(s.ctx)
	if err != nil {
		return nil, xerr.NewBusinessErrorWith(err.Error())
	}

	var preferences map[string]interface{}
	if err := json.Unmarshal([]byte(req.Preferences), &preferences); err != nil {
		return nil, xerr.NewBusinessErrorWith("偏好设置 JSON 格式无效")
	}

	user, err := s.svcCtx.UserModel.FindByID(s.ctx, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, xerr.NewBusinessErrorWith("用户不存在")
		}
		return nil, xerr.NewCodeErrorWithCause(xerr.ServerCommonError, "查询用户失败", err)
	}
	user.Preferences = preferences
	if err := s.svcCtx.UserModel.Save(s.ctx, user); err != nil {
		return nil, xerr.NewCodeErrorWithCause(xerr.ServerCommonError, "更新偏好设置失败", err)
	}
	return baseResp("更新成功"), nil
}

func (s *UserService) ensureOtherActiveAdmin(user *model.User, currentID uint) error {
	if user == nil || user.Status != 1 || !hasRole(user.Roles, "admin") {
		return nil
	}
	activeUsers, err := s.svcCtx.UserModel.FindActiveUsersExcept(s.ctx, currentID)
	if err != nil {
		return xerr.NewCodeErrorWithCause(xerr.ServerCommonError, "检查管理员用户失败", err)
	}
	for _, activeUser := range activeUsers {
		if hasRole(activeUser.Roles, "admin") {
			return nil
		}
	}
	return xerr.NewBusinessErrorWith("至少保留一个启用的管理员用户")
}

func baseResp(message string) *types.BaseResp {
	return &types.BaseResp{Code: xerr.OK, Msg: message}
}
