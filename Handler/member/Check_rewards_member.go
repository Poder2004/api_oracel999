package handlers

import (
	"fmt" // เพิ่ม: import ที่จำเป็น
	"net/http"
	"strings"

	"my-go-project/models"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// --- Struct สำหรับ Response ของ /rewards/latest ---
type LatestRewardsResponse struct {
	Prize1 []string `json:"prize_1"`
	Prize2 []string `json:"prize_2"`
	Prize3 []string `json:"prize_3"`
	Last3  string   `json:"last_3"`
	Last2  string   `json:"last_2"`
}

// //- ดึงผลรางวัลล่าสุดทั้งหมด
// func GetLatestRewards(c *gin.Context, db *gorm.DB) {

// 	var rewards []struct {
// 		LottoNumber string
// 		PrizeTier   int
// 	}

// 	const sql = `
// 		SELECT
// 			lotto.lotto_number,
// 			rewards.prize_tier
// 		FROM
// 			rewards
// 		JOIN lotto ON lotto.lotto_id = rewards.lotto_id
// 		ORDER BY
// 			rewards.prize_tier ASC`

// 	// Execute คำสั่ง SQL และ Scan ผลลัพธ์
// 	if err := db.Raw(sql).Scan(&rewards).Error; err != nil {
// 		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "database error"})
// 		return
// 	}

// 	// --- ส่วนของการประมวลผลเพื่อสร้าง Response  ---
// 	resp := LatestRewardsResponse{
// 		Prize1: []string{},
// 		Prize2: []string{},
// 		Prize3: []string{},
// 	}

// 	for _, r := range rewards {
// 		switch r.PrizeTier {
// 		case 1:
// 			resp.Prize1 = append(resp.Prize1, r.LottoNumber)
// 		case 2:
// 			resp.Prize2 = append(resp.Prize2, r.LottoNumber)
// 		case 3:
// 			resp.Prize3 = append(resp.Prize3, r.LottoNumber)
// 		case 5: // Tier 5 คือเลขท้าย 2 ตัว
// 			if len(r.LottoNumber) == 6 {
// 				resp.Last2 = r.LottoNumber[4:]
// 			}
// 		}
// 	}

// 	// เลขท้าย 3 ตัวมาจากรางวัลที่ 1
// 	if len(resp.Prize1) > 0 {
// 		prize1Number := resp.Prize1[0]
// 		if len(prize1Number) == 6 {
// 			resp.Last3 = prize1Number[3:]
// 		}
// 	}

// 	c.JSON(http.StatusOK, gin.H{
// 		"status": "success",
// 		"data":   resp,
// 	})
// }

// --- Struct สำหรับ Response ของ /rewards/check (อัปเดต) ---
type CheckResult struct {
	IsWinner    bool    `json:"is_winner"`
	PrizeTier   int     `json:"prize_tier"`
	PrizeMoney  float64 `json:"prize_money"`
	Message     string  `json:"message"`
	LottoNumber string  `json:"lotto_number"`
}

// - ตรวจสอบสลากของผู้ใช้
func CheckUserLotto(c *gin.Context, db *gorm.DB) {
	// --- 1. ส่วนประกาศตัวแปรที่หายไป ---
	userNumber := c.Query("number")
	if len(userNumber) != 6 {
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "number must be 6 digits"})
		return
	}

	var winningLottos []models.Reward
	if err := db.Preload("Lotto").Find(&winningLottos).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "could not fetch rewards"})
		return
	}

	var prize1Number string
	var prize5Number string
	// --- สิ้นสุดส่วนที่หายไป ---

	// --- 2. ส่วนโค้ดที่คุณส่งมา (ตอนนี้จะทำงานได้แล้ว) ---
	// ตรวจรางวัลใหญ่ (6 ตัวตรง)
	for _, winningLotto := range winningLottos {
		if winningLotto.Lotto != nil && winningLotto.Lotto.LottoNumber == userNumber {
			c.JSON(http.StatusOK, gin.H{
				"status": "success",
				"data": CheckResult{
					IsWinner:    true,
					PrizeTier:   winningLotto.PrizeTier,
					PrizeMoney:  winningLotto.PrizeMoney,
					Message:     fmt.Sprintf("คุณถูกลอตเตอรี่ รางวัลที่ %d", winningLotto.PrizeTier),
					LottoNumber: userNumber,
				},
			})
			return
		}
		if winningLotto.PrizeTier == 1 && winningLotto.Lotto != nil {
			prize1Number = winningLotto.Lotto.LottoNumber
		}
		if winningLotto.PrizeTier == 5 && winningLotto.Lotto != nil && len(winningLotto.Lotto.LottoNumber) == 6 {
			prize5Number = winningLotto.Lotto.LottoNumber[4:]
		}
	}

	// ตรวจเลขท้าย 3 ตัว (รางวัลที่ 4)
	if prize1Number != "" && len(prize1Number) == 6 {
		if strings.HasSuffix(userNumber, prize1Number[3:]) {
			c.JSON(http.StatusOK, gin.H{
				"status": "success",
				"data": CheckResult{
					IsWinner:    true,
					PrizeTier:   4,
					PrizeMoney:  4000,
					Message:     "คุณถูกรางวัลเลขท้าย 3 ตัว",
					LottoNumber: userNumber,
				},
			})
			return
		}
	}
	// ตรวจเลขท้าย 2 ตัว (รางวัลที่ 5)
	if prize5Number != "" {
		if strings.HasSuffix(userNumber, prize5Number) {
			c.JSON(http.StatusOK, gin.H{
				"status": "success",
				"data": CheckResult{
					IsWinner:    true,
					PrizeTier:   5,
					PrizeMoney:  2000,
					Message:     "คุณถูกรางวัลเลขท้าย 2 ตัว",
					LottoNumber: userNumber,
				},
			})
			return
		}
	}

	// ถ้าไม่ถูกอะไรเลย
	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data": CheckResult{
			IsWinner:    false,
			Message:     "อาจจะยังก่อนน๊า คุณไม่ถูกรางวัล",
			LottoNumber: userNumber,
		},
	})
}

// ใช้ CashInRequest struct เดิม
type CashInRequest struct {
	UserID      uint   `json:"user_id"`
	LottoNumber string `json:"lotto_number"`
}

func CashIn(c *gin.Context, db *gorm.DB) {
	var req CashInRequest

	// 1. Bind JSON request body (ส่วนนี้เหมือนเดิม)
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request data"})
		return
	}

	// --- เริ่มการตรวจสอบบน Server ---

	// 2. ค้นหา Lotto จากหมายเลขที่ส่งมาด้วย Raw SQL
	var lotto models.Lotto
	result := db.Raw("SELECT lotto_id FROM lotto WHERE lotto_number = ? LIMIT 1", req.LottoNumber).Scan(&lotto)
	if result.Error != nil || result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Lotto number not found"})
		return
	}

	// 3. ตรวจสอบว่า User เป็นเจ้าของ Lotto ใบนี้จริงหรือไม่ด้วย Raw SQL
	var purchaseDetailID uint
	result = db.Raw(`
		SELECT pd.pd_id 
		FROM purchases_detail as pd 
		JOIN purchases as p ON p.purchase_id = pd.purchase_id 
		WHERE pd.lotto_id = ? AND p.user_id = ? 
		LIMIT 1`, lotto.LottoID, req.UserID).Scan(&purchaseDetailID)

	if result.Error != nil || result.RowsAffected == 0 {
		c.JSON(http.StatusForbidden, gin.H{"error": "You do not own this lottery ticket"})
		return
	}

	// 4. ตรวจสอบว่า Lotto ใบนี้ถูกรางวัลจริงหรือไม่ด้วย Raw SQL
	var reward models.Reward
	result = db.Raw("SELECT * FROM rewards WHERE lotto_id = ? LIMIT 1", lotto.LottoID).Scan(&reward)
	if result.Error != nil || result.RowsAffected == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "This ticket is not a winning ticket"})
		return
	}

	// 5. ตรวจสอบว่าเคยขึ้นเงินรางวัลนี้ไปแล้วหรือยัง (ส่วนนี้เหมือนเดิม)
	if reward.Status == "ขึ้นเงิน" {
		c.JSON(http.StatusConflict, gin.H{"error": "This prize has already been claimed"})
		return
	}

	// --- เริ่ม Transaction ---
	tx := db.Begin()
	if tx.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start transaction"})
		return
	}

	// 6. อัปเดต Wallet ของ User ด้วย tx.Exec()
	err := tx.Exec("UPDATE users SET wallet = wallet + ? WHERE user_id = ?", reward.PrizeMoney, req.UserID).Error
	if err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update user wallet"})
		return
	}

	// 7. อัปเดตสถานะของรางวัลเป็น "ขึ้นเงิน" ด้วย tx.Exec()
	err = tx.Exec("UPDATE rewards SET status = ? WHERE reward_id = ?", "ขึ้นเงิน", reward.RewardID).Error
	if err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update reward status"})
		return
	}

	// Commit Transaction (ส่วนนี้เหมือนเดิม)
	if err := tx.Commit().Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to commit transaction"})
		return
	}
	
	// --- สิ้นสุด Transaction ---

	// 8. ส่ง Response สำเร็จกลับไป (ส่วนนี้เหมือนเดิม)
	c.JSON(http.StatusOK, gin.H{
		"message":     "Prize claimed successfully!",
		"prize_money": reward.PrizeMoney,
	})
}