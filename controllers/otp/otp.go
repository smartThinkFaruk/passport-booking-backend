package otp

import (
	"passport-booking/logger"
	"passport-booking/models/otp"
	otpService "passport-booking/services/otp"
	"passport-booking/types"
	otpTypes "passport-booking/types/otp"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// Controller handles OTP-related HTTP requests
type Controller struct {
	DB         *gorm.DB
	Logger     *logger.AsyncLogger
	OTPService *otpService.Service
}

// NewOTPController creates a new OTP controller
func NewOTPController(db *gorm.DB, asyncLogger *logger.AsyncLogger) *Controller {
	return &Controller{
		DB:         db,
		Logger:     asyncLogger,
		OTPService: otpService.NewOTPService(db),
	}
}

// SendOTP sends an OTP to the provided phone number
func (oc *Controller) SendOTP(c *fiber.Ctx) error {
	var req otpTypes.SendOTPRequest
	if err := c.BodyParser(&req); err != nil {
		logger.Error("Failed to parse request body", err)
		return c.Status(fiber.StatusBadRequest).JSON(types.ApiResponse{
			Status:  fiber.StatusBadRequest,
			Message: "Invalid request body",
			Data:    nil,
		})
	}

	// Validate purpose
	var purpose otp.OTPPurpose
	switch req.Purpose {
	case "delivery_phone_apply_verification":
		purpose = otp.OTPPurposeDeliveryApplyPhone
	case "delivery_phone_confirm_verification":
		purpose = otp.OTPPurposeDeliveryConfirmPhone
	default:
		return c.Status(fiber.StatusBadRequest).JSON(types.ApiResponse{
			Status:  fiber.StatusBadRequest,
			Message: "Invalid OTP purpose",
			Data:    nil,
		})
	}

	// Check if there's already a valid OTP
	existingOTP, err := oc.OTPService.GetOTPStatus(req.Phone, purpose)
	if err != nil {
		logger.Error("Failed to check existing OTP", err)
		return c.Status(fiber.StatusInternalServerError).JSON(types.ApiResponse{
			Status:  fiber.StatusInternalServerError,
			Message: "Internal server error",
			Data:    nil,
		})
	}

	if existingOTP != nil {
		return c.Status(fiber.StatusTooManyRequests).JSON(types.ApiResponse{
			Status:  fiber.StatusTooManyRequests,
			Message: "OTP already sent. Please wait before requesting a new one.",
			Data: otpTypes.OTPResponse{
				Message:   "OTP already sent",
				ExpiresAt: existingOTP.ExpiresAt.Format("2006-01-02 15:04:05"),
				Success:   false,
			},
		})
	}

	// Send new OTP
	otpRecord, err := oc.OTPService.SendOTP(req.Phone, purpose)
	if err != nil {
		logger.Error("Failed to send OTP", err)
		return c.Status(fiber.StatusInternalServerError).JSON(types.ApiResponse{
			Status:  fiber.StatusInternalServerError,
			Message: "Failed to send OTP",
			Data:    nil,
		})
	}

	return c.Status(fiber.StatusOK).JSON(types.ApiResponse{
		Status:  fiber.StatusOK,
		Message: "OTP sent successfully",
		Data: otpTypes.OTPResponse{
			Message:   "OTP sent to your phone number",
			ExpiresAt: otpRecord.ExpiresAt.Format("2006-01-02 15:04:05"),
			Success:   true,
		},
	})
}

// VerifyOTP verifies the provided OTP
func (oc *Controller) VerifyOTP(c *fiber.Ctx) error {
	var req otpTypes.VerifyOTPRequest
	if err := c.BodyParser(&req); err != nil {
		logger.Error("Failed to parse request body", err)
		return c.Status(fiber.StatusBadRequest).JSON(types.ApiResponse{
			Status:  fiber.StatusBadRequest,
			Message: "Invalid request body",
			Data:    nil,
		})
	}

	// Validate purpose
	var purpose otp.OTPPurpose
	switch req.Purpose {
	case "delivery_phone_apply_verification":
		purpose = otp.OTPPurposeDeliveryApplyPhone
	case "delivery_phone_confirm_verification":
		purpose = otp.OTPPurposeDeliveryConfirmPhone
	default:
		return c.Status(fiber.StatusBadRequest).JSON(types.ApiResponse{
			Status:  fiber.StatusBadRequest,
			Message: "Invalid OTP purpose",
			Data:    nil,
		})
	}

	// Verify OTP
	isValid, err := oc.OTPService.VerifyOTP(req.Phone, req.OTPCode, purpose)
	if err != nil {
		logger.Error("Failed to verify OTP", err)
		return c.Status(fiber.StatusInternalServerError).JSON(types.ApiResponse{
			Status:  fiber.StatusInternalServerError,
			Message: "Internal server error",
			Data:    nil,
		})
	}

	if !isValid {
		return c.Status(fiber.StatusBadRequest).JSON(types.ApiResponse{
			Status:  fiber.StatusBadRequest,
			Message: "Invalid or expired OTP",
			Data: otpTypes.OTPResponse{
				Message: "Invalid or expired OTP",
				Success: false,
			},
		})
	}

	return c.Status(fiber.StatusOK).JSON(types.ApiResponse{
		Status:  fiber.StatusOK,
		Message: "OTP verified successfully",
		Data: otpTypes.OTPResponse{
			Message: "OTP verified successfully",
			Success: true,
		},
	})
}
