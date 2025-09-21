package models

// ตาราง Reward
type Reward struct {
	RewardID   uint    `json:"reward_id"  gorm:"column:reward_id;primaryKey;autoIncrement"`
	LottoID    uint    `json:"lotto_id"   gorm:"column:lotto_id;not null;index"` // ใน DB มี UNIQUE(lotto_id) (one-to-one)
	PrizeMoney float64 `json:"prize_money" gorm:"column:prize_money;type:decimal(10,2);not null"`
	PrizeTier  int     `json:"prize_tier"  gorm:"column:prize_tier;not null"`

	// relations
	Lotto *Lotto `json:"-" gorm:"foreignKey:LottoID;references:LottoID;constraint:OnUpdate:RESTRICT,OnDelete:RESTRICT"`
}

func (Reward) TableName() string { return "rewards" }
