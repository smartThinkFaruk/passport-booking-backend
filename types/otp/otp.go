package otp

// SendOTPRequest represents the request payload for sending OTP
type SendOTPRequest struct {
	Phone   string `json:"phone" validate:"required,min=10,max=20"`
	Purpose string `json:"purpose" validate:"required,oneof=delivery_phone_verification registration login"`
}

// VerifyOTPRequest represents the request payload for verifying OTP
type VerifyOTPRequest struct {
	Phone   string `json:"phone" validate:"required,min=10,max=20"`
	OTPCode string `json:"otp_code" validate:"required,len=6"`
	Purpose string `json:"purpose" validate:"required,oneof=delivery_phone_verification registration login"`
}

// OTPResponse represents the response for OTP operations
type OTPResponse struct {
	Message   string `json:"message"`
	ExpiresAt string `json:"expires_at,omitempty"`
	Success   bool   `json:"success"`
}
