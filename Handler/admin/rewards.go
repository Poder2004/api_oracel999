package handlers

import (
	"fmt"
	"net/http"

	"my-go-project/models"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// --- Struct สำหรับ "ปล่อยรางวัล" (รับข้อมูลจาก Client) ---
// --- Struct สำหรับ "ปล่อยรางวัล" (รับข้อมูลจาก Client) ---
type ReleaseRequest struct {
    Rewards []struct {
        LottoID    uint    `json:"lotto_id" binding:"required,gt=0"`
        PrizeTier  int     `json:"prize_tier" binding:"required,gt=0"`
        PrizeMoney float64 `json:"prize_money" binding:"required,gte=0"`
    } `json:"rewards" binding:"required,min=1"`
}

// --- Struct สำหรับ "สุ่มรางวัล" (ส่งข้อมูลให้ Client ดูก่อน) ---
type RewardPreview struct {
	PrizeTier    int          `json:"prize_tier"`
	PrizeMoney   float64      `json:"prize_money"`
	WinningLotto models.Lotto `json:"winning_lotto"`
}

// GET /rewards/generate-preview
// ฟังก์ชันสำหรับ "สุ่มรางวัล" เพื่อให้ Admin ตรวจสอบก่อน
func GenerateRewardsPreview(c *gin.Context, db *gorm.DB) {
	var lottos []models.Lotto

	// 1. สุ่มสลากที่ยังขายอยู่ (status = 'sell') มา 4 ใบ
	//    สำหรับรางวัลที่ 1, 2, 3, และ 5
	if err := db.Model(&models.Lotto{}).
		Where("status = ?", "sell").
		Order("RAND()").
		Limit(4).
		Find(&lottos).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "database error"})
		return
	}

	// 2. ตรวจสอบว่ามีสลากเพียงพอที่จะออกรางวัลหรือไม่
	if len(lottos) < 4 {
		c.JSON(http.StatusConflict, gin.H{"status": "error", "message": fmt.Sprintf("มีสลากไม่เพียงพอที่จะออกรางวัล (ต้องการ 4 ใบ แต่พบเพียง %d ใบ)", len(lottos))})
		return
	}

	// 3. จัดเรียงข้อมูลเพื่อส่งกลับไปให้ Admin ดู
	previews := []RewardPreview{
		{PrizeTier: 1, PrizeMoney: 999999.00, WinningLotto: lottos[0]},
		{PrizeTier: 2, PrizeMoney: 200000.00, WinningLotto: lottos[1]},
		{PrizeTier: 3, PrizeMoney: 50000.00,  WinningLotto: lottos[2]},
		{PrizeTier: 4, PrizeMoney: 30000.00,  WinningLotto: lottos[0]},
		{PrizeTier: 5, PrizeMoney: 10000.00,   WinningLotto: lottos[3]},
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "สุ่มผลรางวัลสำหรับตรวจสอบสำเร็จ",
		"data":    previews,
	})
}

// POST /rewards/release
// ฟังก์ชันสำหรับ "ปล่อยรางวัล" (บันทึกข้อมูลลง DB จริง)
// ฟังก์ชันสำหรับ "ปล่อยรางวัล" (บันทึกข้อมูลลง DB จริง และ อัปเดตผลการซื้อ)
func ReleaseRewards(c *gin.Context, db *gorm.DB) {
	var req ReleaseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "invalid request body"})
		return
	}

	// เริ่ม Transaction เพื่อความปลอดภัย
	tx := db.Begin()
	if tx.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "failed to start transaction"})
		return
	}

	// 1. ลบข้อมูลรางวัลเก่าทั้งหมดในตาราง rewards ทิ้ง
	if err := tx.Exec("DELETE FROM rewards").Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "failed to clear old rewards"})
		return
	}

	// 2. เตรียมข้อมูลรางวัลชุดใหม่ที่จะ INSERT
	newRewards := make([]models.Reward, 0, len(req.Rewards))
	for _, r := range req.Rewards {
		newRewards = append(newRewards, models.Reward{
			LottoID:    r.LottoID,
			PrizeMoney: r.PrizeMoney,
			PrizeTier:  r.PrizeTier,
		})
	}

	// 3. INSERT รางวัลชุดใหม่ทั้งหมดลงในตาราง
	if err := tx.Create(&newRewards).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "failed to insert new rewards"})
		return
	}

	// --- 🚀 ส่วนที่เพิ่มเข้ามาใหม่ ---
	// 4. รวบรวม ID ของสลากที่ถูกรางวัลทั้งหมด
	winningLottoIDs := make([]uint, 0, len(req.Rewards))
	for _, r := range req.Rewards {
		winningLottoIDs = append(winningLottoIDs, r.LottoID)
	}

	// 5. อัปเดตสถานะ 'ถูก' สำหรับสลากที่ถูกรางวัลใน purchases_detail
	// GORM: UPDATE purchases_detail SET status = 'ถูก' WHERE lotto_id IN (...)
	if err := tx.Model(&models.PurchaseDetail{}).
		Where("lotto_id IN ?", winningLottoIDs).
		Update("status", "ถูก").Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "failed to update winning purchase details"})
		return
	}

	// 6. อัปเดตสถานะ 'ไม่ถูก' สำหรับสลากที่ไม่ถูกรางวัลใน purchases_detail
	// GORM: UPDATE purchases_detail SET status = 'ไม่ถูก' WHERE lotto_id NOT IN (...) AND status = 'ยัง'
	if err := tx.Model(&models.PurchaseDetail{}).
		Where("lotto_id NOT IN ?", winningLottoIDs).
		Where("status = ?", "ยัง"). // อัปเดตเฉพาะรายการที่ยังไม่เคยตรวจ
		Update("status", "ไม่ถูก").Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "failed to update losing purchase details"})
		return
	}
	// --- สิ้นสุดส่วนที่เพิ่มเข้ามาใหม่ ---

	// 7. ถ้าทุกอย่างสำเร็จ ให้ Commit Transaction
	if err := tx.Commit().Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "failed to commit transaction"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": fmt.Sprintf("ปล่อยรางวัลสำเร็จ! มีผลรางวัลใหม่ทั้งหมด %d รางวัล และอัปเดตผลการซื้อเรียบร้อยแล้ว", len(newRewards)),
	})
}


type CurrentRewardResponse struct {
	PrizeTier   int     `json:"prize_tier"`
	PrizeMoney  float64 `json:"prize_money"`
	LottoNumber string  `json:"lotto_number"`
}


// 🚀 NEW ENDPOINT 🚀
// GET /rewards/current
// ฟังก์ชันสำหรับดึงข้อมูลรางวัลที่ประกาศแล้วทั้งหมด
func GetCurrentRewards(c *gin.Context, db *gorm.DB) {
	var results []CurrentRewardResponse

	// ใช้ GORM เพื่อ JOIN ตาราง rewards และ lotto
	// 1. เริ่มจาก Model Reward
	// 2. เลือกคอลัมน์ที่ต้องการ โดยระบุชื่อตารางเพื่อความชัดเจน
	// 3. Join ตาราง lotto โดยใช้เงื่อนไข lotto.lotto_id = rewards.lotto_id
	// 4. เรียงลำดับจากรางวัลที่ 1 ไปน้อยสุด
	// 5. ใช้ .Scan() เพื่อ map ผลลัพธ์ลงใน struct ที่เราสร้างขึ้นมา (CurrentRewardResponse)
	err := db.Model(&models.Reward{}).
		Select("rewards.prize_tier, rewards.prize_money, lotto.lotto_number").
		Joins("JOIN lotto ON lotto.lotto_id = rewards.lotto_id").
		Order("rewards.prize_tier ASC").
		Scan(&results).Error

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "failed to fetch current rewards: " + err.Error()})
		return
	}

	// กรณีไม่มีข้อมูลรางวัลในระบบ
	if len(results) == 0 {
		c.JSON(http.StatusOK, gin.H{
			"status":  "success",
			"message": "ยังไม่มีการประกาศรางวัล",
			"data":    []CurrentRewardResponse{}, // ส่ง array ว่างกลับไป
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "ดึงข้อมูลรางวัลสำเร็จ",
		"data":    results,
	})
}