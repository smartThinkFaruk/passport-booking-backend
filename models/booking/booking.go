package booking

import (
	"passport-booking/models/address"
	"passport-booking/models/user"
	"time"
)

// Booking represents a booking record with user information and address
type Booking struct {
	ID uint `gorm:"primaryKey;autoIncrement" json:"id"`

	// Foreign key for users relationship
	UserID uint      `gorm:"not null" json:"user_id"`
	User   user.User `gorm:"foreignKey:UserID" json:"user"`

	AppOrOrderID string  `gorm:"type:varchar(255);not null;unique" json:"app_or_order_id"`
	CurrentBagID *string `gorm:"type:varchar(255);index" json:"current_bag_id,omitempty"`
	Barcode      *string `gorm:"type:varchar(255)" json:"barcode,omitempty"`
	Name         string  `gorm:"type:varchar(255);not null" json:"name"`
	FatherName   string  `gorm:"type:varchar(255);not null" json:"father_name"`
	MotherName   string  `gorm:"type:varchar(255);not null" json:"mother_name"`
	Phone        string  `gorm:"type:varchar(20);not null" json:"phone"`

	DeliveryPhone                      *string `gorm:"type:varchar(20)" json:"delivery_phone"`
	DeliveryPhoneAppliedVerified       bool    `gorm:"default:false" json:"delivery_phone_applied_verified"`
	DeliveryPhoneAppliedOTPEncrypted   *string `gorm:"column:delivery_phone_apply_otp_encrypted;type:text" json:"delivery_phone_apply_otp_encrypted,omitempty"`
	DeliveryPhoneConfirmedVerified     bool    `gorm:"default:false" json:"delivery_phone_confirmed_verified"`
	DeliveryApplicationIDVerified      bool    `gorm:"default:false" json:"delivery_application_id_verified"`
	DeliveryPhoneConfirmedOTPEncrypted *string `gorm:"column:delivery_phone_confirm_otp_encrypted;type:text" json:"delivery_phone_confirm_otp_encrypted,omitempty"`

	Address               string  `gorm:"type:text;not null" json:"address"`
	EmergencyContactName  *string `gorm:"type:varchar(255)" json:"emergency_contact_name,omitempty"`
	EmergencyContactPhone *string `gorm:"type:varchar(20)" json:"emergency_contact_phone,omitempty"`
	DeliveryBranchCode    *string `gorm:"type:varchar(100)" json:"delivery_branch_code,omitempty"`
	// Foreign key for address relationship
	DeliveryAddressID *uint            `json:"delivery_address_id,omitempty"`
	DeliveryAddress   *address.Address `gorm:"foreignKey:DeliveryAddressID" json:"delivery_address,omitempty"`

	Status      BookingStatus `gorm:"size:30;not null;default:initial;index" json:"status"`
	BookingType BookingType   `gorm:"size:20;index" json:"booking_type"` // "agent" or "customer"
	BookingDate time.Time     `gorm:"autoCreateTime" json:"booking_date"`
	CreatedBy   string        `gorm:"type:varchar(255);not null" json:"created_by"`
	CreatedAt   time.Time     `gorm:"autoCreateTime;index" json:"created_at"`
	UpdatedBy   string        `gorm:"type:varchar(255)" json:"updated_by,omitempty"`
	UpdatedAt   time.Time     `gorm:"autoUpdateTime" json:"updated_at"`
	DeletedAt   *time.Time    `gorm:"index" json:"deleted_at,omitempty"` // Soft delete field
}

// BookingStatus represents the status of a booking
type BookingStatus string

const (
	BookingStatusInitial              BookingStatus = "initial"
	BookingStatusPreBooked            BookingStatus = "pre_booked"
	BookingStatusBooked               BookingStatus = "booked"
	BookingStatusReceivedByPostman    BookingStatus = "received_by_postman"
	BookingStatusReceivedByPostMaster BookingStatus = "received_by_postmaster"
	BookingStatusReturn               BookingStatus = "return"
	BookingStatusDelivered            BookingStatus = "delivered"
)

type BookingType string

const (
	BookingTypeAgent    BookingType = "agent"
	BookingTypeCustomer BookingType = "customer"
)
