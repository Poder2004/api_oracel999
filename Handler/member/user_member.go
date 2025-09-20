package handlers

import (
	"my-go-project/models"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// RegisterHandler รับคำขอสมัครสมาชิก
func RegisterHandler(c *gin.Context, db *gorm.DB) {
	var json models.User
	if err := c.ShouldBindJSON(&json); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	encryptedPassword, err := bcrypt.GenerateFromPassword([]byte(json.Password), 10)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to encrypt password"})
		return
	}

	
	sql := "INSERT INTO users (username, email, password, wallet) VALUES (?, ?, ?, ?)"
	result := db.Exec(sql, json.Username, json.Email, string(encryptedPassword), json.Wallet)
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": result.Error.Error()})
		return
	}

	if result.RowsAffected > 0 {
		c.JSON(http.StatusOK, gin.H{
			"status":  "ok",
			"message": "User successfully created",
		})
	} else {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": "User creation failed",
		})
	}
}

// LoginHandler รับคำขอล็อกอิน (เวอร์ชันใช้ Raw SQL)
func LoginHandler(c *gin.Context, db *gorm.DB) {
	var input struct {
		Email    string `json:"email" binding:"required,email"`
		Password string `json:"password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": err.Error()})
		return
	}

	// --- ส่วนที่เปลี่ยนไป ---
	// 1. เตรียมตัวแปร user และคำสั่ง SQL
	var user models.User
	sql := "SELECT * FROM users WHERE email = ?"

	// 2. ใช้ db.Raw() เพื่อรันคำสั่ง SQL และ .Scan() เพื่อนำผลลัพธ์ใส่ในตัวแปร user
	if err := db.Raw(sql, input.Email).Scan(&user).Error; err != nil {
		// หากไม่พบข้อมูล (gorm.ErrRecordNotFound) หรือเกิดข้อผิดพลาดอื่น
		c.JSON(http.StatusUnauthorized, gin.H{"status": "error", "message": "Invalid email or password"})
		return
	}

	// ตรวจรหัสผ่าน
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(input.Password)); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"status": "error", "message": "Invalid email or password"})
		return
	}

	// ส่งกลับ
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
	// --- ส่วนที่แก้ไข ---
	// เตรียมคำสั่ง SQL
	sql := "SELECT username, email FROM users WHERE user_id = ?"
	// ใช้ db.Raw() และ .Scan() เพื่อรันคำสั่ง SQL นั้น
	if err := db.Raw(sql, userID).Scan(&user).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}
	// --- สิ้นสุดส่วนที่แก้ไข ---

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
	// --- ส่วนที่แก้ไข ---
	// เตรียมคำสั่ง SQL ที่ต้องการ
	sql := "SELECT wallet FROM users WHERE user_id = ?"
	// สั่งให้ GORM รันคำสั่ง SQL นี้ แล้วนำผลลัพธ์มาใส่ในตัวแปร user
	if err := db.Raw(sql, userID).Scan(&user).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}
	// --- สิ้นสุดส่วนที่แก้ไข ---

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "wallet found",
		"user": gin.H{
			"wallet": user.Wallet,
		},
	})
}
