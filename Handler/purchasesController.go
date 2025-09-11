// Z:\api_oracel999\Handler\purchasesController.go
package handlers

import (
	"errors"
	"net/http"
	"sort"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"my-go-project/models"
)

// ---------- Request Models ----------
type BuyRequest struct {
	UserID      uint     `json:"user_id"  binding:"required"`
	LottoIDs    []uint   `json:"lotto_ids" binding:"required,min=1"` // เฉพาะใบที่ผู้ใช้ติ๊กเลือก
	ClientTotal *float64 `json:"client_total,omitempty"`             // (optional) ส่งมาเทียบได้ แต่เซิร์ฟเวอร์คำนวณเองเสมอ
}

// ---------- Quote ยอดในตะกร้า ----------
/*
POST /purchases/quote
Body:
{
  "lotto_ids": [60,99]
}
Response: items + total_price + not_available
*/
func QuotePurchase(c *gin.Context, db *gorm.DB) {
	var req struct {
		LottoIDs []uint `json:"lotto_ids" binding:"required,min=1"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "invalid request"})
		return
	}

	// ดึงเฉพาะใบที่ยังขายอยู่
	var lottos []models.Lotto
	if err := db.Where("lotto_id IN ? AND status = ?", req.LottoIDs, "sell").
		Order("lotto_id ASC").
		Find(&lottos).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": err.Error()})
		return
	}

	// สร้างผลลัพธ์
	found := make(map[uint]struct{}, len(lottos))
	total := 0.0
	type Item struct {
		LottoID     uint    `json:"lotto_id"`
		LottoNumber string  `json:"lotto_number"`
		Price       float64 `json:"price"`
	}
	items := make([]Item, 0, len(lottos))
	for _, l := range lottos {
		found[l.LottoID] = struct{}{}
		total += l.Price
		items = append(items, Item{
			LottoID:     l.LottoID,
			LottoNumber: l.LottoNumber,
			Price:       l.Price,
		})
	}

	// ใบที่ขอมาแต่ไม่ว่างขาย/ไม่พบ
	notAvail := make([]uint, 0)
	for _, id := range req.LottoIDs {
		if _, ok := found[id]; !ok {
			notAvail = append(notAvail, id)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"status":        "success",
		"items":         items,
		"total_price":   total,
		"not_available": notAvail,
	})
}

// ---------- ซื้อจริง (INSERT ทั้งบิล) ----------
var errNotAvailable = errors.New("some tickets are not available")

/*
POST /purchases
Body:

	{
	  "user_id": 1,
	  "lotto_ids": [60,99]
	}
*/
func CreatePurchase(c *gin.Context, db *gorm.DB) {
	var req BuyRequest
	if err := c.ShouldBindJSON(&req); err != nil || len(req.LottoIDs) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "invalid request"})
		return
	}

	// ตรวจว่ามีผู้ใช้นี้จริง
	var user models.User
	if err := db.First(&user, "user_id = ?", req.UserID).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "user not found"})
		return
	}

	// ตัด id ซ้ำ
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

	// ตัวแปรไว้ตอบกลับหลังจบทรานแซกชัน
	var (
		purchaseID   uint
		totalPrice   float64
		respItems    []map[string]any
		notAvailable []uint
	)

	// ---------- เริ่ม Transaction ----------
	err := db.Transaction(func(tx *gorm.DB) error {
		// ล็อกแถวที่กำลังจะซื้อ (กันแข่งกันซื้อ)
		var lottos []models.Lotto
		if err := tx.
			Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("lotto_id IN ? AND status = ?", uniq, "sell").
			Order("lotto_id ASC").
			Find(&lottos).Error; err != nil {
			return err
		}

		// ตรวจสอบว่าใบที่ต้องการซื้อยังอยู่ไหม
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

		// รวมราคา + เตรียมรายการตอบกลับ
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

		// สร้างหัวบิล
		p := models.Purchase{
			UserID:     req.UserID,
			TotalPrice: totalPrice,
		}
		if err := tx.Create(&p).Error; err != nil {
			return err
		}
		purchaseID = p.PurchaseID

		// สร้างรายละเอียดบิล
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

		// เปลี่ยนสถานะเป็น sold
		if err := tx.Model(&models.Lotto{}).
			Where("lotto_id IN ?", uniq).
			Update("status", "sold").Error; err != nil {
			return err
		}

		// ✅ หักเงิน wallet
		if user.Wallet < totalPrice {
			return errors.New("ยอดเงินในกระเป๋าไม่เพียงพอ")
		}
		if err := tx.Model(&models.User{}).
			Where("user_id = ?", req.UserID).
			Update("wallet", gorm.Expr("wallet - ?", totalPrice)).Error; err != nil {
			return err
		}

		// ✅ ดึง wallet ใหม่หลังหัก
		if err := tx.Model(&models.User{}).
			Select("wallet").
			Where("user_id = ?", req.UserID).
			First(&user).Error; err != nil {
			return err
		}

		return nil // <-- จบ Transaction ตรงนี้
	})
	// ---------- จบ Transaction ----------

	// ---------- ตอบกลับ ----------
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
		"wallet":      user.Wallet, // ✅ กระเป๋าหลังหัก
	})
}

// ---------- ดึงรายการสลากที่ผู้ใช้ซื้อ ----------
/*
GET /users/:user_id/purchases
Response: lotto_id, lotto_name (= lotto_number), status (จาก purchases_detail)
*/
func ListPurchasedLottosByUser(c *gin.Context, db *gorm.DB) {
	uidStr := c.Param("user_id")
	if uidStr == "" {
		uidStr = c.Query("user_id")
	}
	uid, err := strconv.Atoi(uidStr)
	if err != nil || uid <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "invalid user_id"})
		return
	}

	type Row struct {
		LottoID   uint   `json:"lotto_id"`
		LottoName string `json:"lotto_name"`
		Status    string `json:"status"`
	}
	var rows []Row

	if err := db.Table("purchases_detail AS pd").
		Select("l.lotto_id, l.lotto_number AS lotto_name, pd.status").
		Joins("JOIN purchases p ON p.purchase_id = pd.purchase_id").
		Joins("JOIN lotto l ON l.lotto_id = pd.lotto_id").
		Where("p.user_id = ?", uid).
		Order("pd.pd_id ASC").
		Scan(&rows).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"total":  len(rows),
		"data":   rows,
	})
}
