package otp

import (
	"passport-booking/models/booking"
	"time"
)

// OTPEvent represents an OTP event record mirroring OTP fields
type OTPEvent struct {
	ID uint `gorm:"primaryKey;autoIncrement" json:"id"`

	BookingID uint            `gorm:"not null" json:"booking_id"`
	Booking   booking.Booking `gorm:"foreignKey:BookingID" json:"booking"`

	Phone         string     `gorm:"type:varchar(20);not null;index" json:"phone"`
	OTPCode       string     `gorm:"column:otp_code;type:varchar(6);not null" json:"otp_code"`
	Purpose       OTPPurpose `gorm:"type:varchar(50);not null" json:"purpose"`
	IsUsed        bool       `gorm:"default:false" json:"is_used"`
	RetryCount    int        `gorm:"default:0" json:"retry_count"`
	MaxRetries    int        `gorm:"default:3" json:"max_retries"`
	IsBlocked     bool       `gorm:"default:false" json:"is_blocked"`
	BlockedUntil  *time.Time `gorm:"index" json:"blocked_until,omitempty"`
	LastAttemptAt *time.Time `gorm:"index" json:"last_attempt_at,omitempty"`
	ExpiresAt     time.Time  `gorm:"not null" json:"expires_at"`
	CreatedAt     time.Time  `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt     time.Time  `gorm:"autoUpdateTime" json:"updated_at"`

	EventType string `gorm:"type:varchar(50);not null" json:"event_type"` // created, verified, expired, etc.
}
