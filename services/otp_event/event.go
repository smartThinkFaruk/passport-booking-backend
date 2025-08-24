package otp_event

import (
	"passport-booking/models/otp"

	"gorm.io/gorm"
)

// SnapshotOTPToEvent writes a full snapshot of an OTP row into OTPEvent with the given event type.
func SnapshotOTPToEvent(tx *gorm.DB, o *otp.OTP, eventType string) error {
	// Make sure related booking is present for event row
	// If caller already preloaded, this will be filled; else we fetch minimal required ids.
	if err := tx.Preload("Booking").First(o, o.ID).Error; err != nil {
		return err
	}

	ev := otp.OTPEvent{
		BookingID:     o.BookingID,
		Booking:       o.Booking, // optional; gorm will set by ID
		Phone:         o.Phone,
		OTPCode:       o.OTPCode,
		Purpose:       o.Purpose,
		IsUsed:        o.IsUsed,
		RetryCount:    o.RetryCount,
		MaxRetries:    o.MaxRetries,
		IsBlocked:     o.IsBlocked,
		BlockedUntil:  o.BlockedUntil,
		LastAttemptAt: o.LastAttemptAt,
		ExpiresAt:     o.ExpiresAt,
		CreatedAt:     o.CreatedAt,
		UpdatedAt:     o.UpdatedAt,
		EventType:     eventType,
	}

	return tx.Create(&ev).Error
}
