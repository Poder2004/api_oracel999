package handlers

import (
	"fmt"
	"net/http"

	"my-go-project/models"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

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


	if err := db.Model(&models.Lotto{}).
	
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


func ReleaseRewards(c *gin.Context, db *gorm.DB) {
	var req ReleaseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "invalid request body"})
		return
	}

	// เริ่ม Transaction
	tx := db.Begin()
	if tx.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "failed to start transaction"})
		return
	}

	// 1. ลบรางวัลเก่า
	if err := tx.Exec("DELETE FROM rewards").Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "failed to clear old rewards"})
		return
	}

	// 2. เตรียมและ INSERT รางวัลใหม่
	newRewards := make([]models.Reward, 0, len(req.Rewards))
	for _, r := range req.Rewards {
		newRewards = append(newRewards, models.Reward{
			LottoID:    r.LottoID,
			PrizeMoney: r.PrizeMoney,
			PrizeTier:  r.PrizeTier,
		})
	}
	if err := tx.Create(&newRewards).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "failed to insert new rewards"})
		return
	}

		// --- สร้าง slice ของ lotto_number ---
		lottoNumbers := make([]string, 0, len(req.Rewards))
		for _, r := range req.Rewards {
			var lotto models.Lotto
			if err := tx.Select("lotto_number").Where("lotto_id = ?", r.LottoID).First(&lotto).Error; err != nil {
				tx.Rollback()
				c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "failed to fetch lotto numbers"})
				return
			}
			lottoNumbers = append(lottoNumbers, lotto.LottoNumber)
		}

			if err := tx.Exec(`
		UPDATE purchases_detail pd
		JOIN lotto l ON l.lotto_id = pd.lotto_id
		SET pd.status = 'ถูก'
		WHERE l.lotto_number IN ?
	`, lottoNumbers).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "failed to update winning purchase details"})
		return
	}

	// สถานะไม่ถูก
	if err := tx.Exec(`
		UPDATE purchases_detail pd
		JOIN lotto l ON l.lotto_id = pd.lotto_id
		SET pd.status = 'ไม่ถูก'
		WHERE l.lotto_number NOT IN ? AND pd.status = 'ยัง'
	`, lottoNumbers).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "failed to update losing purchase details"})
		return
	}

	if err := tx.Exec(`
		UPDATE purchases_detail pd
		JOIN lotto l ON l.lotto_id = pd.lotto_id
		JOIN rewards r ON r.prize_tier = 4
		JOIN lotto lr ON lr.lotto_id = r.lotto_id
		SET pd.status = 'ถูก'
		WHERE RIGHT(l.lotto_number, 3) = RIGHT(lr.lotto_number, 3)
	`).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "failed to update prize tier 4 winning details"})
		return
	}

			if err := tx.Exec(`
			UPDATE purchases_detail pd
			JOIN lotto l ON l.lotto_id = pd.lotto_id
			JOIN rewards r ON r.prize_tier = 5
			JOIN lotto lr ON lr.lotto_id = r.lotto_id
			SET pd.status = 'ถูก'
			WHERE RIGHT(l.lotto_number, 2) = RIGHT(lr.lotto_number, 2)
		`).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "failed to update prize tier 5 winning details"})
			return
		}

	// Commit Transaction
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