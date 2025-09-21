package main

import (
	"log"
	"my-go-project/database"
	"my-go-project/routers"
	"os"

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

	// // เริ่มรันเซิร์ฟเวอร์
	// r.Run(":8080")
	// ✅ เริ่มรันเซิร์ฟเวอร์ โดยใช้ PORT จาก Render
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080" // default เวลา run local
	}
	log.Printf("🚀 Server running on port %s", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatal("❌ Failed to start server: ", err)
	}
}
