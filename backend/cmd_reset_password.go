package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"logflux/common/password"
	"logflux/internal/config"
	usermodel "logflux/model/user"

	"github.com/zeromicro/go-zero/core/conf"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// runResetPassword 实现 `reset-password` 子命令：为指定账号生成新的随机密码并打印一次。
// 这是忘记密码时唯一的恢复路径，不依赖 Web 登录态。
func runResetPassword(args []string) int {
	fs := flag.NewFlagSet("reset-password", flag.ExitOnError)
	configFile := fs.String("f", "etc/config.yaml", "配置文件路径")
	username := fs.String("u", "admin", "需要重置密码的用户名")
	length := fs.Int("len", password.DefaultLength, "随机密码长度")
	if err := fs.Parse(args); err != nil {
		return 1
	}

	var c config.Config
	if err := conf.Load(*configFile, &c); err != nil {
		fmt.Fprintf(os.Stderr, "读取配置失败: %v\n", err)
		return 1
	}

	db, err := gorm.Open(postgres.Open(c.Database.DSN()), &gorm.Config{})
	if err != nil {
		fmt.Fprintf(os.Stderr, "连接数据库失败: %v\n", err)
		return 1
	}

	target := strings.TrimSpace(*username)
	var user usermodel.User
	if err := db.Where("username = ?", target).First(&user).Error; err != nil {
		fmt.Fprintf(os.Stderr, "用户不存在: %s\n", target)
		return 1
	}

	plain, err := password.Generate(*length)
	if err != nil {
		fmt.Fprintf(os.Stderr, "生成密码失败: %v\n", err)
		return 1
	}
	hashed, err := password.Hash(plain)
	if err != nil {
		fmt.Fprintf(os.Stderr, "加密密码失败: %v\n", err)
		return 1
	}

	// 同时解除禁用，否则重置密码后仍然无法登录
	if err := db.Model(&user).Updates(map[string]interface{}{
		"password": hashed,
		"status":   1,
	}).Error; err != nil {
		fmt.Fprintf(os.Stderr, "更新密码失败: %v\n", err)
		return 1
	}

	fmt.Printf("账号 %s 的密码已重置（该密码只显示一次）：\n%s\n", target, plain)
	return 0
}
