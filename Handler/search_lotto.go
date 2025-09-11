package handlers

import (
	"net/http"
	"regexp"
	"strconv"

	"my-go-project/models"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// --- 1. แก้ไข: เปลี่ยนชื่อฟังก์ชันให้ตรงกับที่ Router เรียก ---
// GET /lotto/search?number=xxxxxx[&status=sell]
func SearchLottoByNumber(c *gin.Context, db *gorm.DB) {
	numberQuery := c.Query("number")
	// --- 1. แก้ไข: เปลี่ยนจาก DefaultQuery เป็น Query ---
	// การใช้ c.Query("status") จะทำให้ถ้าผู้ใช้ไม่ส่ง status มา, ค่าจะเป็น "" (สตริงว่าง)
	// ซึ่งจะทำให้เงื่อนไข if ด้านล่างไม่ทำงาน และไม่เกิดการกรองสถานะ
	status := c.Query("status") // ไม่มีการกำหนดค่าเริ่มต้นอีกต่อไป
	limitStr := c.DefaultQuery("limit", "200")

	limit, _ := strconv.Atoi(limitStr)
	if limit <= 0 || limit > 1000 {
		limit = 200
	}

	isDigits := regexp.MustCompile(`^\d{1,6}$`).MatchString
	if !isDigits(numberQuery) {
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "number ต้องเป็นตัวเลข 1-6 หลัก"})
		return
	}

	tx := db.Model(&models.Lotto{})

	// --- 2. ส่วนนี้ทำงานถูกต้องเหมือนเดิม ---
	// ถ้าผู้ใช้ส่ง ?status=sell หรือ ?status=sold เข้ามา, โค้ดส่วนนี้จะทำงาน
	// แต่ถ้าไม่ส่งมา, status จะเป็น "" และโค้ดจะข้ามส่วนนี้ไป
	if status == "sell" || status == "sold" {
		tx = tx.Where("LOWER(TRIM(status)) = ?", status)
	}

	tx = tx.Where("lotto_number LIKE ?", "%"+numberQuery+"%")

	var items []models.Lotto
	if err := tx.Order("lotto_id ASC").Limit(limit).Find(&items).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"count":  len(items),
		"data":   items,
	})
}

// GET /lotto/random?sell_only=true
// (ฟังก์ชันนี้ถูกต้องอยู่แล้ว ไม่ต้องแก้ไข)
func RandomLotto(c *gin.Context, db *gorm.DB) {
	sellOnly := c.DefaultQuery("sell_only", "true") // กำหนดค่าเริ่มต้นเป็น true

	tx := db.Model(&models.Lotto{})
	if sellOnly == "true" {
		tx = tx.Where("LOWER(TRIM(status)) = ?", "sell")
	}

	var item models.Lotto
	// ORDER BY RAND() LIMIT 1 (สำหรับ MySQL)
	if err := tx.Order("RAND()").Limit(1).First(&item).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusOK, gin.H{"status": "success", "data": nil})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   item,
	})
}
