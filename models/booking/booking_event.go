// models/booking/booking_event.go
package booking

import (
	"passport-booking/models/address"
	"passport-booking/models/user"
	"time"
)

type BookingEvent struct {
	ID uint `gorm:"primaryKey;autoIncrement" json:"id"`

	// Who owns the booking
	UserID uint      `gorm:"not null" json:"user_id"`
	User   user.User `gorm:"foreignKey:UserID" json:"user"`

	// DO NOT make this unique here (events are many per booking)
	AppOrOrderID string  `gorm:"type:varchar(255);not null;index" json:"app_or_order_id"`
	CurrentBagID *string `gorm:"type:varchar(255)" json:"current_bag_id,omitempty"`
	Barcode      *string `gorm:"type:varchar(255)" json:"barcode,omitempty"`
	Name         string  `gorm:"type:varchar(255);not null" json:"name"`
	FatherName   string  `gorm:"type:varchar(255);not null" json:"father_name"`
	MotherName   string  `gorm:"type:varchar(255);not null" json:"mother_name"`
	Phone        string  `gorm:"type:varchar(20);not null" json:"phone"`
	DeliveryPhone *string `gorm:"type:varchar(20)" json:"delivery_phone"`

	// keep field names consistent with Booking
	DeliveryPhoneAppliedVerified       bool    `gorm:"default:false" json:"delivery_phone_applied_verified"`
	DeliveryPhoneAppliedOTPEncrypted   *string `gorm:"column:delivery_phone_applied_otp_encrypted;type:text" json:"delivery_phone_applied_otp_encrypted,omitempty"`
	DeliveryPhoneConfirmedVerified     bool    `gorm:"default:false" json:"delivery_phone_confirmed_verified"`
	DeliveryPhoneConfirmedOTPEncrypted *string `gorm:"column:delivery_phone_confirmed_otp_encrypted;type:text" json:"delivery_phone_confirmed_otp_encrypted,omitempty"`

	Address               string  `gorm:"type:text;not null" json:"address"`
	EmergencyContactName  *string `gorm:"type:varchar(255)" json:"emergency_contact_name,omitempty"`
	EmergencyContactPhone *string `gorm:"type:varchar(20)" json:"emergency_contact_phone,omitempty"`
	DeliveryBranchCode    *string `gorm:"type:varchar(100)" json:"delivery_branch_code,omitempty"`
	// Foreign key for address relationship - nullable for two-step booking process

	DeliveryAddressID *uint            `gorm:"" json:"delivery_address_id,omitempty"`
	DeliveryAddress   *address.Address `gorm:"foreignKey:DeliveryAddressID" json:"delivery_address,omitempty"`

	Status      BookingStatus `gorm:"size:20;not null;default:initial" json:"status"`
	BookingType BookingType   `gorm:"size:20" json:"booking_type"` // "agent" or "customer"
	BookingDate time.Time     `gorm:"" json:"booking_date"`
	EventType string `gorm:"type:varchar(50);not null;index" json:"event_type"` // created, updated, delivery_phone_send_otp, phone_applied_verified, otp_resent, etc.
	CreatedBy   string        `gorm:"type:varchar(255);not null" json:"created_by"`
	CreatedAt   time.Time     `gorm:"autoCreateTime" json:"created_at"`
	UpdatedBy   string        `gorm:"type:varchar(255)" json:"updated_by,omitempty"`
	UpdatedAt   time.Time     `gorm:"autoUpdateTime" json:"updated_at"`
	DeletedAt   *time.Time    `gorm:"index" json:"deleted_at,omitempty"`
}
