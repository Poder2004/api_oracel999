package handlers

import (
	"fmt"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"my-go-project/models"
)

func GetAllLotto(c *gin.Context, db *gorm.DB) {

	const sql = "SELECT * FROM lottos ORDER BY lotto_id ASC"

	// 2. Execute คำสั่ง SQL และ Scan ผลลัพธ์ลงใน slice `items`
	var items []models.Lotto
	if err := db.Raw(sql).Scan(&items).Error; err != nil {
		c.JSON(500, gin.H{"status": "error", "message": err.Error()})
		return
	}

	// --- ส่วนของการตอบกลับ ---
	c.JSON(200, gin.H{
		"status": "success",
		"count":  len(items), // ใช้ count แทน page/limit เพื่อแยกให้ออกว่าเป็นตัวใหม่
		"data":   items,
	})
}


// สุ่มเลข 6 หลักแบบสตริง เช่น "042317"
func random6() string {
	return fmt.Sprintf("%06d", rand.Intn(1_000_000))
}

func InsertLotto(c *gin.Context, db *gorm.DB) {
	//ส่วนของการ Seed, อ่านค่า count, และกำหนด createdBy
	rand.Seed(time.Now().UnixNano())
	want := 100
	if v := c.Query("count"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			want = n
		}
	}
	if want > 10000 {
		want = 10000
	}
	var createdBy *uint = nil

	// --- ส่วนของ Loop และการสร้าง Batch  ---
	inserted, attempts := 0, 0
	const maxAttempts = 200

	for inserted < want && attempts < maxAttempts {
		attempts++

		batchSize := want - inserted
		if batchSize > 200 {
			batchSize = 200
		}

		seen := make(map[string]struct{}, batchSize)
		batch := make([]models.Lotto, 0, batchSize)

		for len(batch) < batchSize {
			n := random6()
			if _, dup := seen[n]; dup {
				continue
			}
			seen[n] = struct{}{}
			batch = append(batch, models.Lotto{
				LottoNumber: n,
				Status:      "sell",
				Price:       80,
				CreatedBy:   createdBy,
			})
		}

		var args []interface{}
		var sqlBuilder strings.Builder

		// สำหรับ MySQL, "INSERT IGNORE" คือวิธีที่ง่ายที่สุดในการทำ "Do Nothing" on conflict
		sqlBuilder.WriteString("INSERT IGNORE INTO lottos (lotto_number, status, price, created_by) VALUES ")

		// 2. สร้าง placeholders '(?,?,?,?)' และ arguments สำหรับแต่ละรายการใน batch
		for i, item := range batch {
			if i > 0 {
				sqlBuilder.WriteString(", ") // เติมจุลภาคคั่นระหว่าง VALUES
			}
			sqlBuilder.WriteString("(?, ?, ?, ?)")
			args = append(args, item.LottoNumber, item.Status, item.Price, item.CreatedBy)
		}

		// 3. Execute คำสั่ง SQL ที่สร้างขึ้นมา
		res := db.Exec(sqlBuilder.String(), args...)
		// ----------------------------------------------------

		if res.Error != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"status":  "error",
				"message": "เพิ่มลอตเตอรี่ไม่สำเร็จ: " + res.Error.Error(),
			})
			return
		}

		inserted += int(res.RowsAffected)
	}

	c.JSON(http.StatusOK, gin.H{
		"status":   "success",
		"inserted": inserted,
	})
}
