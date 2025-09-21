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

	const sql = "SELECT * FROM lotto ORDER BY lotto_id ASC"

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

type PreviewUpdateItem struct {
	LottoID        uint   `json:"lotto_id"`
	LottoNumberOld string `json:"lotto_number_old"`
	LottoNumberNew string `json:"lotto_number_new"`
}

// POST /lottos/preview-update?count=100&status=sell
// เลือก lotto เดิม (ตามสถานะ) มา N รายการ แล้วสุ่มเลขใหม่ที่ไม่ชน DB ให้แต่ละรายการ (ยังไม่บันทึก)
// เพิ่ม import ถ้ายังไม่มี
// import "strings"

// สุ่มเลขใหม่ 100 ตัว ไม่ยุ่งกับ DB เดิม
func PreviewNewLotto(c *gin.Context) {
	rand.Seed(time.Now().UnixNano())

	// จำนวนที่ต้องการสุ่ม (default = 100)
	want := 100
	if v := c.Query("count"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			want = n
		}
	}
	if want > 10000 {
		want = 10000
	}

	seen := make(map[string]struct{}, want)
	out := make([]map[string]interface{}, 0, want)

	for len(out) < want {
		n := random6()
		if _, dup := seen[n]; dup {
			continue
		}
		seen[n] = struct{}{}

		out = append(out, map[string]interface{}{
			"lotto_number": n,
			"status":       "sell", // default
			"price":        80,     // default
			"created_by":   nil,    // default
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"count":  len(out),
		"items":  out, // ใช้ key "items" ให้พร้อมส่งต่อไป ResetAndInsertLotto ได้เลย
	})
}

// Reset + Insert ใหม่
type NewLottoItem struct {
	LottoNumber string  `json:"lotto_number"`
	Status      string  `json:"status"`
	Price       float64 `json:"price"`
	CreatedBy   *uint   `json:"created_by"`
}

type ResetInsertReq struct {
	Items []NewLottoItem `json:"items"`
}

// handlersadmin/lotto_handler.go (หรือไฟล์ที่คุณเก็บ handler)

// ClearLottoDataHandler clears all data from the lotto table.
func ClearLottoDataHandler(c *gin.Context, db *gorm.DB) {
    // เริ่ม Transaction เพื่อให้แน่ใจว่าการลบและรีเซ็ตจะสำเร็จไปพร้อมกัน
    tx := db.Begin()
    if tx.Error != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "failed to begin transaction"})
        return
    }

    // 1. ลบข้อมูลทั้งหมดในตาราง lotto
    if err := tx.Exec("DELETE FROM lotto").Error; err != nil {
        tx.Rollback() // หากลบไม่สำเร็จ ให้ยกเลิกทั้งหมด
        c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "delete failed: " + err.Error()})
        return
    }

    // 2. รีเซ็ต AUTO_INCREMENT กลับไปเริ่มที่ 1
    if err := tx.Exec("ALTER TABLE lotto AUTO_INCREMENT = 1").Error; err != nil {
        tx.Rollback() // หากรีเซ็ตไม่สำเร็จ ให้ยกเลิกทั้งหมด
        c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "reset auto_increment failed: " + err.Error()})
        return
    }

    // ยืนยันการเปลี่ยนแปลงทั้งหมด
    if err := tx.Commit().Error; err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "commit failed: " + err.Error()})
        return
    }

    c.JSON(http.StatusOK, gin.H{
        "status":  "success",
        "message": "Lotto data has been cleared.",
    })
}

// handlersadmin/lotto_handler.go (หรือไฟล์ที่คุณเก็บ handler)

// InsertLottoHandler inserts a new batch of lotto items.
func InsertLottoHandler(c *gin.Context, db *gorm.DB) {
    var req ResetInsertReq
    if err := c.ShouldBindJSON(&req); err != nil || len(req.Items) == 0 {
        c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "invalid payload"})
        return
    }

    tx := db.Begin()
    if tx.Error != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "failed to begin transaction"})
        return
    }

    var args []interface{}
    var sqlBuilder strings.Builder
    sqlBuilder.WriteString("INSERT INTO lotto (lotto_number, status, price, created_by) VALUES ")

    for i, item := range req.Items {
        if i > 0 {
            sqlBuilder.WriteString(", ")
        }
        sqlBuilder.WriteString("(?, ?, ?, ?)")
        status := item.Status
        if status == "" {
            status = "sell"
        }
        price := item.Price
        if price <= 0 {
            price = 80
        }
        args = append(args, item.LottoNumber, status, price, item.CreatedBy)
    }

    if err := tx.Exec(sqlBuilder.String(), args...).Error; err != nil {
        tx.Rollback()
        c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "insert failed: " + err.Error()})
        return
    }

    if err := tx.Commit().Error; err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "commit failed: " + err.Error()})
        return
    }

    c.JSON(http.StatusOK, gin.H{
        "status":   "success",
        "inserted": len(req.Items),
    })
}

