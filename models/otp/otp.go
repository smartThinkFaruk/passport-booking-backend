package otp

import (
	"time"
)

// OTP represents an OTP record for phone verification
type OTP struct {
	ID            uint       `gorm:"primaryKey;autoIncrement" json:"id"`
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
}

// OTPPurpose represents the purpose of the OTP
type OTPPurpose string

const (
	OTPPurposeDeliveryApplyPhone   OTPPurpose = "delivery_phone_apply_verification"
	OTPPurposeDeliveryConfirmPhone OTPPurpose = "delivery_phone_confirm_verification"
)

// IsExpired checks if the OTP has expired
func (o *OTP) IsExpired() bool {
	return time.Now().After(o.ExpiresAt)
}

// IsValid checks if the OTP is valid (not used and not expired)
func (o *OTP) IsValid() bool {
	return !o.IsUsed && !o.IsExpired() && !o.IsBlocked
}

// IsBlocked checks if the OTP is blocked due to too many retry attempts
func (o *OTP) IsCurrentlyBlocked() bool {
	if !o.IsBlocked {
		return false
	}

	// If BlockedUntil is nil, it's permanently blocked
	if o.BlockedUntil == nil {
		return true
	}

	// Check if the block period has expired
	if time.Now().After(*o.BlockedUntil) {
		return false
	}

	return true
}

// CanRetry checks if the OTP can be retried
func (o *OTP) CanRetry() bool {
	return !o.IsUsed && !o.IsExpired() && !o.IsCurrentlyBlocked() && o.RetryCount < o.MaxRetries
}

// IncrementRetry increments the retry count and blocks if max retries exceeded
func (o *OTP) IncrementRetry() {
	now := time.Now()
	o.RetryCount++
	o.LastAttemptAt = &now

	// Block if max retries exceeded
	if o.RetryCount >= o.MaxRetries {
		o.IsBlocked = true
		// Block for 15 minutes after max retries
		blockUntil := now.Add(15 * time.Minute)
		o.BlockedUntil = &blockUntil
	}
}

// Reset resets the OTP retry state (used when unblocking)
func (o *OTP) Reset() {
	o.RetryCount = 0
	o.IsBlocked = false
	o.BlockedUntil = nil
	o.LastAttemptAt = nil
}
