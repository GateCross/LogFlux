package model

import (
	"context"
	"strings"

	"gorm.io/gorm"
)

// UserModel 定义用户表的数据访问能力。
type UserModel interface {
	FindByID(ctx context.Context, id uint) (*User, error)
	FindByUsername(ctx context.Context, username string, includeDeleted bool) (*User, error)
	List(ctx context.Context, username string, page, pageSize int) ([]User, int64, error)
	Create(ctx context.Context, user *User) error
	UpdateFields(ctx context.Context, user *User, updates map[string]interface{}) error
	Save(ctx context.Context, user *User) error
	Delete(ctx context.Context, user *User) error
	FindActiveUsersExcept(ctx context.Context, exceptID uint) ([]User, error)
}

type defaultUserModel struct {
	db *gorm.DB
}

// NewUserModel 创建用户模型。
func NewUserModel(db *gorm.DB) UserModel {
	return &defaultUserModel{db: db}
}

func (m *defaultUserModel) conn(ctx context.Context) *gorm.DB {
	if ctx == nil {
		ctx = context.Background()
	}
	return m.db.WithContext(ctx)
}

func (m *defaultUserModel) FindByID(ctx context.Context, id uint) (*User, error) {
	var user User
	if err := m.conn(ctx).First(&user, id).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

func (m *defaultUserModel) FindByUsername(ctx context.Context, username string, includeDeleted bool) (*User, error) {
	var user User
	db := m.conn(ctx)
	if includeDeleted {
		db = db.Unscoped()
	}
	if err := db.Where("username = ?", strings.TrimSpace(username)).First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

func (m *defaultUserModel) List(ctx context.Context, username string, page, pageSize int) ([]User, int64, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}

	db := m.conn(ctx).Model(&User{})
	if keyword := strings.TrimSpace(username); keyword != "" {
		db = db.Where("username LIKE ?", "%"+keyword+"%")
	}

	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var users []User
	offset := (page - 1) * pageSize
	if err := db.Limit(pageSize).Offset(offset).Find(&users).Error; err != nil {
		return nil, 0, err
	}
	return users, total, nil
}

func (m *defaultUserModel) Create(ctx context.Context, user *User) error {
	return m.conn(ctx).Create(user).Error
}

func (m *defaultUserModel) UpdateFields(ctx context.Context, user *User, updates map[string]interface{}) error {
	if len(updates) == 0 {
		return nil
	}
	return m.conn(ctx).Model(user).Updates(updates).Error
}

func (m *defaultUserModel) Save(ctx context.Context, user *User) error {
	return m.conn(ctx).Save(user).Error
}

func (m *defaultUserModel) Delete(ctx context.Context, user *User) error {
	return m.conn(ctx).Delete(user).Error
}

func (m *defaultUserModel) FindActiveUsersExcept(ctx context.Context, exceptID uint) ([]User, error) {
	var users []User
	if err := m.conn(ctx).Select("roles").Where("status = ? AND id <> ?", 1, exceptID).Find(&users).Error; err != nil {
		return nil, err
	}
	return users, nil
}
