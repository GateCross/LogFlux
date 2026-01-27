package main

import (
	"fmt"
	"log"

	"logflux/model"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func main() {
	dsn := "host=192.168.50.10 user=postgres password=postgres dbname=logflux port=5432 sslmode=disable TimeZone=Asia/Shanghai"
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("failed to connect database: %v", err)
	}

	// 1. Check User admin
	var user model.User
	if err := db.Where("username = ?", "admin").First(&user).Error; err != nil {
		log.Printf("Failed to find admin user: %v", err)
	} else {
		fmt.Printf("User: %s, ID: %d, Roles: %v\n", user.Username, user.ID, user.Roles)
	}

	// 2. Check Role admin
	var role model.Role
	if err := db.Where("name = ?", "admin").First(&role).Error; err != nil {
		log.Printf("Failed to find admin role: %v", err)
	} else {
		fmt.Printf("Role: %s, Permissions: %v\n", role.Name, role.Permissions)
	}

	// 3. Test Permission Check Logic
	userPermissions := make(map[string]bool)
	var roles []model.Role
	if len(user.Roles) > 0 {
		db.Where("name = ANY(?)", user.Roles).Find(&roles)
	}
	for _, r := range roles {
		for _, perm := range r.Permissions {
			userPermissions[perm] = true
		}
	}
	fmt.Printf("Calculated User Permissions: %v\n", userPermissions)
}
