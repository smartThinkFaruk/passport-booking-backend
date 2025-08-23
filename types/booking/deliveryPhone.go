package booking

// VerifyDeliveryPhoneRequest represents the request for verifying delivery phone for a booking
type VerifyDeliveryPhoneRequest struct {
	BookingID uint   `json:"booking_id" validate:"required"`
	Phone     string `json:"phone" validate:"required,min=10,max=20"`
	OTPCode   string `json:"otp_code" validate:"required,len=6"`
}

// UpdateDeliveryPhoneRequest represents the request for updating delivery phone
type UpdateDeliveryPhoneRequest struct {
	BookingID     uint   `json:"booking_id" validate:"required"`
	DeliveryPhone string `json:"delivery_phone" validate:"required,min=10,max=20"`
}

// GetOTPRetryInfoRequest represents the request for getting OTP retry information
type GetOTPRetryInfoRequest struct {
	Phone string `json:"phone" validate:"required,min=10,max=20"`
}

// ResendOTPRequest represents the request for resending OTP
type ResendOTPRequest struct {
	BookingID uint   `json:"booking_id" validate:"required"`
	Phone     string `json:"phone" validate:"required,min=10,max=20"`
}
