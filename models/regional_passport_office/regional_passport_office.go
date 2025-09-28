package regional_passport_office

import "time"

type RegionalPassportOffice struct {
	ID        uint      `gorm:"primaryKey;autoIncrement"`
	Code      string    `gorm:"size:20;not null;uniqueIndex"  json:"code"`
	Name      string    `gorm:"size:120;not null;index"       json:"name"`
	Address   string    `gorm:"type:text;not null"            json:"address"`
	Mobile    string    `gorm:"size:20;index"                 json:"mobile"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}
