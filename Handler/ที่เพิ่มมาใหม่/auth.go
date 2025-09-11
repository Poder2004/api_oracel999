package handlers

import (
	"my-go-project/models"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// RegisterHandler รับคำขอการสมัครสมาชิก
// RegisterHandler รับคำขอสมัครสมาชิก
func RegisterHandler(c *gin.Context, db *gorm.DB) {
	var json models.User
	if err := c.ShouldBindJSON(&json); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// เข้ารหัสรหัสผ่านก่อนบันทึก
	encryptedPassword, err := bcrypt.GenerateFromPassword([]byte(json.Password), 10)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to encrypt password"})
		return
	}
	json.Password = string(encryptedPassword)

	// บันทึกข้อมูลผู้ใช้ใหม่
	result := db.Create(&json)
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": result.Error.Error()})
		return
	}

	// ตรวจสอบว่าข้อมูลถูกบันทึกสำเร็จ
	if json.UserID > 0 {
		c.JSON(http.StatusOK, gin.H{
			"status":  "ok",
			"userID":  json.UserID,
			"message": "User successfully created",
		})
	} else {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": "User creation failed",
		})
	}
}

// LoginHandler รับคำขอล็อกอิน
func LoginHandler(c *gin.Context, db *gorm.DB) {
	var input struct {
		Email    string `json:"email" binding:"required,email"`
		Password string `json:"password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": err.Error()})
		return
	}

	// ค้นหาผู้ใช้
	var user models.User
	if err := db.Where("email = ?", input.Email).First(&user).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"status": "error", "message": "Invalid email or password"})
		return
	}

	// ตรวจรหัสผ่าน
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(input.Password)); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"status": "error", "message": "Invalid email or password"})
		return
	}

	// ตอบกลับ (ไม่ส่ง password)
	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Login successful",
		"user": gin.H{
			"user_id":  user.UserID,
			"username": user.Username,
			"email":    user.Email,
			"role":     user.Role,
			"wallet":   user.Wallet,
		},
	})
}

func Profile(c *gin.Context, db *gorm.DB) {
	userIDStr := c.Query("user_id") // ดึง user_id จาก query string
	userID, err := strconv.ParseUint(userIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user_id"})
		return
	}

	var user models.User
	if err := db.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "profile found",
		"user": gin.H{
			"username": user.Username,
			"email":    user.Email,
		},
	})
}

func Wallet(c *gin.Context, db *gorm.DB) {
	userIDStr := c.Query("user_id") // ดึง user_id จาก query string
	userID, err := strconv.ParseUint(userIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user_id"})
		return
	}

	var user models.User
	if err := db.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "wallet found",
		"user": gin.H{
			"wallet": user.Wallet,
		},
	})
}