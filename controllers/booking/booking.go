package booking

import (
	"fmt"
	"passport-booking/database"
	"passport-booking/logger"
	addressModel "passport-booking/models/address"
	bookingModel "passport-booking/models/booking"
	"passport-booking/models/otp"
	"passport-booking/services/booking_event"
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

func (bc *BookingController) Index(c *fiber.Ctx) error {
	var bookings []bookingModel.Booking
	if err := bc.DB.Preload("User").Preload("AddressInfo").Find(&bookings).Error; err != nil {
		logger.Error("Failed to fetch bookings", err)
		return c.Status(fiber.StatusInternalServerError).JSON(types.ApiResponse{
			Status:  fiber.StatusInternalServerError,
			Message: "Failed to fetch bookings",
			Data:    nil,
		})
	}

	return c.Status(fiber.StatusOK).JSON(types.ApiResponse{
		Status:  fiber.StatusOK,
		Message: "Bookings fetched successfully",
		Data:    bookings,
	})
}

// Store creates a new booking with basic information (first step)
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

	// Validate request using the validation method from types
	if err := req.Validate(); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(types.ApiResponse{
			Status:  fiber.StatusBadRequest,
			Message: err.Error(),
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
	err = database.DB.Preload("User").Where("app_or_order_id = ?", req.AppOrOrderID).First(&existingBooking).Error

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

	var booking bookingModel.Booking

	// Use DB.Transaction for automatic rollback on error
	err = database.DB.Transaction(func(tx *gorm.DB) error {
		// Create booking record with basic information only
		booking = bookingModel.Booking{
			UserID:                userID,
			AppOrOrderID:          req.AppOrOrderID,
			Name:                  req.Name,
			FatherName:            req.FatherName,
			MotherName:            req.MotherName,
			Phone:                 req.Phone,
			Address:               req.Address,
			EmergencyContactName:  &req.EmergencyContactName,
			EmergencyContactPhone: &req.EmergencyContactPhone,
			Status:                bookingModel.BookingStatusInitial,
			BookingDate:           time.Now(),
			CreatedBy:             strconv.FormatUint(uint64(userID), 10),
			CreatedAt:             time.Now(),
		}

		if err := tx.Create(&booking).Error; err != nil {
			logger.Error("Failed to create booking", err)
			return err
		}

		if err := booking_event.SnapshotBookingToEvent(tx, &booking, "created", strconv.FormatUint(uint64(userID), 10)); err != nil {
			logger.Error("Failed to write booking event (created)", err)
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
	err = database.DB.Preload("User").First(&createdBooking, booking.ID).Error
	if err != nil {
		logger.Error("Failed to load created booking data", err)
		return c.Status(fiber.StatusInternalServerError).JSON(types.ApiResponse{
			Status:  fiber.StatusInternalServerError,
			Message: "Booking created but failed to retrieve complete data",
			Data:    nil,
		})
	}

	// Return success response with basic booking data
	return c.Status(fiber.StatusCreated).JSON(types.ApiResponse{
		Status:  fiber.StatusCreated,
		Message: "Booking created successfully. Please complete the delivery information.",
		Data:    createdBooking,
	})
}

// StoreUpdate updates an existing booking with delivery and address information (second step)
func (bc *BookingController) StoreUpdate(c *fiber.Ctx) error {
	var req bookingTypes.BookingStoreUpdateRequest
	if err := c.BodyParser(&req); err != nil {
		logger.Error("Failed to parse request body", err)
		return c.Status(fiber.StatusBadRequest).JSON(types.ApiResponse{
			Status:  fiber.StatusBadRequest,
			Message: "Invalid request body",
			Data:    nil,
		})
	}

	// Validate request using the validation method from types
	if err := req.Validate(); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(types.ApiResponse{
			Status:  fiber.StatusBadRequest,
			Message: err.Error(),
			Data:    nil,
		})
	}

	// Get booking ID from URL parameter
	bookingIDParam := c.Params("id")
	bookingID, err := strconv.Atoi(bookingIDParam)
	if err != nil || bookingID <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(types.ApiResponse{
			Status:  fiber.StatusBadRequest,
			Message: "Invalid booking ID",
			Data:    nil,
		})
	}

	// Get user information from token
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

	// Find the existing booking
	var booking bookingModel.Booking
	if err := bc.DB.First(&booking, bookingID).Error; err != nil {
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
			Message: "Database error",
			Data:    nil,
		})
	}

	// Check if the booking belongs to the current user
	if booking.UserID != userID {
		return c.Status(fiber.StatusForbidden).JSON(types.ApiResponse{
			Status:  fiber.StatusForbidden,
			Message: "You don't have permission to update this booking",
			Data:    nil,
		})
	}

	var address addressModel.Address

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

		// Update booking with delivery and address information
		booking.DeliveryBranchCode = &req.DeliveryBranchCode
		booking.ReceiverName = &req.ReceiverName
		booking.AddressID = &address.ID
		booking.Status = bookingModel.BookingStatusPreBooked
		booking.UpdatedBy = strconv.FormatUint(uint64(userID), 10)
		booking.UpdatedAt = time.Now()

		if err := tx.Save(&booking).Error; err != nil {
			logger.Error("Failed to update booking", err)
			return err
		}

		if err := booking_event.SnapshotBookingToEvent(tx, &booking, "delivery_info_updated", strconv.FormatUint(uint64(userID), 10)); err != nil {
			logger.Error("Failed to write booking event (delivery_info_updated)", err)
			return err
		}

		return nil
	})

	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(types.ApiResponse{
			Status:  fiber.StatusInternalServerError,
			Message: "Failed to update booking",
			Data:    nil,
		})
	}

	// Log success
	logger.Success(fmt.Sprintf("Booking delivery information updated successfully for ID: %d", booking.ID))

	// Load the complete booking data with relationships
	var updatedBooking bookingModel.Booking
	err = database.DB.Preload("User").Preload("AddressInfo").First(&updatedBooking, booking.ID).Error
	if err != nil {
		logger.Error("Failed to load updated booking data", err)
		return c.Status(fiber.StatusInternalServerError).JSON(types.ApiResponse{
			Status:  fiber.StatusInternalServerError,
			Message: "Booking updated but failed to retrieve complete data",
			Data:    nil,
		})
	}

	return c.Status(fiber.StatusOK).JSON(types.ApiResponse{
		Status:  fiber.StatusOK,
		Message: "Booking delivery information updated successfully",
		Data:    updatedBooking,
	})
}

// show indivisual booking info
func (bc *BookingController) Show(c *fiber.Ctx) error {
	bookingIDParam := c.Params("id")
	bookingID, err := strconv.Atoi(bookingIDParam)
	if err != nil || bookingID <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(types.ApiResponse{
			Status:  fiber.StatusBadRequest,
			Message: "Invalid booking ID",
			Data:    nil,
		})
	}

	var booking bookingModel.Booking
	if err := bc.DB.Preload("User").Preload("AddressInfo").First(&booking, bookingID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return c.Status(fiber.StatusNotFound).JSON(types.ApiResponse{
				Status:  fiber.StatusNotFound,
				Message: "Booking not found",
				Data:    nil,
			})
		}
		logger.Error("Failed to fetch booking", err)
		return c.Status(fiber.StatusInternalServerError).JSON(types.ApiResponse{
			Status:  fiber.StatusInternalServerError,
			Message: "Failed to fetch booking",
			Data:    nil,
		})
	}

	return c.Status(fiber.StatusOK).JSON(types.ApiResponse{
		Status:  fiber.StatusOK,
		Message: "Booking fetched successfully",
		Data:    booking,
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

	// Validate request using the validation method from types
	if err := req.Validate(); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(types.ApiResponse{
			Status:  fiber.StatusBadRequest,
			Message: err.Error(),
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
	booking.DeliveryPhoneAppliedVerified = false // Reset verification status

	if err := bc.DB.Save(&booking).Error; err != nil {
		logger.Error("Failed to update delivery phone", err)
		return c.Status(fiber.StatusInternalServerError).JSON(types.ApiResponse{
			Status:  fiber.StatusInternalServerError,
			Message: "Failed to update delivery phone",
			Data:    nil,
		})
	}

	if err := booking_event.SnapshotBookingToEvent(bc.DB, &booking, "delivery_phone_updated", strconv.FormatUint(uint64(booking.UserID), 10)); err != nil {
		logger.Error("Failed to write booking event (delivery_phone_updated)", err)
	}

	// Send OTP to the new delivery phone
	otpSvc := otpService.NewOTPService(bc.DB)
	otpRecord, err := otpSvc.SendOTPWithBookingID(req.DeliveryPhone, otp.OTPPurposeDeliveryApplyPhone, &req.BookingID)
	if err != nil {
		logger.Error("Failed to send OTP to delivery phone", err)

		// Check if it's a blocking error that should be returned as error response
		errMsg := err.Error()
		if errMsg == "OTP requests are blocked permanently due to too many failed attempts" ||
			(len(errMsg) > 20 && errMsg[:20] == "OTP requests are blocked until") {
			return c.Status(fiber.StatusTooManyRequests).JSON(types.ApiResponse{
				Status:  fiber.StatusTooManyRequests,
				Message: err.Error(),
				Data:    nil,
			})
		}

		// For other OTP errors, return error response instead of continuing
		return c.Status(fiber.StatusInternalServerError).JSON(types.ApiResponse{
			Status:  fiber.StatusInternalServerError,
			Message: "Failed to send OTP to delivery phone",
			Data: map[string]interface{}{
				"booking":   booking,
				"otp_error": err.Error(),
			},
		})
	} else {
		logger.Success(fmt.Sprintf("OTP sent to delivery phone %s for booking ID: %d", req.DeliveryPhone, req.BookingID))
	}

	responseData := map[string]interface{}{
		"booking": booking,
	}

	if otpRecord != nil {
		responseData["otp_info"] = map[string]interface{}{
			"otp_id":     otpRecord.ID,
			"expires_at": otpRecord.ExpiresAt,
			"phone":      req.DeliveryPhone,
		}
	}

	return c.Status(fiber.StatusOK).JSON(types.ApiResponse{
		Status:  fiber.StatusOK,
		Message: "Delivery phone updated and OTP sent successfully",
		Data:    responseData,
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

	// Validate request using the validation method from types
	if err := req.Validate(); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(types.ApiResponse{
			Status:  fiber.StatusBadRequest,
			Message: err.Error(),
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

	if booking.DeliveryPhoneAppliedVerified {
		return c.Status(fiber.StatusBadRequest).JSON(types.ApiResponse{
			Status:  fiber.StatusBadRequest,
			Message: "Delivery phone is already verified",
			Data:    nil,
		})
	}

	// Verify OTP using OTP service
	otpSvc := otpService.NewOTPService(bc.DB)
	isValid, otpRecord, err := otpSvc.VerifyOTPWithDetails(req.Phone, req.OTPCode, otp.OTPPurposeDeliveryApplyPhone)
	if err != nil {
		logger.Error("Failed to verify OTP", err)

		// If we have an OTP record, we can provide more detailed error information
		if otpRecord != nil {
			remainingAttempts := otpRecord.MaxRetries - otpRecord.RetryCount
			isBlocked := otpRecord.IsCurrentlyBlocked()
			isExpired := otpRecord.IsExpired()

			// Handle OTP expiration separately
			if isExpired {
				return c.Status(fiber.StatusBadRequest).JSON(types.ApiResponse{
					Status:  fiber.StatusBadRequest,
					Message: "OTP has expired. Please request a new OTP",
					Data: map[string]interface{}{
						"error":              "OTP_EXPIRED",
						"expired_at":         otpRecord.ExpiresAt,
						"is_expired":         true,
						"is_blocked":         isBlocked,
						"remaining_attempts": remainingAttempts,
						"success":            false,
					},
				})
			}

			// Handle blocked OTP separately
			if isBlocked {
				return c.Status(fiber.StatusTooManyRequests).JSON(types.ApiResponse{
					Status:  fiber.StatusTooManyRequests,
					Message: err.Error(), // This will contain the detailed blocked message
					Data: map[string]interface{}{
						"error":              "OTP_BLOCKED",
						"is_blocked":         true,
						"blocked_until":      otpRecord.BlockedUntil,
						"remaining_attempts": remainingAttempts,
						"success":            false,
					},
				})
			}

			// Handle other OTP verification errors (like wrong OTP)
			return c.Status(fiber.StatusBadRequest).JSON(types.ApiResponse{
				Status:  fiber.StatusBadRequest,
				Message: err.Error(), // This will contain the detailed error message with attempts
				Data: map[string]interface{}{
					"error":              "OTP_INVALID",
					"remaining_attempts": remainingAttempts,
					"is_blocked":         isBlocked,
					"is_expired":         isExpired,
					"success":            false,
				},
			})
		}

		// Fallback for other errors
		return c.Status(fiber.StatusInternalServerError).JSON(types.ApiResponse{
			Status:  fiber.StatusInternalServerError,
			Message: err.Error(), // Show the actual error message instead of generic
			Data:    nil,
		})
	}

	if !isValid {
		// This case should rarely happen now since we handle specific errors above
		return c.Status(fiber.StatusBadRequest).JSON(types.ApiResponse{
			Status:  fiber.StatusBadRequest,
			Message: "Invalid OTP",
			Data:    nil,
		})
	}

	// Encrypt OTP data for storage
	var deliveryPhoneAppliedOTPEncrypted string

	if otpRecord != nil {
		// Encrypt the delivered OTP (the OTP code that was verified)
		encryptedDeliveryPhoneAppliedOTP, err := utils.EncryptData(otpRecord.OTPCode)
		if err != nil {
			logger.Error("Failed to encrypt delivered OTP", err)
		} else {
			deliveryPhoneAppliedOTPEncrypted = encryptedDeliveryPhoneAppliedOTP
		}
	}

	// Mark delivery phone as verified and store encrypted OTPs
	booking.DeliveryPhoneAppliedVerified = true
	booking.DeliveryPhoneAppliedOTPEncrypted = &deliveryPhoneAppliedOTPEncrypted

	// Save the updated booking
	if err := bc.DB.Save(&booking).Error; err != nil {
		logger.Error("Failed to update delivery phone verification status", err)
		return c.Status(fiber.StatusInternalServerError).JSON(types.ApiResponse{
			Status:  fiber.StatusInternalServerError,
			Message: "Failed to update verification status",
			Data:    nil,
		})
	}

	if err := booking_event.SnapshotBookingToEvent(bc.DB, &booking, "phone_applied_verified", strconv.FormatUint(uint64(booking.UserID), 10)); err != nil {
		logger.Error("Failed to write booking event (phone_applied_verified)", err)
	}

	logger.Success(fmt.Sprintf("Delivery phone verified for booking ID: %d", booking.ID))

	responseData := map[string]interface{}{
		"booking":  booking,
		"verified": true,
	}

	return c.Status(fiber.StatusOK).JSON(types.ApiResponse{
		Status:  fiber.StatusOK,
		Message: "Delivery phone verified successfully",
		Data:    responseData,
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

	// Validate request using the validation method from types
	if err := req.Validate(); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(types.ApiResponse{
			Status:  fiber.StatusBadRequest,
			Message: err.Error(),
			Data:    nil,
		})
	}

	// Basic check for phone number presence
	if req.Phone == "" {
		return c.Status(fiber.StatusBadRequest).JSON(types.ApiResponse{
			Status:  fiber.StatusBadRequest,
			Message: "phone is required",
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
	// Validate request using the validation method from types
	if err := req.Validate(); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(types.ApiResponse{
			Status:  fiber.StatusBadRequest,
			Message: err.Error(),
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
	if booking.DeliveryPhoneAppliedVerified {
		return c.Status(fiber.StatusBadRequest).JSON(types.ApiResponse{
			Status:  fiber.StatusBadRequest,
			Message: "Delivery phone is already verified",
			Data:    nil,
		})
	}

	// Send OTP using OTP service (with retry handling)
	otpSvc := otpService.NewOTPService(bc.DB)
	otpRecord, err := otpSvc.SendOTPWithBookingID(req.Phone, otp.OTPPurposeDeliveryApplyPhone, &req.BookingID)
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
