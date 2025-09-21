package database

import (
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// ฟังก์ชันสำหรับการเชื่อมต่อฐานข้อมูล
func SetupDatabaseConnection() (*gorm.DB, error) {
	dsn := "mb68_66011212129:px4uyNPZOfxE@tcp(202.28.34.203:3306)/mb68_66011212129?charset=utf8mb4&parseTime=True&loc=Local"
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	return db, nil
}
