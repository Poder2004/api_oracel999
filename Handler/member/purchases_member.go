// Z:\api_oracel999\Handler\purchasesController.go
package handlers

import (
	"errors"
	"net/http"
	"sort"
	"strconv"

	"my-go-project/models"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// ---------- Request Models ----------
type BuyRequest struct {
	UserID      uint     `json:"user_id"  binding:"required"`
	LottoIDs    []uint   `json:"lotto_ids" binding:"required,min=1"`
	ClientTotal *float64 `json:"client_total,omitempty"` // (optional) ส่งมาเทียบได้ แต่เซิร์ฟเวอร์คำนวณเองเสมอ
}

// ---------- ซื้อจริง (INSERT ทั้งบิล) ----------
var errNotAvailable = errors.New("some tickets are not available")

func CreatePurchase(c *gin.Context, db *gorm.DB) {
	// --- ส่วนของการรับและตรวจสอบ Input  ---
	var req BuyRequest
	if err := c.ShouldBindJSON(&req); err != nil || len(req.LottoIDs) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "invalid request"})
		return
	}

	var user models.User
	// ตรวจสอบว่ามีผู้ใช้นี้จริง
	if err := db.Raw("SELECT * FROM users WHERE user_id = ?", req.UserID).Scan(&user).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "user not found"})
		return
	}

	//ส่วนของการตัด ID ซ้ำ  ---
	idset := map[uint]struct{}{}
	uniq := make([]uint, 0, len(req.LottoIDs))
	for _, id := range req.LottoIDs {
		if id == 0 {
			continue
		}
		if _, ok := idset[id]; !ok {
			idset[id] = struct{}{}
			uniq = append(uniq, id)
		}
	}
	if len(uniq) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "no lotto ids"})
		return
	}

	// --- ตัวแปรสำหรับตอบกลับ  ---
	var (
		purchaseID   uint
		totalPrice   float64
		respItems    []map[string]any //รายการสลากที่ซื้อสำเร็จ
		notAvailable []uint           //id ที่ไม่สามารถซื้อได้ (ถูกคนอื่นซื้อแล้ว)
	)

	// ---------- เริ่ม Transaction ----------
	//ใช้ Transaction เพื่อให้ทุกขั้นตอน (ตรวจสอบ, insert, update, หักเงิน) ต้องสำเร็จพร้อมกัน
	//ถ้ามี error → rollback ย้อนกลับทั้งหมด
	err := db.Transaction(func(tx *gorm.DB) error {

		var lottos []models.Lotto
		lockSQL := "SELECT * FROM lotto WHERE lotto_id IN (?) AND status = ? ORDER BY lotto_id ASC FOR UPDATE"
		if err := tx.Raw(lockSQL, uniq, "sell").Scan(&lottos).Error; err != nil {
			return err
		}

		// --- ตรวจสอบว่าสลากครบไหม  ---
		if len(lottos) != len(uniq) {
			found := make(map[uint]struct{}, len(lottos))
			for _, l := range lottos {
				found[l.LottoID] = struct{}{}
			}
			for _, id := range uniq {
				if _, ok := found[id]; !ok {
					notAvailable = append(notAvailable, id)
				}
			}
			return errNotAvailable
		}

		// --- รวมราคา และเตรียม response ---
		for _, l := range lottos {
			totalPrice += l.Price
			respItems = append(respItems, map[string]any{
				"lotto_id":     l.LottoID,
				"lotto_number": l.LottoNumber,
				"price":        l.Price,
			})
		}
		sort.Slice(respItems, func(i, j int) bool {
			return respItems[i]["lotto_id"].(uint) < respItems[j]["lotto_id"].(uint)
		})

		//  สร้างหัวบิล (ใช้ Create เพื่อให้ได้ PurchaseID กลับมา)
		p := models.Purchase{
			UserID:     req.UserID,
			TotalPrice: totalPrice,
		}
		if err := tx.Create(&p).Error; err != nil {
			return err
		}
		purchaseID = p.PurchaseID // GORM จะใส่ ID ที่เพิ่งสร้างให้เราอัตโนมัติ

		//  สร้างรายละเอียดบิล 
		details := make([]models.PurchaseDetail, 0, len(lottos))
		for _, l := range lottos {
			details = append(details, models.PurchaseDetail{
				PurchaseID: purchaseID,
				LottoID:    l.LottoID,
			})
		}
		if err := tx.Create(&details).Error; err != nil {
			return err
		}

		// เปลี่ยนสถานะลอตเตอรี่เป็น "sold"
		updateStatusSQL := "UPDATE lotto SET status = ? WHERE lotto_id IN (?)"
		if err := tx.Exec(updateStatusSQL, "sold", uniq).Error; err != nil {
			return err
		}

		//  หักเงินในกระเป๋า (wallet)
		if user.Wallet < totalPrice {
			return errors.New("ยอดเงินในกระเป๋าไม่เพียงพอ")
		}
		updateWalletSQL := "UPDATE users SET wallet = wallet - ? WHERE user_id = ?"
		if err := tx.Exec(updateWalletSQL, totalPrice, req.UserID).Error; err != nil {
			return err
		}

		//  ดึงยอดเงินในกระเป๋าใหม่หลังหักเงิน
		selectWalletSQL := "SELECT wallet FROM users WHERE user_id = ?"
		if err := tx.Raw(selectWalletSQL, req.UserID).Scan(&user).Error; err != nil {
			return err
		}

		return nil // Commit Transaction
	})
	// ---------- จบ Transaction ----------

	// --- ส่วนของการตอบกลับ  ---
	if errors.Is(err, errNotAvailable) {
		c.JSON(http.StatusConflict, gin.H{
			"status":        "error",
			"message":       "some tickets are not available",
			"not_available": notAvailable,
		})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":      "success",
		"purchase_id": purchaseID,
		"total_price": totalPrice,
		"items":       respItems,
		"wallet":      user.Wallet,
	})
}

// ---------- ดึงรายการสลากที่ผู้ใช้ซื้อ ----------

func ListPurchasedLottosByUser(c *gin.Context, db *gorm.DB) {
	// --- ส่วนของการรับและตรวจสอบ Input---
	// 	Path Parameter เช่น /users/purchases/5
	// Query String เช่น /users/purchases?user_id=5
	uidStr := c.Param("user_id")
	if uidStr == "" {
		uidStr = c.Query("user_id")
	}
	uid, err := strconv.Atoi(uidStr)
	if err != nil || uid <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "invalid user_id"})
		return
	}

	// กำหนด struct สำหรับรับข้อมูล
	type Row struct {
		LottoID   uint   `json:"lotto_id"`
		LottoName string `json:"lotto_name"`
		Status    string `json:"status"`
	}
	var rows []Row

	const sql = `
		SELECT
			l.lotto_id,
			l.lotto_number AS lotto_name,
			pd.status
		FROM
			purchases_detail AS pd
		JOIN purchases p ON p.purchase_id = pd.purchase_id
		JOIN lotto l ON l.lotto_id = pd.lotto_id
		WHERE
			p.user_id = ?
		ORDER BY
			pd.pd_id ASC`

	// Execute คำสั่ง SQL และ Scan ผลลัพธ์ลงใน slice `rows`
	if err := db.Raw(sql, uid).Scan(&rows).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": err.Error()})
		return
	}

	// --- ส่วนของการตอบกลับ  ---
	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"total":  len(rows),
		"data":   rows,
	})
}
