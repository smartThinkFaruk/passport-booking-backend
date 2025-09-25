package delivery

import (
	"fmt"
	"passport-booking/models/otp"
)

type DeliveryPhoneSendOtpRequest struct {
	BookingID string         `json:"booking_id" validate:"required"`
	Purpose   otp.OTPPurpose `json:"purpose" validate:"required"`
}

// Validate validates the DeliveryPhoneSendOtpRequest fields
func (r *DeliveryPhoneSendOtpRequest) Validate() error {
	// Validate BookingID is not empty
	if r.BookingID == "" {
		return fmt.Errorf("booking_id is required")
	}

	if r.Purpose == "" {
		return fmt.Errorf("purpose is required")
	}

	// Validate purpose is one of the allowed values
	if r.Purpose != otp.OTPPurposeDeliveryApplyPhone && r.Purpose != otp.OTPPurposeDeliveryConfirmPhone {
		return fmt.Errorf("purpose must be either 'delivery_phone_apply_verification' or 'delivery_phone_confirm_verification'")
	}
	return nil
}

type VerifyDeliveryPhoneRequest struct {
	BookingID string         `json:"booking_id" validate:"required"`
	OTPCode   string         `json:"otp_code" validate:"required"`
	Purpose   otp.OTPPurpose `json:"purpose" validate:"required"`
}

// Validate validates the VerifyDeliveryPhoneRequest fields
func (r *VerifyDeliveryPhoneRequest) Validate() error {
	if r.BookingID == "" {
		return fmt.Errorf("booking_id is required")
	}

	if r.OTPCode == "" {
		return fmt.Errorf("otp_code is required")
	}

	if r.Purpose == "" {
		return fmt.Errorf("purpose is required")
	}

	// Validate purpose is one of the allowed values
	if r.Purpose != otp.OTPPurposeDeliveryApplyPhone && r.Purpose != otp.OTPPurposeDeliveryConfirmPhone {
		return fmt.Errorf("purpose must be either 'delivery_phone_apply_verification' or 'delivery_phone_confirm_verification'")
	}

	return nil
}

type VerifyApplicationIDRequest struct {
	BookingID     string `json:"booking_id" validate:"required"`
	ApplicationID string `json:"application_id" validate:"required"`
}

// Validate validates the VerifyApplicationIDRequest fields
func (r *VerifyApplicationIDRequest) Validate() error {
	if r.BookingID == "" {
		return fmt.Errorf("booking_id is required")
	}

	if r.ApplicationID == "" {
		return fmt.Errorf("application_id is required")
	}

	return nil
}
