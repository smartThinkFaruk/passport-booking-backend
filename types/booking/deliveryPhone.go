package booking

import "fmt"

// VerifyDeliveryPhoneRequest represents the request for verifying delivery phone for a booking
type VerifyDeliveryPhoneRequest struct {
	BookingID uint   `json:"booking_id" validate:"required"`
	Phone     string `json:"phone" validate:"required,min=10,max=20"`
	OTPCode   string `json:"otp_code" validate:"required,len=6"`
}

func (r *VerifyDeliveryPhoneRequest) Validate() error {
	if r.BookingID == 0 {
		return fmt.Errorf("booking_id is required")
	}
	if r.Phone == "" {
		return fmt.Errorf("phone is required")
	}
	if len(r.Phone) < 10 {
		return fmt.Errorf("phone must be at least 10 characters")
	}
	if len(r.Phone) > 20 {
		return fmt.Errorf("phone must not exceed 20 characters")
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
	BookingID     uint   `json:"booking_id" validate:"required"`
	DeliveryPhone string `json:"delivery_phone" validate:"required,min=10,max=20"`
}

// Validate validates the UpdateDeliveryPhoneRequest fields
func (r *UpdateDeliveryPhoneRequest) Validate() error {
	if r.BookingID == 0 {
		return fmt.Errorf("booking_id is required")
	}
	if r.DeliveryPhone == "" {
		return fmt.Errorf("delivery_phone is required")
	}
	if len(r.DeliveryPhone) < 10 {
		return fmt.Errorf("delivery_phone must be at least 10 characters")
	}
	if len(r.DeliveryPhone) > 20 {
		return fmt.Errorf("delivery_phone must not exceed 20 characters")
	}
	return nil
}

// GetOTPRetryInfoRequest represents the request for getting OTP retry information
type GetOTPRetryInfoRequest struct {
	Phone string `json:"phone" validate:"required,min=10,max=20"`
}
func (r *GetOTPRetryInfoRequest) Validate() error {
	if r.Phone == "" {
		return fmt.Errorf("phone is required")
	}
	if len(r.Phone) < 10 {
		return fmt.Errorf("phone must be at least 10 characters")
	}
	if len(r.Phone) > 20 {
		return fmt.Errorf("phone must not exceed 20 characters")
	}
	return nil
}

// ResendOTPRequest represents the request for resending OTP
type ResendOTPRequest struct {
	BookingID uint   `json:"booking_id" validate:"required"`
	Phone     string `json:"phone" validate:"required,min=10,max=20"`
}

func (r *ResendOTPRequest) Validate() error {
	if r.BookingID == 0 {
		return fmt.Errorf("booking_id is required")
	}
	if r.Phone == "" {
		return fmt.Errorf("phone is required")
	}
	if len(r.Phone) < 10 {
		return fmt.Errorf("phone must be at least 10 characters")
	}
	if len(r.Phone) > 20 {
		return fmt.Errorf("phone must not exceed 20 characters")
	}
	return nil
}

// SendOTPForBookingRequest represents the request for sending OTP for a booking
type SendOTPForBookingRequest struct {
	BookingID uint   `json:"booking_id" validate:"required"`
	Phone     string `json:"phone" validate:"required,min=10,max=20"`
}

func (r *SendOTPForBookingRequest) Validate() error {
	if r.BookingID == 0 {
		return fmt.Errorf("booking_id is required")
	}
	if r.Phone == "" {
		return fmt.Errorf("phone is required")
	}
	if len(r.Phone) < 10 {
		return fmt.Errorf("phone must be at least 10 characters")
	}
	if len(r.Phone) > 20 {
		return fmt.Errorf("phone must not exceed 20 characters")
	}
	return nil
}

// VerifyOTPForBookingRequest represents the request for verifying OTP for a booking
type VerifyOTPForBookingRequest struct {
	BookingID uint   `json:"booking_id" validate:"required"`
	Phone     string `json:"phone" validate:"required,min=10,max=20"`
	OTPCode   string `json:"otp_code" validate:"required,len=6"`
}

func (r *VerifyOTPForBookingRequest) Validate() error {
	if r.BookingID == 0 {
		return fmt.Errorf("booking_id is required")
	}
	if r.Phone == "" {
		return fmt.Errorf("phone is required")
	}
	if len(r.Phone) < 10 {
		return fmt.Errorf("phone must be at least 10 characters")
	}
	if len(r.Phone) > 20 {
		return fmt.Errorf("phone must not exceed 20 characters")
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
	Phone     string `json:"phone" validate:"required,min=10,max=20"`
}

func (r *GetBookingOTPStatusRequest) Validate() error {
	if r.BookingID == 0 {
		return fmt.Errorf("booking_id is required")
	}
	if r.Phone == "" {
		return fmt.Errorf("phone is required")
	}
	if len(r.Phone) < 10 {
		return fmt.Errorf("phone must be at least 10 characters")
	}
	if len(r.Phone) > 20 {
		return fmt.Errorf("phone must not exceed 20 characters")
	}
	return nil
}
