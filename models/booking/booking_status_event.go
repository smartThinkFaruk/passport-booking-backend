package booking

import (
	"time"
)

// BookingStatusEvent represents a status change event for a booking
type BookingStatusEvent struct {
	ID uint `gorm:"primaryKey;autoIncrement" json:"id"`

	// Foreign key for booking relationship
	BookingID uint    `gorm:"not null;index" json:"booking_id"`
	Booking   Booking `gorm:"foreignKey:BookingID" json:"booking"`

	Status    BookingStatus `gorm:"size:20;not null" json:"status"`
	CreatedBy string        `gorm:"type:varchar(255);not null" json:"created_by"`
	CreatedAt time.Time     `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time     `gorm:"autoUpdateTime" json:"updated_at"`
}

// TableName sets the table name for the BookingStatusEvent model
func (BookingStatusEvent) TableName() string {
	return "booking_status_events"
}
