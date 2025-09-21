package handlers

import (
	"fmt" // เพิ่ม: import ที่จำเป็น
	"net/http"
	"strings"

	"my-go-project/models"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// --- Struct สำหรับ Response ---
type CheckResult struct {
	IsWinner    bool    `json:"is_winner"`
	PrizeTier   int     `json:"prize_tier"`
	PrizeMoney  float64 `json:"prize_money"`
	Message     string  `json:"message"`
	LottoNumber string  `json:"lotto_number"`
}

// - ตรวจสอบสลากของผู้ใช้
func CheckUserLotto(c *gin.Context, db *gorm.DB) {

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
	var prize1Money float64
	var prize5Money float64

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
			if winningLotto.PrizeTier == 4 {
				prize1Money = winningLotto.PrizeMoney
			}
		}
		if winningLotto.PrizeTier == 5 && winningLotto.Lotto != nil && len(winningLotto.Lotto.LottoNumber) == 6 {
			prize5Number = winningLotto.Lotto.LottoNumber[4:]
			prize5Money = winningLotto.PrizeMoney
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
					PrizeMoney:  prize1Money,
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
					PrizeMoney:  prize5Money,
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

// ใช้ CashInRequest struct
type CashInRequest struct {
	UserID      uint   `json:"user_id"`
	LottoNumber string `json:"lotto_number"`
}

func CashIn(c *gin.Context, db *gorm.DB) {
	var req CashInRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request data"})
		return
	}

	// 1. หา Lotto จากหมายเลข
	var lotto models.Lotto
	result := db.Raw("SELECT lotto_id FROM lotto WHERE lotto_number = ? LIMIT 1", req.LottoNumber).Scan(&lotto)
	if result.Error != nil || result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Lotto number not found"})
		return
	}

	// 2. ตรวจสอบว่า User เป็นเจ้าของ Lotto ใบนี้จริงหรือไม่ และดึง status cash_in มาด้วย
	type PDRow struct {
		PDID   uint
		CashIn string
	}
	var pd PDRow
	result = db.Raw(`
		SELECT pd.pd_id, pd.cash_in
		FROM purchases_detail AS pd
		JOIN purchases AS p ON p.purchase_id = pd.purchase_id
		WHERE pd.lotto_id = ? AND p.user_id = ?
		LIMIT 1`, lotto.LottoID, req.UserID).Scan(&pd)

	if result.Error != nil || result.RowsAffected == 0 {
		c.JSON(http.StatusForbidden, gin.H{"error": "You do not own this lottery ticket"})
		return
	}
	if pd.CashIn == "ขึ้นเงิน" {
		c.JSON(http.StatusConflict, gin.H{"error": "This prize has already been claimed"})
		return
	}

	// 3. ตรวจสอบว่า Lotto ใบนี้ถูกรางวัลจริงหรือไม่
	var reward models.Reward
	result = db.Raw("SELECT * FROM rewards WHERE lotto_id = ? LIMIT 1", lotto.LottoID).Scan(&reward)
	if result.Error != nil || result.RowsAffected == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "This ticket is not a winning ticket"})
		return
	}

	// 4. Transaction เริ่ม
	tx := db.Begin()
	if tx.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start transaction"})
		return
	}

	// 5. อัปเดต Wallet ของ User
	err := tx.Exec("UPDATE users SET wallet = wallet + ? WHERE user_id = ?", reward.PrizeMoney, req.UserID).Error
	if err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update user wallet"})
		return
	}

	// 6. อัปเดต purchases_detail.cash_in = 'ขึ้นเงิน'
	err = tx.Exec("UPDATE purchases_detail SET cash_in = ? WHERE pd_id = ?", "ขึ้นเงิน", pd.PDID).Error
	if err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update purchase detail status"})
		return
	}

	// 7. Commit Transaction
	if err := tx.Commit().Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to commit transaction"})
		return
	}

	// 8. Response สำเร็จ
	c.JSON(http.StatusOK, gin.H{
		"message":     "Prize claimed successfully!",
		"prize_money": reward.PrizeMoney,
	})
}
