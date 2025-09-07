package address

import (
	"time"
)

// Address represents sender or recipient address information
type Address struct {
	ID             uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	Division       *string   `gorm:"size:255" json:"division,omitempty"`
	District       *string   `gorm:"size:255" json:"district,omitempty"`
	PoliceStation  *string   `gorm:"size:255" json:"police_station,omitempty"`
	PostOffice     *string   `gorm:"size:255" json:"post_office,omitempty"`
	PostOfficeCode *string   `gorm:"size:255" json:"post_office_code,omitempty"`
	StreetAddress  *string   `gorm:"size:255" json:"street_address,omitempty"`
	CreatedAt      time.Time `gorm:"autoCreateTime" json:"created_at"`
}
