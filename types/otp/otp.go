package otp

import (
	"fmt"
	"passport-booking/utils"
)

// SendOTPRequest represents the request payload for sending OTP
type SendOTPRequest struct {
	Phone   string `json:"phone" validate:"required,phone"`
	Purpose string `json:"purpose" validate:"required,oneof=delivery_phone_verification registration login"`
}

// Validate validates the SendOTPRequest fields
func (r *SendOTPRequest) Validate() error {
	if r.Phone == "" {
		return fmt.Errorf("phone is required")
	}
	if !utils.ValidatePhoneNumber(r.Phone) {
		return fmt.Errorf("phone number is invalid")
	}
	if r.Purpose == "" {
		return fmt.Errorf("purpose is required")
	}
	if r.Purpose != "delivery_phone_verification" && r.Purpose != "registration" && r.Purpose != "login" {
		return fmt.Errorf("purpose must be one of: delivery_phone_verification, registration, login")
	}
	return nil
}

// VerifyOTPRequest represents the request payload for verifying OTP
type VerifyOTPRequest struct {
	Phone   string `json:"phone" validate:"required,phone"`
	OTPCode string `json:"otp_code" validate:"required,len=6"`
	Purpose string `json:"purpose" validate:"required,oneof=delivery_phone_verification registration login"`
}

// Validate validates the VerifyOTPRequest fields
func (r *VerifyOTPRequest) Validate() error {
	if r.Phone == "" {
		return fmt.Errorf("phone is required")
	}
	if !utils.ValidatePhoneNumber(r.Phone) {
		return fmt.Errorf("phone number is invalid")
	}
	if r.OTPCode == "" {
		return fmt.Errorf("otp_code is required")
	}
	if len(r.OTPCode) != 6 {
		return fmt.Errorf("otp_code must be exactly 6 characters")
	}
	if r.Purpose == "" {
		return fmt.Errorf("purpose is required")
	}
	if r.Purpose != "delivery_phone_verification" && r.Purpose != "registration" && r.Purpose != "login" {
		return fmt.Errorf("purpose must be one of: delivery_phone_verification, registration, login")
	}
	return nil
}

// OTPResponse represents the response for OTP operations
type OTPResponse struct {
	Message           string `json:"message"`
	ExpiresAt         string `json:"expires_at,omitempty"`
	Success           bool   `json:"success"`
	RemainingAttempts *int   `json:"remaining_attempts,omitempty"`
	IsBlocked         *bool  `json:"is_blocked,omitempty"`
	NewOTPSent        bool   `json:"new_otp_sent,omitempty"`
}
