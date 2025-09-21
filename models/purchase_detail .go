package models

// ตาราง Purchases_detail
type PurchaseDetail struct {
	PDID       uint   `json:"pd_id"       gorm:"column:pd_id;primaryKey;autoIncrement"`
	PurchaseID uint   `json:"purchase_id" gorm:"column:purchase_id;not null;index"`
	LottoID    uint   `json:"lotto_id"    gorm:"column:lotto_id;not null;index"` // ใน DB มี UNIQUE(lotto_id) อยู่แล้ว
	Status     string `json:"status"      gorm:"column:status;type:enum('ยัง','ถูก','ไม่ถูก');not null;default:'ยัง'"`
	CashIn     string `json:"cash_in"     gorm:"column:cash_in;type:enum('ซื้อ','ขึ้นเงิน');not null;default:'ซื้อ'"`

	// relations
	Purchase *Purchase `json:"-" gorm:"foreignKey:PurchaseID;references:PurchaseID;constraint:OnUpdate:RESTRICT,OnDelete:RESTRICT"`
	Lotto    *Lotto    `json:"-" gorm:"foreignKey:LottoID;references:LottoID;constraint:OnUpdate:RESTRICT,OnDelete:RESTRICT"`
}

func (PurchaseDetail) TableName() string { return "purchases_detail" }
