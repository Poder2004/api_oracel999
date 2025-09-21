package handlers

import (
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"my-go-project/models"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func SearchLottoByNumber(c *gin.Context, db *gorm.DB) {
	
	numberQuery := c.Query("number")
	status := c.Query("status")
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

	var args []interface{}

	sql := "SELECT * FROM lotto"

	var whereClauses []string // เก็บเงื่อนไขแต่ละอัน

	// เพิ่มเงื่อนไขการค้นหาด้วย `number` 
	whereClauses = append(whereClauses, "lotto_number LIKE ?")
	args = append(args, "%"+numberQuery+"%")

	// เพิ่มเงื่อนไขการค้นหาด้วย `status` (ถ้ามี)
	if status == "sell" || status == "sold" {
		whereClauses = append(whereClauses, "LOWER(TRIM(status)) = ?")
		args = append(args, status)
	}


	if len(whereClauses) > 0 {
		sql += " WHERE " + strings.Join(whereClauses, " AND ")
	}

	sql += " ORDER BY lotto_id ASC LIMIT ?"
	args = append(args, limit)

	
	var items []models.Lotto
	if err := db.Raw(sql, args...).Scan(&items).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"count":  len(items),
		"data":   items,
	})
}

func RandomLotto(c *gin.Context, db *gorm.DB) {
	sellOnly := c.DefaultQuery("sell_only", "true") // กำหนดค่าเริ่มต้นเป็น true


	var args []interface{}
	sql := "SELECT * FROM lotto"

	if sellOnly == "true" {
		sql += " WHERE LOWER(TRIM(status)) = ?"
		args = append(args, "sell")
	}

	//  เพิ่มส่วนท้ายของ Query เพื่อสุ่มและจำกัดแค่ 1 แถว
	sql += " ORDER BY RAND() LIMIT 1"


	var item models.Lotto
	if err := db.Raw(sql, args...).Scan(&item).Error; err != nil {
		// การจัดการ Error ยังคงเหมือนเดิม
		if err == gorm.ErrRecordNotFound {
			// ถ้าไม่พบข้อมูลเลย ให้ส่งค่า data เป็น null กลับไป
			c.JSON(http.StatusOK, gin.H{"status": "success", "data": nil})
			return
		}
		// หากเกิด Error อื่นๆ
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": err.Error()})
		return
	}


	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   item,
	})
}
