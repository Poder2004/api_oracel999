package main

import (
	"my-go-project/database"
	"my-go-project/routers"

	"github.com/gin-gonic/gin"
)

func main() {
	// ตั้งค่าการเชื่อมต่อฐานข้อมูล
	db, err := database.SetupDatabaseConnection()
	if err != nil {
		panic("Failed to connect to the database")
	}

	// สร้าง Gin router
	r := gin.Default()

	// เรียกใช้ routes จาก package routers
	routers.SetupRouter(r, db)

	// เริ่มรันเซิร์ฟเวอร์
	r.Run(":8080")
}
