package booking

import (
	"passport-booking/models/user"
	"passport-booking/models/address"
	"time"
)

// Booking represents a booking record with user information and address
type Booking struct {
	ID                    uint    `gorm:"primaryKey;autoIncrement" json:"id"`

	// Foreign key for users relationship
	UserID                uint    `gorm:"not null" json:"user_id"`
	User 			      user.User `gorm:"foreignKey:UserID" json:"user"`
	
	AppOrOrderID          string  `gorm:"type:varchar(255);not null;unique" json:"app_or_order_id"`
	CurrentBagID          *string `gorm:"type:varchar(255)" json:"current_bag_id,omitempty"`
	Barcode               *string `gorm:"type:varchar(255)" json:"barcode,omitempty"`
	Name                  string  `gorm:"type:varchar(255);not null" json:"name"`
	FatherName            string  `gorm:"type:varchar(255);not null" json:"father_name"`
	MotherName            string  `gorm:"type:varchar(255);not null" json:"mother_name"`
	Phone                 string  `gorm:"type:varchar(20);not null" json:"phone"`
	Address               string  `gorm:"type:text;not null" json:"address"`
	EmergencyContactName  *string `gorm:"type:varchar(255)" json:"emergency_contact_name,omitempty"`
	EmergencyContactPhone *string `gorm:"type:varchar(20)" json:"emergency_contact_phone,omitempty"`

	// Foreign key for address relationship
	AddressID   uint            `gorm:"not null" json:"address_id"`
	AddressInfo address.Address `gorm:"foreignKey:AddressID" json:"address_info"`

	Status	  string    `gorm:"type:varchar(50);not null" json:"status"` 
	CreatedBy string    `gorm:"type:varchar(255);not null" json:"created_by"`
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedBy string    `gorm:"type:varchar(255)" json:"updated_by,omitempty"`
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updated_at"`
	DeletedAt *time.Time `gorm:"index" json:"deleted_at,omitempty"` // Soft delete field
}
