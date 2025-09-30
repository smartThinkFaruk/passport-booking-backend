package parcel_booking

import (
	"time"

	"passport-booking/models/user"
)

// ParcelBookingStatusEvent tracks status history for ParcelBooking.
type ParcelBookingStatusEvent struct {
	ID              uint          `gorm:"primaryKey;autoIncrement" json:"id"`
	ParcelBookingID uint          `gorm:"not null;index"           json:"parcel_booking_id"`
	ParcelBooking   ParcelBooking `gorm:"foreignKey:ParcelBookingID" json:"-"`

	Status    string    `gorm:"size:50;not null" json:"status"` // e.g. "Booked", "Delivered"
	CreatedBy uint      `gorm:"not null;index"   json:"created_by"`
	User      user.User `gorm:"foreignKey:CreatedBy" json:"user"`

	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}
