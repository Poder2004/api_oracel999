package handlers

import (
	"fmt"
	"net/http"
	"strings"

	"my-go-project/models" // อย่าลืมแก้ path ให้ถูกต้อง

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// Struct สำหรับ Response
type CheckResult struct {
	IsWinner    bool    `json:"is_winner"`
	PrizeTier   int     `json:"prize_tier"`
	PrizeMoney  float64 `json:"prize_money"`
	Message     string  `json:"message"`
	LottoNumber string  `json:"lotto_number"`
}

// - ตรวจสอบสลากของผู้ใช้ (แก้ไขแล้ว)
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

	// --- Reworked Logic ---
	// ตัวแปรสำหรับเก็บรางวัลเลขท้ายโดยเฉพาะ
	var prize4Numbers []string
	var prize5Number string
	var prize4Money float64
	var prize5Money float64

	// ตรวจรางวัลใหญ่ (6 ตัวตรง) ก่อน
	for _, winningLotto := range winningLottos {
		if winningLotto.Lotto != nil && winningLotto.Lotto.LottoNumber == userNumber {
			// กรณีถูกรางวัลที่ 1, 2, 3 หรือรางวัลอื่นๆ ที่เลขตรงกันทั้ง 6 หลัก
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

		// เก็บข้อมูลรางวัลเลขท้ายเพื่อตรวจสอบทีหลัง
		switch winningLotto.PrizeTier {
		case 4: // รางวัลเลขท้าย 3 ตัว
			if winningLotto.Lotto != nil && len(winningLotto.Lotto.LottoNumber) == 6 {
				prize4Numbers = append(prize4Numbers, winningLotto.Lotto.LottoNumber[3:])
				if prize4Money == 0 { // เก็บเงินรางวัลแค่ครั้งเดียว
					prize4Money = winningLotto.PrizeMoney
				}
			}
		case 5: // รางวัลเลขท้าย 2 ตัว
			if winningLotto.Lotto != nil && len(winningLotto.Lotto.LottoNumber) == 6 {
				prize5Number = winningLotto.Lotto.LottoNumber[4:]
				prize5Money = winningLotto.PrizeMoney
			}
		}
	}

	// ตรวจรางวัลเลขท้าย 3 ตัว (รางวัลที่ 4)
	if len(prize4Numbers) > 0 {
		for _, p4num := range prize4Numbers {
			if strings.HasSuffix(userNumber, p4num) {
				c.JSON(http.StatusOK, gin.H{
					"status": "success",
					"data": CheckResult{
						IsWinner:    true,
						PrizeTier:   4,
						PrizeMoney:  prize4Money, // ใช้เงินรางวัลของ Tier 4
						Message:     "คุณถูกรางวัลเลขท้าย 3 ตัว",
						LottoNumber: userNumber,
					},
				})
				return
			}
		}
	}

	// ตรวจรางวัลเลขท้าย 2 ตัว (รางวัลที่ 5)
	if prize5Number != "" {
		if strings.HasSuffix(userNumber, prize5Number) {
			c.JSON(http.StatusOK, gin.H{
				"status": "success",
				"data": CheckResult{
					IsWinner:    true,
					PrizeTier:   5,
					PrizeMoney:  prize5Money, // ใช้เงินรางวัลของ Tier 5
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
			Message:     "เสียใจด้วยน๊า คุณไม่ถูกรางวัล",
			LottoNumber: userNumber,
		},
	})
}

// ใช้ CashInRequest struct
// ใช้ CashInRequest struct
type CashInRequest struct {
	UserID      uint   `json:"user_id"`
	LottoNumber string `json:"lotto_number"`
}

// ... (import statements and CashInRequest struct are the same) ...

// แก้ไขแล้ว
func CashIn(c *gin.Context, db *gorm.DB) {
	var req CashInRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request data"})
		return
	}

	var lotto models.Lotto
	result := db.Raw("SELECT lotto_id FROM lotto WHERE lotto_number = ? LIMIT 1", req.LottoNumber).Scan(&lotto)
	if result.Error != nil || result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Lotto number not found"})
		return
	}

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

	// --- 4. ตรวจสอบว่าถูกรางวัลหรือไม่ (Reworked Logic) ---
	var reward models.Reward
	result = db.Raw("SELECT * FROM rewards WHERE lotto_id = ? LIMIT 1", lotto.LottoID).Scan(&reward)

	userNumber := req.LottoNumber
	isWinner := false
	prizeTier := 0
	prizeMoney := float64(0)

	// Case 1: รางวัลตรง (รางวัลที่ 1–3)
	if result.Error == nil && result.RowsAffected > 0 {
		// ตรวจสอบให้แน่ใจว่าเป็นรางวัลประเภทเลขตรง 6 หลัก
		if reward.PrizeTier >= 1 && reward.PrizeTier <= 3 {
			isWinner = true
			prizeTier = reward.PrizeTier
			prizeMoney = reward.PrizeMoney
		}
	}

	// Case 2: รางวัลเลขท้าย (รางวัลที่ 4–5)
	if !isWinner {
		var allRewards []models.Reward
		if err := db.Preload("Lotto").Find(&allRewards).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch reward numbers"})
			return
		}

		// ตัวแปรสำหรับเก็บรางวัลเลขท้ายโดยเฉพาะ
		var prize4Numbers []string
		var prize4Money float64
		var prize5Number string
		var prize5Money float64

		for _, r := range allRewards {
			switch r.PrizeTier {
			case 4: // รางวัลเลขท้าย 3 ตัว
				if r.Lotto != nil && len(r.Lotto.LottoNumber) == 6 {
					prize4Numbers = append(prize4Numbers, r.Lotto.LottoNumber[3:])
					if prize4Money == 0 {
						prize4Money = r.PrizeMoney
					}
				}
			case 5: // รางวัลเลขท้าย 2 ตัว
				if r.Lotto != nil && len(r.Lotto.LottoNumber) == 6 {
					prize5Number = r.Lotto.LottoNumber[4:]
					prize5Money = r.PrizeMoney
				}
			}
		}

		// ตรวจรางวัลที่ 4 (เลขท้าย 3 ตัว)
		if len(prize4Numbers) > 0 {
			for _, p4num := range prize4Numbers {
				if strings.HasSuffix(userNumber, p4num) {
					isWinner = true
					prizeTier = 4
					prizeMoney = prize4Money // ใช้เงินรางวัลของ Tier 4
					break
				}
			}
		}

		// ตรวจรางวัลที่ 5 (เลขท้าย 2 ตัว)
		if !isWinner && prize5Number != "" {
			if strings.HasSuffix(userNumber, prize5Number) {
				isWinner = true
				prizeTier = 5
				prizeMoney = prize5Money // ใช้เงินรางวัลของ Tier 5
			}
		}
	}

	// ถ้าไม่ถูกรางวัลเลย
	if !isWinner {
		c.JSON(http.StatusBadRequest, gin.H{"error": "This ticket is not a winning ticket"})
		return
	}

	// --- 5. Transaction ---
	tx := db.Begin()
	if tx.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start transaction"})
		return
	}

	// อัปเดต Wallet ของ User
	err := tx.Exec("UPDATE users SET wallet = wallet + ? WHERE user_id = ?", prizeMoney, req.UserID).Error
	if err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update user wallet"})
		return
	}

	// อัปเดต purchases_detail.cash_in = 'ขึ้นเงิน'
	err = tx.Exec("UPDATE purchases_detail SET cash_in = ? WHERE pd_id = ?", "ขึ้นเงิน", pd.PDID).Error
	if err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update purchase detail status"})
		return
	}

	if err := tx.Commit().Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to commit transaction"})
		return
	}

	// --- 6. Response สำเร็จ ---
	c.JSON(http.StatusOK, gin.H{
		"message":     fmt.Sprintf("Prize claimed successfully! (Tier %d)", prizeTier),
		"prize_money": prizeMoney,
	})
}
