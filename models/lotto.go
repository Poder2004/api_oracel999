package models

// ตาราง Lotto
type Lotto struct {
	LottoID     uint    `json:"lotto_id"     gorm:"column:lotto_id;primaryKey;autoIncrement"`
	LottoNumber string  `json:"lotto_number" gorm:"column:lotto_number;type:varchar(6);not null"`
	Status      string  `json:"status"       gorm:"column:status;type:enum('sell','sold');not null;default:'sell'"`
	Price       float64 `json:"price"        gorm:"column:price;type:decimal(10,2);default:80"`
	CreatedBy   *uint   `json:"created_by"   gorm:"column:created_by;index:idx_lotto_created_by"`

	// relations
	Creator          *User            `json:"-" gorm:"foreignKey:CreatedBy;references:UserID;constraint:OnUpdate:RESTRICT,OnDelete:SET NULL"`
	Reward           *Reward          `json:"-" gorm:"foreignKey:LottoID;references:LottoID;constraint:OnUpdate:RESTRICT,OnDelete:RESTRICT"`
	PurchasesDetails []PurchaseDetail `json:"-" gorm:"foreignKey:LottoID;references:LottoID"`
}

func (Lotto) TableName() string { return "lotto" }
