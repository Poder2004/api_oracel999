package models

// ตาราง User
type User struct {
	UserID   uint    `json:"user_id" gorm:"column:user_id;primaryKey;autoIncrement"`
	Username string  `json:"username" gorm:"column:username;type:varchar(255);not null"`
	Email    string  `json:"email"    gorm:"column:email;type:varchar(255);not null"`
	Password string  `json:"password" gorm:"column:password;type:varchar(255);not null"`
	Role     string  `json:"role"     gorm:"column:role;type:enum('member','admin');not null;default:'member'"`
	Wallet   float64 `json:"wallet"   gorm:"column:wallet;type:decimal(10,2);default:0"`

	// relations
	Purchases []Purchase `json:"-" gorm:"foreignKey:UserID;references:UserID"`
}

func (User) TableName() string { return "users" }
