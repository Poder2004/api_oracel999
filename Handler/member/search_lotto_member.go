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
	// --- ส่วนของการรับและตรวจสอบ Input (เหมือนเดิม) ---
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

	// --- ส่วนที่เปลี่ยนมาใช้ db.Raw() ---

	// 1. เตรียมตัวแปรสำหรับเก็บ arguments ที่จะส่งเข้า placeholder '?'
	// ใช้ `interface{}` เพราะค่าอาจเป็นได้หลายชนิด (string, int)
	var args []interface{}

	// 2. เริ่มสร้างคำสั่ง SQL พื้นฐาน
	sql := "SELECT * FROM lotto"

	// 3. สร้าง WHERE clause แบบ Dynamic
	var whereClauses []string // เก็บเงื่อนไขแต่ละอัน

	// เพิ่มเงื่อนไขการค้นหาด้วย `number` (มีเสมอ)
	whereClauses = append(whereClauses, "lotto_number LIKE ?")
	args = append(args, "%"+numberQuery+"%")

	// เพิ่มเงื่อนไขการค้นหาด้วย `status` (ถ้ามี)
	if status == "sell" || status == "sold" {
		whereClauses = append(whereClauses, "LOWER(TRIM(status)) = ?")
		args = append(args, status)
	}

	// 4. นำ WHERE clauses ทั้งหมดมาต่อกันด้วย " AND " และเพิ่มเข้าไปใน SQL หลัก
	if len(whereClauses) > 0 {
		sql += " WHERE " + strings.Join(whereClauses, " AND ")
	}

	// 5. เพิ่มส่วนท้ายของ Query (ORDER BY, LIMIT)
	sql += " ORDER BY lotto_id ASC LIMIT ?"
	args = append(args, limit)

	// 6. Execute คำสั่ง SQL ที่สร้างขึ้นมาทั้งหมด
	var items []models.Lotto
	if err := db.Raw(sql, args...).Scan(&items).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": err.Error()})
		return
	}

	// --- ส่วนของการตอบกลับ (เหมือนเดิม) ---
	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"count":  len(items),
		"data":   items,
	})
}

func RandomLotto(c *gin.Context, db *gorm.DB) {
	sellOnly := c.DefaultQuery("sell_only", "true") // กำหนดค่าเริ่มต้นเป็น true

	// 1. เตรียมตัวแปรสำหรับเก็บ arguments และคำสั่ง SQL พื้นฐาน
	var args []interface{}
	sql := "SELECT * FROM lotto"

	// 2. เพิ่มเงื่อนไข WHERE แบบ Dynamic
	// ถ้าผู้ใช้ต้องการเฉพาะสถานะ 'sell' ให้เพิ่ม WHERE clause เข้าไป
	if sellOnly == "true" {
		sql += " WHERE LOWER(TRIM(status)) = ?"
		args = append(args, "sell")
	}

	// 3. เพิ่มส่วนท้ายของ Query เพื่อสุ่มและจำกัดแค่ 1 แถว
	sql += " ORDER BY RAND() LIMIT 1"

	// 4. Execute คำสั่ง SQL ที่สร้างขึ้น และ Scan ผลลัพธ์ลงในตัวแปร item
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

	// --- ส่วนของการตอบกลับ (เหมือนเดิม) ---
	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   item,
	})
}
