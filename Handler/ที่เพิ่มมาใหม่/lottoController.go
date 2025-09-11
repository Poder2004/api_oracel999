package handlers

import (
	"fmt"
	"math/rand"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"my-go-project/models"
)

// handlers/lotto_simple.go
func GetAllLottoASC(c *gin.Context, db *gorm.DB) {
	fmt.Println(">>> HIT GetAllLottoASC") // debug log
	var items []models.Lotto
	if err := db.Model(&models.Lotto{}).
		Order("lotto_id ASC").
		Find(&items).Error; err != nil {
		c.JSON(500, gin.H{"status": "error", "message": err.Error()})
		return
	}
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

// POST /lotto/seed?count=100
// สุ่มเลข 6 หลัก "ไม่ซ้ำ" ใส่ตาราง lotto ให้ครบ count (ดีฟอลต์ 100)
// ใช้ UNIQUE(lotto_number) กันซ้ำระดับ DB + DoNothing เพื่อข้ามเลขที่ชน
func InsertLotto(c *gin.Context, db *gorm.DB) {
	// seed rand (ครั้งแรกของโปรเซสพอ แต่อยู่ตรงนี้ก็ใช้ได้)
	rand.Seed(time.Now().UnixNano())

	// อ่านจำนวนที่ต้องการ
	want := 100
	if v := c.Query("count"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			want = n
		}
	}
	if want > 10000 { // กันยิงหนักเกินไป
		want = 10000
	}

	// ถ้ามี auth แล้วอยากบันทึกผู้สร้าง:
	var createdBy *uint = nil
	// ex:
	// if uid, ok := c.Get("user_id"); ok {
	//     if id, ok2 := uid.(uint); ok2 { createdBy = &id }
	// }

	inserted, attempts := 0, 0
	const maxAttempts = 200

	for inserted < want && attempts < maxAttempts {
		attempts++

		// ขนาด batch
		batchSize := want - inserted
		if batchSize > 200 {
			batchSize = 200
		}

		// กันซ้ำใน batch ก่อน
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

		// Insert โดยข้ามเลขที่ชน (ต้องมี UNIQUE(lotto_number) ใน DB)
		res := db.
			Clauses(clause.OnConflict{
				Columns:   []clause.Column{{Name: "lotto_number"}},
				DoNothing: true,
			}).
			Create(&batch)

		if res.Error != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"status":  "error",
				"message": "เพิ่มลอตเตอรี่ไม่สำเร็จ: " + res.Error.Error(),
			})
			return
		}

		inserted += int(res.RowsAffected)
	}

	if inserted < want {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": fmt.Sprintf("สุ่มและเพิ่มได้ %d จาก %d (เลขชนกับของเก่ามากเกินไป)", inserted, want),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":   "success",
		"inserted": inserted,
	})
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

func PreviewUpdateLotto(c *gin.Context, db *gorm.DB) {
	rand.Seed(time.Now().UnixNano())

	// จำนวนที่ต้องการ
	want := 100
	if v := c.Query("count"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			want = n
		}
	}
	if want > 5000 {
		want = 5000
	}

	// ✅ รองรับหลายสถานะคอมมา, ดีฟอลต์ = ทั้ง sell และ sold
	statusParam := strings.ToLower(strings.TrimSpace(c.DefaultQuery("status", "sell,sold")))
	var statuses []string
	if statusParam != "" {
		for _, p := range strings.Split(statusParam, ",") {
			p = strings.TrimSpace(p)
			if p != "" {
				statuses = append(statuses, p)
			}
		}
	}

	type row struct {
		LottoID     uint
		LottoNumber string
	}

	// ✅ เลือกแถวแบบสุ่มทุกครั้ง เพื่อให้ "ชุดที่" เปลี่ยนได้จริง
	var targets []row
	tx := db.Model(&models.Lotto{}).
		Select("lotto_id, lotto_number")
	if len(statuses) > 0 {
		tx = tx.Where("status IN ?", statuses)
	}
	// 👇 จุดสำคัญ: สุ่ม
	if err := tx.Order("RAND()").Limit(want).Scan(&targets).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": err.Error()})
		return
	}
	if len(targets) == 0 {
		c.JSON(http.StatusOK, gin.H{"status": "success", "count": 0, "data": []PreviewUpdateItem{}})
		return
	}

	// เช็คเลขซ้ำใน DB
	numberExists := func(n string) (bool, error) {
		var cnt int64
		if err := db.Model(&models.Lotto{}).Where("lotto_number = ?", n).Count(&cnt).Error; err != nil {
			return false, err
		}
		return cnt > 0, nil
	}

	seen := make(map[string]struct{}, want) // กันซ้ำใน request
	out := make([]PreviewUpdateItem, 0, len(targets))

	// พยายามสุ่มเลขใหม่ให้แต่ละแถว
	const maxAttemptsPerItem = 10000
	for _, t := range targets {
		found := false
		for attempt := 0; attempt < maxAttemptsPerItem; attempt++ {
			n := random6()

			// หลีกเลี่ยงเลขเดิมของแถว
			if n == t.LottoNumber {
				continue
			}
			// หลีกเลี่ยงเลขที่สุ่มซ้ำกันเองใน request
			if _, dup := seen[n]; dup {
				continue
			}
			// หลีกเลี่ยงชนกับ DB
			ok, err := numberExists(n)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": err.Error()})
				return
			}
			if ok {
				continue
			}

			seen[n] = struct{}{}
			out = append(out, PreviewUpdateItem{
				LottoID:        t.LottoID,
				LottoNumberOld: t.LottoNumber,
				LottoNumberNew: n,
			})
			found = true
			break
		}
		// ถ้าเจอเลขไม่ได้ในโควต้า attempt ก็ข้ามแถวนี้ไป (จะได้น้อยกว่า want)
		if !found {
			continue
		}
	}
		sort.Slice(out, func(i, j int) bool {
			return out[i].LottoID < out[j].LottoID
		})

	// กัน cache ขา client (เผื่อ proxy แปลกๆ)
	c.Header("Cache-Control", "no-store")

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"count":  len(out),
		"data":   out,
	})
}


type UpdateItem struct {
	LottoID     uint   `json:"lotto_id"`
	LottoNumber string `json:"lotto_number"`
}

type BulkUpdateReq struct {
	Items []UpdateItem `json:"items"`
}

func BulkUpdateLottoNumbers(c *gin.Context, db *gorm.DB) {
	var req BulkUpdateReq
	if err := c.ShouldBindJSON(&req); err != nil || len(req.Items) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "invalid payload"})
		return
	}

	// ✅ ตรวจเลขซ้ำใน payload เองก่อน เพื่อลดโอกาสชน UNIQUE ใน DB
	seen := make(map[string]struct{}, len(req.Items))
	for i := range req.Items {
		n := strings.TrimSpace(req.Items[i].LottoNumber)

		// บังคับเป็นเลข 6 หลัก (เติมศูนย์ซ้าย) และเช็คว่าเป็นตัวเลขล้วน
		if len(n) == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "empty lotto_number in payload"})
			return
		}
		if len(n) > 6 {
			n = n[len(n)-6:] // เก็บ 6 ตัวท้ายสุด
		}
		// ให้เป็น 6 หลักเสมอ
		if _, err := strconv.Atoi(n); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "lotto_number must be numeric"})
			return
		}
		n = fmt.Sprintf("%06s", n)

		req.Items[i].LottoNumber = n

		if _, dup := seen[n]; dup {
			c.JSON(http.StatusBadRequest, gin.H{
				"status":  "error",
				"message": "duplicated lotto_number in request: " + n,
			})
			return
		}
		seen[n] = struct{}{}
	}

	tx := db.Begin()
	if tx.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": tx.Error.Error()})
		return
	}

	// ✅ อัปเดตรายแถว: lotto_number + status='sell'
	for _, it := range req.Items {
		if err := tx.Model(&models.Lotto{}).
			Where("lotto_id = ?", it.LottoID).
			Updates(map[string]any{
				"lotto_number": it.LottoNumber,
				"status":       "sell",
			}).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusConflict, gin.H{
				"status":  "error",
				"message": "update failed (maybe duplicate lotto_number): " + err.Error(),
				"failed":  it,
			})
			return
		}
	}

	if err := tx.Commit().Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"updated": len(req.Items),
	})
}

func LottoCount(c *gin.Context, db *gorm.DB) {
	var cnt int64
	if err := db.Model(&models.Lotto{}).Count(&cnt).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "success", "count": cnt})
}