package handlers

import (
	"log"
	"net/http"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

)

// ClearDataHandler คือ Gin handler สำหรับลบข้อมูลโดยมีเงื่อนไข
func ClearDataHandler(c *gin.Context, db *gorm.DB) {
	// 1. รับ ID ของ Admin ที่ส่งคำสั่งลบมาจาก Request Body
	var requestBody struct {
		// ใน Flutter ส่ง `userId` ซึ่งเป็น String, แต่ใน DB เป็น uint
		// เราจะให้ Gin แปลง JSON `{ "admin_user_id": "1" }` เป็น uint ให้
		AdminUserID uint `json:"admin_user_id"`
	}

	if err := c.ShouldBindJSON(&requestBody); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": "Invalid request body: " + err.Error(),
		})
		return
	}

	// ตรวจสอบว่าได้รับ admin_user_id มาหรือไม่
	if requestBody.AdminUserID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": "admin_user_id is required and cannot be zero",
		})
		return
	}

	log.Printf("⚠️ Received request to clear all data, preserving user_id: %d. Starting transaction...", requestBody.AdminUserID)

	// เริ่ม Transaction
	tx := db.Begin()
	if tx.Error != nil {
		log.Printf("Error beginning transaction: %v", tx.Error)
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": "Could not start database transaction",
		})
		return
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// 2. ลบข้อมูลทั้งหมดจากตารางอื่นๆ
	//    (เรียงลำดับโดยคำนึงถึง Foreign Key Constraints ถ้ามี เช่น ลบ detail ก่อน master)
	tablesToClear := []string{
		"rewards",
		"purchases_detail",
		"purchases",
		"lottos",
	}

	for _, table := range tablesToClear {
		log.Printf("Clearing data from table: %s", table)
		if err := tx.Exec("DELETE FROM " + table).Error; err != nil {
			log.Printf("Error clearing table %s: %v", table, err)
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{
				"status":  "error",
				"message": "Failed to clear data from table: " + table,
			})
			return
		}
	}

	// 3. ลบข้อมูลจากตาราง users โดยยกเว้น ID ของคนที่กดลบ
	log.Printf("Clearing data from users table, preserving user_id %d...", requestBody.AdminUserID)
	if err := tx.Exec("DELETE FROM users WHERE user_id <> ?", requestBody.AdminUserID).Error; err != nil {
		log.Printf("Error clearing users table: %v", err)
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": "Failed to clear data from users table",
		})
		return
	}

	// ถ้าทุกอย่างสำเร็จ ให้ Commit Transaction
	if err := tx.Commit().Error; err != nil {
		log.Printf("Error committing transaction: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": "Could not commit transaction",
		})
		return
	}

	log.Printf("✅ GORM Transaction committed successfully. All data cleared, user %d preserved.", requestBody.AdminUserID)

	// ส่ง Response กลับไปให้ Flutter
	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "All data has been cleared successfully, except for the requesting user.",
	})
}




