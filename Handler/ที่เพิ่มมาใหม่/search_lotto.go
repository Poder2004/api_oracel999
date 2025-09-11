package handlers

import (
	"net/http"
	"regexp"
	"strconv"

	"my-go-project/models"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// GET /lottos/search?q=xx[xxx|xxxx..xxxxxx]&status=sell|sold&limit=100
// กติกา: ถ้า q มีความยาว N (2..5) => เช็ค N ตัวท้าย, ถ้า 6 หลัก => เท่ากันทั้งเลข
func SearchLotto(c *gin.Context, db *gorm.DB) {
	q := c.Query("q")
	status := c.Query("status") // ไม่บังคับ: "", "sell", "sold"
	limitStr := c.DefaultQuery("limit", "200")

	limit, _ := strconv.Atoi(limitStr)
	if limit <= 0 || limit > 1000 {
		limit = 200
	}

	// validate: ต้องเป็นตัวเลขล้วน และความยาว 2..6 (อนุโลม 1 ก็ได้ ถ้าต้องการ)
	isDigits := regexp.MustCompile(`^\d{1,6}$`).MatchString
	if !isDigits(q) {
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "q ต้องเป็นตัวเลข 1-6 หลัก"})
		return
	}

	n := len(q)

	tx := db.Model(&models.Lotto{})
	// กรองสถานะถ้าส่งมา
	if status == "sell" || status == "sold" {
		tx = tx.Where("LOWER(TRIM(status)) = ?", status)
	}

	// เงื่อนไขค้นหาตามความยาว
	switch {
	case n == 6:
		tx = tx.Where("lotto_number = ?", q)
	default: // 1..5 หลัก => เช็ค N ตัวท้าย
		// ใช้ RIGHT(...) จะดีที่สุด ไม่สนใจความยาวต้นฉบับ
		tx = tx.Where("RIGHT(lotto_number, ?) = ?", n, q)
	}

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

// GET /lottos/random?sell_only=true
func RandomLotto(c *gin.Context, db *gorm.DB) {
	sellOnly := c.DefaultQuery("sell_only", "false")

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
