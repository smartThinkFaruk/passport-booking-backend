package otp

import (
	"fmt"
	"passport-booking/logger"
	bookingModel "passport-booking/models/booking"
	"passport-booking/models/otp"
	otpService "passport-booking/services/otp"
	"passport-booking/types"
	bookingTypes "passport-booking/types/booking"
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
	isValid, otpRecord, err := oc.OTPService.VerifyOTPWithDetails(req.Phone, req.OTPCode, purpose)
	if err != nil {
		logger.Error("Failed to verify OTP", err)

		// Check if the error is due to OTP expiration and automatically send a new OTP
		if otpRecord != nil && otpRecord.IsExpired() && !otpRecord.IsCurrentlyBlocked() {
			// OTP has expired, send a new one automatically
			newOTPRecord, sendErr := oc.OTPService.SendOTP(req.Phone, purpose)
			if sendErr != nil {
				logger.Error("Failed to send new OTP after expiration", sendErr)
				return c.Status(fiber.StatusInternalServerError).JSON(types.ApiResponse{
					Status:  fiber.StatusInternalServerError,
					Message: "OTP has expired and failed to send new OTP",
					Data: otpTypes.OTPResponse{
						Message: "OTP has expired and failed to send new OTP",
						Success: false,
					},
				})
			}

			// Return response indicating OTP expired and new one was sent
			return c.Status(fiber.StatusBadRequest).JSON(types.ApiResponse{
				Status:  fiber.StatusBadRequest,
				Message: "OTP has expired. A new OTP has been sent to your phone.",
				Data: otpTypes.OTPResponse{
					Message:    "OTP has expired. A new OTP has been sent to your phone.",
					ExpiresAt:  newOTPRecord.ExpiresAt.Format("2006-01-02 15:04:05"),
					Success:    false,
					NewOTPSent: true,
				},
			})
		}

		// If we have an OTP record, we can provide more detailed error information
		if otpRecord != nil {
			remainingAttempts := otpRecord.MaxRetries - otpRecord.RetryCount
			isBlocked := otpRecord.IsCurrentlyBlocked()

			return c.Status(fiber.StatusBadRequest).JSON(types.ApiResponse{
				Status:  fiber.StatusBadRequest,
				Message: err.Error(), // This will contain the detailed error message with attempts
				Data: otpTypes.OTPResponse{
					Message:           err.Error(),
					Success:           false,
					RemainingAttempts: &remainingAttempts,
					IsBlocked:         &isBlocked,
				},
			})
		}

		// Fallback for other errors
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

// SendOTPForBooking sends OTP for a specific booking without updating any phone numbers
// This is purely for verification purposes tied to a booking ID
func (oc *Controller) SendOTPForBooking(c *fiber.Ctx) error {
	var req bookingTypes.SendOTPForBookingRequest
	if err := c.BodyParser(&req); err != nil {
		logger.Error("Failed to parse request body", err)
		return c.Status(fiber.StatusBadRequest).JSON(types.ApiResponse{
			Status:  fiber.StatusBadRequest,
			Message: "Invalid request body",
			Data:    nil,
		})
	}

	// Find the booking to ensure it exists
	var booking bookingModel.Booking
	if err := oc.DB.First(&booking, req.BookingID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return c.Status(fiber.StatusNotFound).JSON(types.ApiResponse{
				Status:  fiber.StatusNotFound,
				Message: "Booking not found",
				Data:    nil,
			})
		}
		logger.Error("Failed to find booking", err)
		return c.Status(fiber.StatusInternalServerError).JSON(types.ApiResponse{
			Status:  fiber.StatusInternalServerError,
			Message: "Internal server error",
			Data:    nil,
		})
	}

	// Validate phone number matches booking's associated phone
	validPhone := false
	if req.Phone == booking.Phone {
		validPhone = true
	} else if booking.DeliveryPhone != nil && req.Phone == *booking.DeliveryPhone {
		validPhone = true
	} else if booking.EmergencyContactPhone != nil && req.Phone == *booking.EmergencyContactPhone {
		validPhone = true
	}

	if !validPhone {
		return c.Status(fiber.StatusBadRequest).JSON(types.ApiResponse{
			Status:  fiber.StatusBadRequest,
			Message: "Phone number is not associated with this booking",
			Data:    nil,
		})
	}

	// Determine OTP purpose based on which phone is being used
	var purpose otp.OTPPurpose
	if booking.DeliveryPhone != nil && req.Phone == *booking.DeliveryPhone {
		purpose = otp.OTPPurposeDeliveryApplyPhone
	} else {
		purpose = otp.OTPPurposeDeliveryConfirmPhone // For other phones
	}

	// Send OTP using OTP service with booking ID (always required for booking OTP)
	otpRecord, err := oc.OTPService.SendOTPWithBookingID(req.Phone, purpose, &req.BookingID)
	if err != nil {
		logger.Error("Failed to send OTP for booking", err)

		// Handle specific error cases
		errMsg := err.Error()
		if errMsg == "OTP requests are blocked permanently due to too many failed attempts" ||
			(len(errMsg) > 20 && errMsg[:20] == "OTP requests are blocked until") {
			return c.Status(fiber.StatusTooManyRequests).JSON(types.ApiResponse{
				Status:  fiber.StatusTooManyRequests,
				Message: err.Error(),
				Data:    nil,
			})
		}

		return c.Status(fiber.StatusInternalServerError).JSON(types.ApiResponse{
			Status:  fiber.StatusInternalServerError,
			Message: "Failed to send OTP",
			Data:    nil,
		})
	}

	logger.Success(fmt.Sprintf("OTP sent to phone %s for booking ID: %d (Purpose: %s)", req.Phone, req.BookingID, purpose))

	return c.Status(fiber.StatusOK).JSON(types.ApiResponse{
		Status:  fiber.StatusOK,
		Message: "OTP sent successfully for booking verification",
		Data: map[string]interface{}{
			"otp_id":     otpRecord.ID,
			"booking_id": req.BookingID,
			"phone":      req.Phone,
			"purpose":    purpose,
			"expires_at": otpRecord.ExpiresAt,
		},
	})
}

// VerifyOTPForBooking verifies OTP for a specific booking without any updates
func (oc *Controller) VerifyOTPForBooking(c *fiber.Ctx) error {
	var req bookingTypes.VerifyOTPForBookingRequest
	if err := c.BodyParser(&req); err != nil {
		logger.Error("Failed to parse request body", err)
		return c.Status(fiber.StatusBadRequest).JSON(types.ApiResponse{
			Status:  fiber.StatusBadRequest,
			Message: "Invalid request body",
			Data:    nil,
		})
	}

	// Find the booking
	var booking bookingModel.Booking
	if err := oc.DB.First(&booking, req.BookingID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return c.Status(fiber.StatusNotFound).JSON(types.ApiResponse{
				Status:  fiber.StatusNotFound,
				Message: "Booking not found",
				Data:    nil,
			})
		}
		logger.Error("Failed to find booking", err)
		return c.Status(fiber.StatusInternalServerError).JSON(types.ApiResponse{
			Status:  fiber.StatusInternalServerError,
			Message: "Internal server error",
			Data:    nil,
		})
	}

	// Validate phone number belongs to booking
	validPhone := false
	if req.Phone == booking.Phone {
		validPhone = true
	} else if booking.DeliveryPhone != nil && req.Phone == *booking.DeliveryPhone {
		validPhone = true
	} else if booking.EmergencyContactPhone != nil && req.Phone == *booking.EmergencyContactPhone {
		validPhone = true
	}

	if !validPhone {
		return c.Status(fiber.StatusBadRequest).JSON(types.ApiResponse{
			Status:  fiber.StatusBadRequest,
			Message: "Phone number is not associated with this booking",
			Data:    nil,
		})
	}

	// Determine the expected purpose
	var purpose otp.OTPPurpose
	if booking.DeliveryPhone != nil && req.Phone == *booking.DeliveryPhone {
		purpose = otp.OTPPurposeDeliveryApplyPhone
	} else {
		purpose = otp.OTPPurposeDeliveryConfirmPhone
	}

	// Verify OTP using OTP service
	isValid, otpRecord, err := oc.OTPService.VerifyOTPWithDetails(req.Phone, req.OTPCode, purpose)
	if err != nil {
		logger.Error("Failed to verify OTP for booking", err)

		// Check if the error is due to OTP expiration and automatically send a new OTP
		if otpRecord != nil && otpRecord.IsExpired() && !otpRecord.IsCurrentlyBlocked() {
			// OTP has expired, send a new one automatically
			newOTPRecord, sendErr := oc.OTPService.SendOTPWithBookingID(req.Phone, purpose, &req.BookingID)
			if sendErr != nil {
				logger.Error("Failed to send new OTP after expiration for booking", sendErr)
				return c.Status(fiber.StatusInternalServerError).JSON(types.ApiResponse{
					Status:  fiber.StatusInternalServerError,
					Message: "OTP has expired and failed to send new OTP",
					Data:    nil,
				})
			}

			// Return response indicating OTP expired and new one was sent
			return c.Status(fiber.StatusBadRequest).JSON(types.ApiResponse{
				Status:  fiber.StatusBadRequest,
				Message: "OTP has expired. A new OTP has been sent to your phone.",
				Data: map[string]interface{}{
					"message":      "OTP has expired. A new OTP has been sent to your phone.",
					"booking_id":   req.BookingID,
					"phone":        req.Phone,
					"purpose":      purpose,
					"expires_at":   newOTPRecord.ExpiresAt,
					"new_otp_sent": true,
				},
			})
		}

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
			Data:    nil,
		})
	}

	logger.Success(fmt.Sprintf("OTP verified for booking ID: %d, phone: %s", booking.ID, req.Phone))

	responseData := map[string]interface{}{
		"booking_id":      booking.ID,
		"phone":           req.Phone,
		"purpose":         purpose,
		"verification_at": otpRecord.UpdatedAt,
		"verified":        true,
	}

	return c.Status(fiber.StatusOK).JSON(types.ApiResponse{
		Status:  fiber.StatusOK,
		Message: "OTP verified successfully for booking",
		Data:    responseData,
	})
}

// GetBookingOTPStatus gets the current OTP status for a booking and phone
func (oc *Controller) GetBookingOTPStatus(c *fiber.Ctx) error {
	var req bookingTypes.GetBookingOTPStatusRequest
	if err := c.BodyParser(&req); err != nil {
		logger.Error("Failed to parse request body", err)
		return c.Status(fiber.StatusBadRequest).JSON(types.ApiResponse{
			Status:  fiber.StatusBadRequest,
			Message: "Invalid request body",
			Data:    nil,
		})
	}

	// Find the booking
	var booking bookingModel.Booking
	if err := oc.DB.First(&booking, req.BookingID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return c.Status(fiber.StatusNotFound).JSON(types.ApiResponse{
				Status:  fiber.StatusNotFound,
				Message: "Booking not found",
				Data:    nil,
			})
		}
		logger.Error("Failed to find booking", err)
		return c.Status(fiber.StatusInternalServerError).JSON(types.ApiResponse{
			Status:  fiber.StatusInternalServerError,
			Message: "Internal server error",
			Data:    nil,
		})
	}

	// Determine purpose based on phone
	var purpose otp.OTPPurpose
	if booking.DeliveryPhone != nil && req.Phone == *booking.DeliveryPhone {
		purpose = otp.OTPPurposeDeliveryApplyPhone
	} else {
		purpose = otp.OTPPurposeDeliveryConfirmPhone
	}

	// Get OTP status
	retryInfo, err := oc.OTPService.GetOTPRetryInfo(req.Phone, purpose)
	if err != nil {
		logger.Error("Failed to get OTP retry info for booking", err)
		return c.Status(fiber.StatusInternalServerError).JSON(types.ApiResponse{
			Status:  fiber.StatusInternalServerError,
			Message: "Internal server error",
			Data:    nil,
		})
	}

	responseData := map[string]interface{}{
		"booking_id": booking.ID,
		"phone":      req.Phone,
		"purpose":    purpose,
		"retry_info": retryInfo,
	}

	return c.Status(fiber.StatusOK).JSON(types.ApiResponse{
		Status:  fiber.StatusOK,
		Message: "OTP status retrieved successfully",
		Data:    responseData,
	})
}
