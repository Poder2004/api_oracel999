package handlers

import (
	"fmt"
	"math/rand"
	"net/http"
	"strconv"
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

// GET /lotto/sell  -> ดึงเฉพาะที่ยังขายอยู่ (status='sell')
func GetLottoSell(c *gin.Context, db *gorm.DB) {
	var items []models.Lotto
	if err := db.Model(&models.Lotto{}).
		Where("status = ?", "sell").
		Order("lotto_id ASC").
		Find(&items).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"total":  len(items),
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

func LottoLucky(c *gin.Context, db *gorm.DB) {
	// 1. กำหนดจำนวนที่ต้องการสุ่ม
	// การใช้ค่าคงที่ช่วยให้จัดการโค้ดได้ง่ายขึ้น
	const luckyLottoCount = 3

	var items []models.Lotto

	// 2. สร้าง Query ที่มีประสิทธิภาพ
	// เราจะให้ Database ทำงานหนักแทนเรา
	if err := db.Model(&models.Lotto{}).
		Where("status = ?", "sell").     // 2.1 กรองเอาเฉพาะสลากที่ยัง "ขาย" อยู่
		Order("RAND()").                 // 2.2 สั่งให้เรียงลำดับแบบสุ่ม (สำคัญมาก!)
		Limit(luckyLottoCount).          // 2.3 จำกัดจำนวนผลลัพธ์แค่ 5 รายการ
		Find(&items).Error; err != nil { // 2.4 ดึงข้อมูลที่ผ่านเงื่อนไขทั้งหมด

		// จัดการ Error กรณีที่การ query ล้มเหลว
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": err.Error()})
		return
	}

	// 3. ส่งข้อมูลที่สุ่มและกรองแล้วกลับไป
	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"total":  len(items), // จำนวนที่ได้มาจริง (อาจน้อยกว่า 5 ถ้ามีไม่ถึง)
		"data":   items,
	})
}

func LottoAuspicious(c *gin.Context, db *gorm.DB) {
	// 1. กำหนดจำนวนที่ต้องการสุ่ม
	// การใช้ค่าคงที่ช่วยให้จัดการโค้ดได้ง่ายขึ้น
	const luckyLottoCount = 3

	var items []models.Lotto

	// 2. สร้าง Query ที่มีประสิทธิภาพ
	// เราจะให้ Database ทำงานหนักแทนเรา
	if err := db.Model(&models.Lotto{}).
		Where("status = ?", "sell").     // 2.1 กรองเอาเฉพาะสลากที่ยัง "ขาย" อยู่
		Order("RAND()").                 // 2.2 สั่งให้เรียงลำดับแบบสุ่ม (สำคัญมาก!)
		Limit(luckyLottoCount).          // 2.3 จำกัดจำนวนผลลัพธ์แค่ 5 รายการ
		Find(&items).Error; err != nil { // 2.4 ดึงข้อมูลที่ผ่านเงื่อนไขทั้งหมด

		// จัดการ Error กรณีที่การ query ล้มเหลว
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": err.Error()})
		return
	}

	// 3. ส่งข้อมูลที่สุ่มและกรองแล้วกลับไป
	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"total":  len(items), // จำนวนที่ได้มาจริง (อาจน้อยกว่า 5 ถ้ามีไม่ถึง)
		"data":   items,
	})
}
