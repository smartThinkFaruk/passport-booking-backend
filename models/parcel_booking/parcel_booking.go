package parcel_booking

import (
	"passport-booking/models/user"
	"time"
)

// ParcelBooking represents the main booking record.
type ParcelBooking struct {
	ID     uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID uint      `gorm:"not null;index"           json:"user_id"`
	User   user.User `gorm:"foreignKey:UserID" json:"user"`

	InsuranceID   *uint   `json:"insurance_id"`
	RpoAddress    string  `gorm:"type:text;not null"       json:"rpo_address"`
	Phone         string  `gorm:"size:20;not null"         json:"phone"`
	PostCode      string  `gorm:"size:20;index"            json:"post_code"`
	RpoName       string  `gorm:"size:120;not null"        json:"rpo_name"`
	Barcode       string  `gorm:"size:50;uniqueIndex"      json:"barcode"`
	TotalCharge   float64 `gorm:"type:decimal(10,2)"       json:"total_charge"`
	ServiceType   string  `gorm:"size:50;not null"         json:"service_type"`
	VasType       string  `gorm:"size:50"                  json:"vas_type"`
	Price         float64 `gorm:"type:decimal(10,2)"       json:"price"`
	Insured       bool    `gorm:"default:false"            json:"insured"`
	CurrentStatus string  `gorm:"size:50;not null;column:current_status" json:"current_status"`
	PushStatus    int     `gorm:"default:0"                json:"push_status"`
	UpdatedBy     string  `gorm:"type:varchar(255)" json:"updated_by,omitempty"`

	CreatedAt     time.Time  `gorm:"autoCreateTime"           json:"created_at"`
	PendingDate   *time.Time `json:"pending_date"`
	BookingDate   *time.Time `json:"booking_date"`
	DeliveredDate *time.Time `json:"delivered_date"`
	UpdatedAt     time.Time  `gorm:"autoUpdateTime"           json:"updated_at"`
}

type ParcelBookingStatus string

const (
	ParcelBookingStatusInitial   ParcelBookingStatus = "initial"
	ParcelBookingStatusPending   ParcelBookingStatus = "pending"
	ParcelBookingStatusBooked    ParcelBookingStatus = "booked"
	ParcelBookingStatusReturn    ParcelBookingStatus = "return"
	ParcelBookingStatusDelivered ParcelBookingStatus = "delivered"
)
