package booking

import (
	"fmt"
	"passport-booking/models/otp"
	"passport-booking/utils"
)

// VerifyDeliveryPhoneRequest represents the request for verifying delivery phone for a booking
type VerifyDeliveryPhoneRequest struct {
	BookingID uint   `json:"booking_id" validate:"required"`
	Phone     string `json:"phone" validate:"required,phone"`
	Purpose   otp.OTPPurpose `json:"purpose" validate:"required"`
	OTPCode   string `json:"otp_code" validate:"required,len=6"`
}

func (r *VerifyDeliveryPhoneRequest) Validate() error {
	if r.BookingID == 0 {
		return fmt.Errorf("booking_id is required")
	}
	if r.Phone == "" {
		return fmt.Errorf("phone is required")
	}
	if !utils.ValidatePhoneNumber(r.Phone) {
		return fmt.Errorf("phone number is invalid")
	}
	if r.Purpose == "" {
		return fmt.Errorf("purpose is required")
	}
	// Validate purpose is one of the allowed values
	if r.Purpose != otp.OTPPurposeDeliveryApplyPhone && r.Purpose != otp.OTPPurposeDeliveryConfirmPhone {
		return fmt.Errorf("purpose must be either 'delivery_phone_apply_verification' or 'delivery_phone_confirm_verification'")		
		
	}
	if r.OTPCode == "" {
		return fmt.Errorf("otp_code is required")
	}
	if len(r.OTPCode) != 6 {
		return fmt.Errorf("otp_code must be exactly 6 characters")
	}
	return nil
}

// UpdateDeliveryPhoneRequest represents the request for updating delivery phone
type UpdateDeliveryPhoneRequest struct {
	BookingID     uint           `json:"booking_id" validate:"required"`
	DeliveryPhone string         `json:"delivery_phone" validate:"required,phone"`
	Purpose       otp.OTPPurpose `json:"purpose" validate:"required"`
}

// Validate validates the UpdateDeliveryPhoneRequest fields
func (r *UpdateDeliveryPhoneRequest) Validate() error {
	if r.BookingID == 0 {
		return fmt.Errorf("booking_id is required")
	}
	if r.DeliveryPhone == "" {
		return fmt.Errorf("delivery_phone is required")
	}
	if !utils.ValidatePhoneNumber(r.DeliveryPhone) {
		return fmt.Errorf("delivery_phone number is invalid")
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

// GetOTPRetryInfoRequest represents the request for getting OTP retry information
type GetOTPRetryInfoRequest struct {
	BookingID uint           `json:"booking_id" validate:"required"`
	Phone   string         `json:"phone" validate:"required,phone"`
	Purpose otp.OTPPurpose `json:"purpose" validate:"required"`
}

func (r *GetOTPRetryInfoRequest) Validate() error {
	if r.BookingID == 0 {
		return fmt.Errorf("booking_id is required")
	}
	if r.Phone != "" && !utils.ValidatePhoneNumber(r.Phone) {
		return fmt.Errorf("phone number is invalid")
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

// ResendOTPRequest represents the request for resending OTP
type ResendOTPRequest struct {
	BookingID uint           `json:"booking_id" validate:"required"`
	Phone     string         `json:"phone" validate:"required,phone"`
	Purpose   otp.OTPPurpose `json:"purpose" validate:"required"`
}

func (r *ResendOTPRequest) Validate() error {
	if r.BookingID == 0 {
		return fmt.Errorf("booking_id is required")
	}
	if r.Phone == "" {
		return fmt.Errorf("phone is required")
	}
	if !utils.ValidatePhoneNumber(r.Phone) {
		return fmt.Errorf("phone number is invalid")
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

// SendOTPForBookingRequest represents the request for sending OTP for a booking
type SendOTPForBookingRequest struct {
	BookingID uint   `json:"booking_id" validate:"required"`
	Phone     string `json:"phone" validate:"required,phone"`
}

func (r *SendOTPForBookingRequest) Validate() error {
	if r.BookingID == 0 {
		return fmt.Errorf("booking_id is required")
	}
	if r.Phone == "" {
		return fmt.Errorf("phone is required")
	}
	if !utils.ValidatePhoneNumber(r.Phone) {
		return fmt.Errorf("phone number is invalid")
	}
	return nil
}

// VerifyOTPForBookingRequest represents the request for verifying OTP for a booking
type VerifyOTPForBookingRequest struct {
	BookingID uint   `json:"booking_id" validate:"required"`
	Phone     string `json:"phone" validate:"required,phone"`
	OTPCode   string `json:"otp_code" validate:"required,len=6"`
}

func (r *VerifyOTPForBookingRequest) Validate() error {
	if r.BookingID == 0 {
		return fmt.Errorf("booking_id is required")
	}
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
	return nil
}

// GetBookingOTPStatusRequest represents the request for getting OTP status for a booking
type GetBookingOTPStatusRequest struct {
	BookingID uint   `json:"booking_id" validate:"required"`
	Phone     string `json:"phone" validate:"required,phone"`
}

func (r *GetBookingOTPStatusRequest) Validate() error {
	if r.BookingID == 0 {
		return fmt.Errorf("booking_id is required")
	}
	if r.Phone == "" {
		return fmt.Errorf("phone is required")
	}
	if !utils.ValidatePhoneNumber(r.Phone) {
		return fmt.Errorf("phone number is invalid")
	}
	return nil
}
