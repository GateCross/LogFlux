package svc

import (
	"crypto/rand"
	"fmt"
	"logflux/model"
	"math/big"
	"strings"

	"github.com/lib/pq"
	"github.com/zeromicro/go-zero/core/logx"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

const (
	defaultAdminUsername = "admin"
	defaultPasswordLen   = 12
	minPasswordLen       = 6
	maxPasswordLen       = 18
)

func ensureAdminUser(db *gorm.DB) {
	var admin model.User
	err := db.Where("username = ?", defaultAdminUsername).First(&admin).Error
	if err == nil {
		ensureAdminPasswordHashed(db, &admin)
		ensureAdminRole(db, &admin)
		return
	}

	if err != gorm.ErrRecordNotFound {
		logx.Errorf("初始化管理员账号失败: %v", err)
		return
	}

	var adminCount int64
	if countErr := db.Model(&model.User{}).Where("? = ANY(roles)", "admin").Count(&adminCount).Error; countErr != nil {
		logx.Errorf("检查管理员账号数量失败: %v", countErr)
		return
	}

	if adminCount > 0 {
		logx.Infof("检测到现有管理员账号，跳过默认管理员初始化")
		return
	}

	plainPassword, err := generateComplexPassword(defaultPasswordLen)
	if err != nil {
		logx.Errorf("生成管理员随机密码失败: %v", err)
		return
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(plainPassword), bcrypt.DefaultCost)
	if err != nil {
		logx.Errorf("加密管理员密码失败: %v", err)
		return
	}

	newAdmin := model.User{
		Username: defaultAdminUsername,
		Password: string(hashedPassword),
		Roles:    pq.StringArray{"admin"},
	}

	if err := db.Create(&newAdmin).Error; err != nil {
		logx.Errorf("创建默认管理员账号失败: %v", err)
		return
	}

	logx.Infof("默认管理员账号已初始化: username=%s", defaultAdminUsername)
	logx.Infof("默认管理员初始密码(仅显示一次): %s", plainPassword)
}

func ensureAdminPasswordHashed(db *gorm.DB, admin *model.User) {
	if _, err := bcrypt.Cost([]byte(admin.Password)); err == nil {
		return
	}

	plainPassword, err := generateComplexPassword(defaultPasswordLen)
	if err != nil {
		logx.Errorf("修复管理员密码时生成随机密码失败: %v", err)
		return
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(plainPassword), bcrypt.DefaultCost)
	if err != nil {
		logx.Errorf("修复管理员密码时加密失败: %v", err)
		return
	}

	if err := db.Model(admin).Update("password", string(hashedPassword)).Error; err != nil {
		logx.Errorf("修复管理员密码失败: %v", err)
		return
	}
	admin.Password = string(hashedPassword)

	logx.Infof("检测到管理员密码为非加密存储，已自动重置")
	logx.Infof("管理员新初始密码(仅显示一次): %s", plainPassword)
}

func ensureAdminRole(db *gorm.DB, admin *model.User) {
	for _, role := range admin.Roles {
		if role == "admin" {
			return
		}
	}

	updatedRoles := append(admin.Roles, "admin")
	if err := db.Model(admin).Update("roles", pq.StringArray(updatedRoles)).Error; err != nil {
		logx.Errorf("修复管理员角色失败: %v", err)
		return
	}
	admin.Roles = pq.StringArray(updatedRoles)
}

func generateComplexPassword(length int) (string, error) {
	if length < minPasswordLen {
		length = minPasswordLen
	}
	if length > maxPasswordLen {
		length = maxPasswordLen
	}

	letterChars := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	digitChars := "0123456789"
	underscoreChars := "_"
	allChars := letterChars + digitChars + underscoreChars

	passwordChars := make([]byte, 0, length)
	requiredSets := []string{letterChars, digitChars, underscoreChars}

	for _, charset := range requiredSets {
		char, err := randomCharFrom(charset)
		if err != nil {
			return "", err
		}
		passwordChars = append(passwordChars, char)
	}

	for len(passwordChars) < length {
		char, err := randomCharFrom(allChars)
		if err != nil {
			return "", err
		}
		passwordChars = append(passwordChars, char)
	}

	if err := secureShuffle(passwordChars); err != nil {
		return "", err
	}

	password := string(passwordChars)
	if !validateComplexity(password) {
		return "", fmt.Errorf("generated password does not meet complexity")
	}

	return password, nil
}

func randomCharFrom(charset string) (byte, error) {
	max := big.NewInt(int64(len(charset)))
	idx, err := rand.Int(rand.Reader, max)
	if err != nil {
		return 0, err
	}
	return charset[idx.Int64()], nil
}

func secureShuffle(data []byte) error {
	for index := len(data) - 1; index > 0; index-- {
		randomIndex, err := rand.Int(rand.Reader, big.NewInt(int64(index+1)))
		if err != nil {
			return err
		}
		swapIndex := int(randomIndex.Int64())
		data[index], data[swapIndex] = data[swapIndex], data[index]
	}
	return nil
}

func validateComplexity(password string) bool {
	if len(password) < minPasswordLen || len(password) > maxPasswordLen {
		return false
	}

	hasLetter := strings.IndexAny(password, "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ") >= 0
	hasDigit := strings.IndexAny(password, "0123456789") >= 0
	hasUnderscore := strings.Contains(password, "_")

	for _, char := range password {
		if (char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z') || (char >= '0' && char <= '9') || char == '_' {
			continue
		}
		return false
	}

	return hasLetter && hasDigit && hasUnderscore
}
