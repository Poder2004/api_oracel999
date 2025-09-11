package models

//  ตาราง Purchases
type Purchase struct {
	PurchaseID uint    `json:"purchase_id" gorm:"column:purchase_id;primaryKey;autoIncrement"`
	UserID     uint    `json:"user_id"      gorm:"column:user_id;not null;index"`
	TotalPrice float64 `json:"total_price"  gorm:"column:total_price;type:decimal(10,2);not null"`

	// relations
	User             *User            `json:"-" gorm:"foreignKey:UserID;references:UserID;constraint:OnUpdate:RESTRICT,OnDelete:RESTRICT"`
	PurchasesDetails []PurchaseDetail `json:"-" gorm:"foreignKey:PurchaseID;references:PurchaseID"`
}

func (Purchase) TableName() string { return "purchases" }
