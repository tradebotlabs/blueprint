package model

// BIG NOTE : primary key should be first field for grom
// Example Model
type Account struct {
	AccountId int64  `gorm:"primaryKey;autoIncrement:true" json:"account_id"`
	NameEn    string `json:"name_en"`
	NameAr    string `json:"name_ar"`
	Contact   string `json:"contact"`
	Address   string `json:"address"`
	Tel       string `json:"tel"`
	Mobile    string `json:"mobile"`
	Email     string `json:"email"`
	Web       string `json:"web"`
	CreateAt  int64  `gorm:"autoUpdateTime:milli," json:"create_at"`
	UpdateAt  int64  `gorm:"autoUpdateTime:milli," json:"update_at"`
}