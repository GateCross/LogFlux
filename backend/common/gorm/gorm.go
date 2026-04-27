package gorm

import (
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func InitGorm(dataSource string) *gorm.DB {
	newLogger := NewLogxLogger()

	db, err := gorm.Open(postgres.Open(dataSource), &gorm.Config{
		Logger: newLogger,
	})
	if err != nil {
		panic("连接数据库失败: " + err.Error())
	}

	sqlDB, err := db.DB()
	if err != nil {
		panic("获取 sqlDB 失败: " + err.Error())
	}

	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetConnMaxLifetime(time.Hour)

	return db
}
