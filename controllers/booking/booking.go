package booking

import (
	"fmt"
	"passport-booking/database"
	"passport-booking/logger"
	addressModel "passport-booking/models/address"
	bookingModel "passport-booking/models/booking"
	"passport-booking/models/otp"
	otpService "passport-booking/services/otp"
	"passport-booking/types"
	bookingTypes "passport-booking/types/booking"
	"passport-booking/utils"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// BookingController handles booking-related HTTP requests
type BookingController struct {
	DB     *gorm.DB
	Logger *logger.AsyncLogger
}

// NewBookingController creates a new booking controller
func NewBookingController(db *gorm.DB, asyncLogger *logger.AsyncLogger) *BookingController {
	return &BookingController{
		DB:     db,
		Logger: asyncLogger,
	}
}

// Store creates a new booking with address
func (bc *BookingController) Store(c *fiber.Ctx) error {
	// Parse request body
	var req bookingTypes.BookingCreateRequest
	if err := c.BodyParser(&req); err != nil {
		logger.Error("Failed to parse request body", err)
		return c.Status(fiber.StatusBadRequest).JSON(types.ApiResponse{
			Status:  fiber.StatusBadRequest,
			Message: "Invalid request body",
			Data:    nil,
		})
	}

	claims, ok := c.Locals("user").(map[string]interface{})
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(types.ApiResponse{
			Message: "Invalid user claims",
			Status:  fiber.StatusUnauthorized,
			Data:    nil,
		})
	}

	userUUID, ok := claims["uuid"].(string)
	if !ok || userUUID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(types.ApiResponse{
			Message: "User UUID not found in token",
			Status:  fiber.StatusUnauthorized,
			Data:    nil,
		})
	}

	userInfo, err := utils.GetUserByUUID(userUUID)

	if err != nil {
		logger.Error("Error finding user by UUID", err)
		status := fiber.StatusInternalServerError
		msg := "Database error"
		if err.Error() == "user not found" {
			status = fiber.StatusUnauthorized
			msg = "User not found"
		}
		return c.Status(status).JSON(types.ApiResponse{
			Message: msg,
			Status:  status,
			Data:    nil,
		})
	}

	userID := uint(userInfo.ID)

	// Check if booking with the same AppOrOrderID already exists
	var existingBooking bookingModel.Booking
	err = database.DB.Preload("User").Preload("AddressInfo").Where("app_or_order_id = ?", req.AppOrOrderID).First(&existingBooking).Error

	if err == nil {
		// Booking already exists, return existing data
		logger.Info(fmt.Sprintf("Booking with AppOrOrderID %s already exists", req.AppOrOrderID))
		return c.Status(fiber.StatusOK).JSON(types.ApiResponse{
			Status:  fiber.StatusOK,
			Message: "Booking already exists",
			Data:    existingBooking,
		})
	} else if err != gorm.ErrRecordNotFound {
		// Some other database error occurred
		logger.Error("Database error while checking existing booking", err)
		return c.Status(fiber.StatusInternalServerError).JSON(types.ApiResponse{
			Status:  fiber.StatusInternalServerError,
			Message: "Database error",
			Data:    nil,
		})
	}

	var address addressModel.Address
	var booking bookingModel.Booking

	// Use DB.Transaction for automatic rollback on error
	err = database.DB.Transaction(func(tx *gorm.DB) error {
		// Create address record
		address = addressModel.Address{
			Division:      &req.Division,
			District:      &req.District,
			PoliceStation: &req.PoliceStation,
			PostOffice:    &req.PostOffice,
			StreetAddress: &req.StreetAddress,
			AddressType:   req.AddressType,
		}

		if err := tx.Create(&address).Error; err != nil {
			logger.Error("Failed to create address", err)
			return err
		}

		// Create booking record
		booking = bookingModel.Booking{
			UserID:                userID,
			AppOrOrderID:          req.AppOrOrderID,
			CurrentBagID:          &req.CurrentBagID,
			Barcode:               &req.Barcode,
			Name:                  req.Name,
			FatherName:            req.FatherName,
			MotherName:            req.MotherName,
			Phone:                 req.Phone,
			Address:               req.Address,
			EmergencyContactName:  &req.EmergencyContactName,
			EmergencyContactPhone: &req.EmergencyContactPhone,
			AddressID:             address.ID, // Link to the created address
			Status:                bookingModel.BookingStatusInitial,
			BookingDate:           time.Now(),
			CreatedBy:             strconv.FormatUint(uint64(userID), 10),
			CreatedAt:             time.Now(),
		}

		// Handle delivery phone if provided
		if req.DeliveryPhone != "" {
			booking.DeliveryPhone = &req.DeliveryPhone
			booking.DeliveryPhoneVerified = false
		}

		if err := tx.Create(&booking).Error; err != nil {
			logger.Error("Failed to create booking", err)
			return err
		}

		return nil
	})

	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(types.ApiResponse{
			Status:  fiber.StatusInternalServerError,
			Message: "Failed to save booking",
			Data:    nil,
		})
	}

	// Log success
	logger.Success(fmt.Sprintf("Booking created successfully with ID: %d", booking.ID))

	// Load the complete booking data with relationships
	var createdBooking bookingModel.Booking
	err = database.DB.Preload("User").Preload("AddressInfo").First(&createdBooking, booking.ID).Error
	if err != nil {
		logger.Error("Failed to load created booking data", err)
		return c.Status(fiber.StatusInternalServerError).JSON(types.ApiResponse{
			Status:  fiber.StatusInternalServerError,
			Message: "Booking created but failed to retrieve complete data",
			Data:    nil,
		})
	}

	// Return success response with complete booking data
	return c.Status(fiber.StatusCreated).JSON(types.ApiResponse{
		Status:  fiber.StatusCreated,
		Message: "Booking created successfully",
		Data:    createdBooking,
	})
}

// UpdateDeliveryPhone updates the delivery phone for a booking
func (bc *BookingController) UpdateDeliveryPhone(c *fiber.Ctx) error {
	var req bookingTypes.UpdateDeliveryPhoneRequest
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
	if err := bc.DB.First(&booking, req.BookingID).Error; err != nil {
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

	// Update delivery phone
	booking.DeliveryPhone = &req.DeliveryPhone
	booking.DeliveryPhoneVerified = false // Reset verification status

	if err := bc.DB.Save(&booking).Error; err != nil {
		logger.Error("Failed to update delivery phone", err)
		return c.Status(fiber.StatusInternalServerError).JSON(types.ApiResponse{
			Status:  fiber.StatusInternalServerError,
			Message: "Failed to update delivery phone",
			Data:    nil,
		})
	}

	return c.Status(fiber.StatusOK).JSON(types.ApiResponse{
		Status:  fiber.StatusOK,
		Message: "Delivery phone updated successfully",
		Data:    booking,
	})
}

// VerifyDeliveryPhone verifies the delivery phone OTP and marks it as verified
func (bc *BookingController) VerifyDeliveryPhone(c *fiber.Ctx) error {
	var req bookingTypes.VerifyDeliveryPhoneRequest
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
	if err := bc.DB.First(&booking, req.BookingID).Error; err != nil {
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

	// Check if the phone matches the booking's delivery phone
	if booking.DeliveryPhone == nil || *booking.DeliveryPhone != req.Phone {
		return c.Status(fiber.StatusBadRequest).JSON(types.ApiResponse{
			Status:  fiber.StatusBadRequest,
			Message: "Phone number does not match booking delivery phone",
			Data:    nil,
		})
	}

	// Verify OTP using OTP service
	otpSvc := otpService.NewOTPService(bc.DB)
	isValid, otpRecord, err := otpSvc.VerifyOTPWithDetails(req.Phone, req.OTPCode, otp.OTPPurposeDeliveryApplyPhone)
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
			Data:    nil,
		})
	}

	// Encrypt OTP data for storage
	var deliveredOTPEncrypted, initiatedOTPEncrypted *string

	if otpRecord != nil {
		// Encrypt the delivered OTP (the OTP code that was verified)
		encryptedDeliveredOTP, err := utils.EncryptData(otpRecord.OTPCode)
		if err != nil {
			logger.Error("Failed to encrypt delivered OTP", err)
		} else {
			deliveredOTPEncrypted = &encryptedDeliveredOTP
		}

		// Encrypt the initiated OTP timestamp for tracking purposes
		encryptedInitiatedOTP, err := utils.EncryptData(otpRecord.CreatedAt.Format(time.RFC3339))
		if err != nil {
			logger.Error("Failed to encrypt initiated OTP timestamp", err)
		} else {
			initiatedOTPEncrypted = &encryptedInitiatedOTP
		}
	}

	// Mark delivery phone as verified and store encrypted OTPs
	booking.DeliveryPhoneVerified = true
	booking.DeliveredOTPEncrypted = deliveredOTPEncrypted
	booking.InitiatedOTPEncrypted = initiatedOTPEncrypted
	if err := bc.DB.Save(&booking).Error; err != nil {
		logger.Error("Failed to update delivery phone verification status", err)
		return c.Status(fiber.StatusInternalServerError).JSON(types.ApiResponse{
			Status:  fiber.StatusInternalServerError,
			Message: "Failed to update verification status",
			Data:    nil,
		})
	}

	logger.Success(fmt.Sprintf("Delivery phone verified for booking ID: %d", booking.ID))

	return c.Status(fiber.StatusOK).JSON(types.ApiResponse{
		Status:  fiber.StatusOK,
		Message: "Delivery phone verified successfully",
		Data:    booking,
	})
}

// GetOTPRetryInfo returns retry information for delivery phone OTP
func (bc *BookingController) GetOTPRetryInfo(c *fiber.Ctx) error {
	var req bookingTypes.GetOTPRetryInfoRequest
	if err := c.BodyParser(&req); err != nil {
		logger.Error("Failed to parse request body", err)
		return c.Status(fiber.StatusBadRequest).JSON(types.ApiResponse{
			Status:  fiber.StatusBadRequest,
			Message: "Invalid request body",
			Data:    nil,
		})
	}

	// Get retry information from OTP service
	otpSvc := otpService.NewOTPService(bc.DB)
	retryInfo, err := otpSvc.GetOTPRetryInfo(req.Phone, otp.OTPPurposeDeliveryConfirmPhone)
	if err != nil {
		logger.Error("Failed to get OTP retry info", err)
		return c.Status(fiber.StatusInternalServerError).JSON(types.ApiResponse{
			Status:  fiber.StatusInternalServerError,
			Message: "Internal server error",
			Data:    nil,
		})
	}

	return c.Status(fiber.StatusOK).JSON(types.ApiResponse{
		Status:  fiber.StatusOK,
		Message: "OTP retry information retrieved successfully",
		Data:    retryInfo,
	})
}

// ResendOTP resends OTP for delivery phone verification
func (bc *BookingController) ResendOTP(c *fiber.Ctx) error {
	var req bookingTypes.ResendOTPRequest
	if err := c.BodyParser(&req); err != nil {
		logger.Error("Failed to parse request body", err)
		return c.Status(fiber.StatusBadRequest).JSON(types.ApiResponse{
			Status:  fiber.StatusBadRequest,
			Message: "Invalid request body",
			Data:    nil,
		})
	}

	// Find the booking to verify the phone number
	var booking bookingModel.Booking
	if err := bc.DB.First(&booking, req.BookingID).Error; err != nil {
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

	// Check if the phone matches the booking's delivery phone
	if booking.DeliveryPhone == nil || *booking.DeliveryPhone != req.Phone {
		return c.Status(fiber.StatusBadRequest).JSON(types.ApiResponse{
			Status:  fiber.StatusBadRequest,
			Message: "Phone number does not match booking delivery phone",
			Data:    nil,
		})
	}

	// Check if already verified
	if booking.DeliveryPhoneVerified {
		return c.Status(fiber.StatusBadRequest).JSON(types.ApiResponse{
			Status:  fiber.StatusBadRequest,
			Message: "Delivery phone is already verified",
			Data:    nil,
		})
	}

	// Send OTP using OTP service (with retry handling)
	otpSvc := otpService.NewOTPService(bc.DB)
	otpRecord, err := otpSvc.SendOTP(req.Phone, otp.OTPPurposeDeliveryApplyPhone)
	if err != nil {
		logger.Error("Failed to send OTP", err)

		// Check if it's a blocking error
		errMsg := err.Error()
		if errMsg == "OTP requests are blocked permanently due to too many failed attempts" ||
			len(errMsg) > 20 && errMsg[:20] == "OTP requests are blocked until" {
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

	logger.Success(fmt.Sprintf("OTP sent to phone %s for booking ID: %d", req.Phone, req.BookingID))

	return c.Status(fiber.StatusOK).JSON(types.ApiResponse{
		Status:  fiber.StatusOK,
		Message: "OTP sent successfully",
		Data: map[string]interface{}{
			"otp_id":     otpRecord.ID,
			"expires_at": otpRecord.ExpiresAt,
			"phone":      req.Phone,
		},
	})
}
