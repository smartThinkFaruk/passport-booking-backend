package delivery

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
	"passport-booking/logger"
	bookingModel "passport-booking/models/booking"
	"passport-booking/services/booking_event"
	otpService "passport-booking/services/otp"
	"passport-booking/types"
	deliveryTypes "passport-booking/types/delivery"
	"passport-booking/utils"
	"strconv"
)

// DeliveryController handles delivery-related HTTP requests
type DeliveryController struct {
	DB             *gorm.DB
	Logger         *logger.AsyncLogger
	loggerInstance *logger.AsyncLogger
}

// NewDeliveryController creates a new delivery controller
func NewDeliveryController(db *gorm.DB, asyncLogger *logger.AsyncLogger) *DeliveryController {
	return &DeliveryController{
		DB:             db,
		Logger:         asyncLogger,
		loggerInstance: asyncLogger,
	}
}

// Helper function to log API requests and responses
func (dc *DeliveryController) logAPIRequest(c *fiber.Ctx) {
	logEntry := utils.CreateSanitizedLogEntry(c)
	dc.loggerInstance.Log(logEntry)
}

// Helper function to send response and log in one call
func (dc *DeliveryController) sendResponseWithLog(c *fiber.Ctx, status int, response types.ApiResponse) error {
	result := c.Status(status).JSON(response)
	dc.logAPIRequest(c)
	return result
}

// DeliveryConfirmationSendOtp sends an OTP for delivery confirmation
func (dc *DeliveryController) DeliveryConfirmationSendOtp(c *fiber.Ctx) error {
	var req deliveryTypes.DeliveryPhoneSendOtpRequest
	if err := c.BodyParser(&req); err != nil {
		logger.Error("Failed to parse request body", err)
		return dc.sendResponseWithLog(c, fiber.StatusBadRequest, types.ApiResponse{
			Status:  fiber.StatusBadRequest,
			Message: "Invalid request body",
			Data:    nil,
		})
	}

	// Validate request using the validation method from types
	if err := req.Validate(); err != nil {
		return dc.sendResponseWithLog(c, fiber.StatusBadRequest, types.ApiResponse{
			Status:  fiber.StatusBadRequest,
			Message: err.Error(),
			Data:    nil,
		})
	}

	// Get user authentication information (postman user)
	claims, ok := c.Locals("user").(map[string]interface{})
	if !ok {
		return dc.sendResponseWithLog(c, fiber.StatusUnauthorized, types.ApiResponse{
			Message: "Invalid user claims",
			Status:  fiber.StatusUnauthorized,
			Data:    nil,
		})
	}

	userUUID, ok := claims["uuid"].(string)
	if !ok || userUUID == "" {
		return dc.sendResponseWithLog(c, fiber.StatusUnauthorized, types.ApiResponse{
			Message: "User UUID not found in token",
			Status:  fiber.StatusUnauthorized,
			Data:    nil,
		})
	}

	// Get postman user info
	postmanInfo, err := utils.GetUserByUUID(userUUID)
	if err != nil {
		logger.Error("Error finding postman by UUID", err)
		status := fiber.StatusInternalServerError
		msg := "Database error"
		if err.Error() == "user not found" {
			status = fiber.StatusUnauthorized
			msg = "Postman not found"
		}
		return dc.sendResponseWithLog(c, status, types.ApiResponse{
			Message: msg,
			Status:  status,
			Data:    nil,
		})
	}

	// Find the booking by barcode
	var booking bookingModel.Booking
	if err := dc.DB.Preload("User").Where("barcode = ?", req.BookingID).First(&booking).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return dc.sendResponseWithLog(c, fiber.StatusNotFound, types.ApiResponse{
				Status:  fiber.StatusNotFound,
				Message: "Booking not found",
				Data:    nil,
			})
		}
		logger.Error("Failed to find booking", err)
		return dc.sendResponseWithLog(c, fiber.StatusInternalServerError, types.ApiResponse{
			Status:  fiber.StatusInternalServerError,
			Message: "Internal server error",
			Data:    nil,
		})
	}

	// Validate booking is ready for delivery confirmation
	// Check if booking status allows delivery confirmation (received by postman)
	if booking.Status != bookingModel.BookingStatusReceivedByPostman {
		return dc.sendResponseWithLog(c, fiber.StatusBadRequest, types.ApiResponse{
			Status:  fiber.StatusBadRequest,
			Message: "Booking must be received by postman before delivery confirmation",
			Data:    nil,
		})
	}

	if booking.DeliveryPhone == nil {
		return dc.sendResponseWithLog(c, fiber.StatusBadRequest, types.ApiResponse{
			Status:  fiber.StatusBadRequest,
			Message: "No delivery phone found for this booking",
			Data:    nil,
		})
	}

	// Reset verification status for delivery confirmation
	booking.DeliveryPhoneConfirmedVerified = false

	if err := dc.DB.Save(&booking).Error; err != nil {
		logger.Error("Failed to update booking delivery confirmation status", err)
		return dc.sendResponseWithLog(c, fiber.StatusInternalServerError, types.ApiResponse{
			Status:  fiber.StatusInternalServerError,
			Message: "Failed to update booking",
			Data:    nil,
		})
	}

	if err := booking_event.SnapshotBookingToEvent(dc.DB, &booking, "delivery_confirmation_send_otp", strconv.FormatUint(uint64(postmanInfo.ID), 10)); err != nil {
		logger.Error("Failed to write booking event (delivery_confirmation_send_otp)", err)
	}

	// Send OTP to the delivery phone for confirmation
	otpSvc := otpService.NewOTPService(dc.DB)
	otpRecord, err := otpSvc.SendOTPWithBookingID(*booking.DeliveryPhone, req.Purpose, &booking.ID)
	if err != nil {
		logger.Error("Failed to send delivery confirmation OTP", err)

		// Check if it's a blocking error that should be returned as error response
		errMsg := err.Error()
		if errMsg == "OTP requests are blocked permanently due to too many failed attempts" ||
			(len(errMsg) > 20 && errMsg[:20] == "OTP requests are blocked until") {
			return dc.sendResponseWithLog(c, fiber.StatusTooManyRequests, types.ApiResponse{
				Status:  fiber.StatusTooManyRequests,
				Message: err.Error(),
				Data:    nil,
			})
		}

		// For other OTP errors, return error response instead of continuing
		return dc.sendResponseWithLog(c, fiber.StatusInternalServerError, types.ApiResponse{
			Status:  fiber.StatusInternalServerError,
			Message: "Failed to send delivery confirmation OTP",
			Data: map[string]interface{}{
				"booking":   booking,
				"otp_error": err.Error(),
			},
		})
	} else {
		logger.Success(fmt.Sprintf("Delivery confirmation OTP sent to phone %s for booking ID: %d (Barcode: %s) by postman: %s", *booking.DeliveryPhone, booking.ID, req.BookingID, postmanInfo.LegalName))
	}

	responseData := map[string]interface{}{
		"booking":      booking,
		"postman_id":   postmanInfo.ID,
		"postman_name": postmanInfo.LegalName,
	}

	if otpRecord != nil {
		responseData["otp_info"] = map[string]interface{}{
			"otp_id":     otpRecord.ID,
			"expires_at": otpRecord.ExpiresAt,
			"phone":      booking.DeliveryPhone,
			"purpose":    req.Purpose,
		}
	}

	return dc.sendResponseWithLog(c, fiber.StatusOK, types.ApiResponse{
		Status:  fiber.StatusOK,
		Message: "Delivery confirmation OTP sent successfully",
		Data:    responseData,
	})
}

// DeliveryConfirmationVerifyOtp verifies the OTP for delivery confirmation
func (dc *DeliveryController) DeliveryConfirmationVerifyOtp(c *fiber.Ctx) error {
	var req deliveryTypes.VerifyDeliveryPhoneRequest
	if err := c.BodyParser(&req); err != nil {
		logger.Error("Failed to parse request body", err)
		return dc.sendResponseWithLog(c, fiber.StatusBadRequest, types.ApiResponse{
			Status:  fiber.StatusBadRequest,
			Message: "Invalid request body",
			Data:    nil,
		})
	}

	// Validate request using the validation method from types
	if err := req.Validate(); err != nil {
		return dc.sendResponseWithLog(c, fiber.StatusBadRequest, types.ApiResponse{
			Status:  fiber.StatusBadRequest,
			Message: err.Error(),
			Data:    nil,
		})
	}

	// Get user authentication information (postman user)
	claims, ok := c.Locals("user").(map[string]interface{})
	if !ok {
		return dc.sendResponseWithLog(c, fiber.StatusUnauthorized, types.ApiResponse{
			Message: "Invalid user claims",
			Status:  fiber.StatusUnauthorized,
			Data:    nil,
		})
	}

	userUUID, ok := claims["uuid"].(string)
	if !ok || userUUID == "" {
		return dc.sendResponseWithLog(c, fiber.StatusUnauthorized, types.ApiResponse{
			Message: "User UUID not found in token",
			Status:  fiber.StatusUnauthorized,
			Data:    nil,
		})
	}

	// Get postman user info
	postmanInfo, err := utils.GetUserByUUID(userUUID)
	if err != nil {
		logger.Error("Error finding postman by UUID", err)
		status := fiber.StatusInternalServerError
		msg := "Database error"
		if err.Error() == "user not found" {
			status = fiber.StatusUnauthorized
			msg = "Postman not found"
		}
		return dc.sendResponseWithLog(c, status, types.ApiResponse{
			Message: msg,
			Status:  status,
			Data:    nil,
		})
	}

	// Find the booking by barcode
	var booking bookingModel.Booking
	if err := dc.DB.Preload("User").Where("barcode = ?", req.BookingID).First(&booking).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return dc.sendResponseWithLog(c, fiber.StatusNotFound, types.ApiResponse{
				Status:  fiber.StatusNotFound,
				Message: "Booking not found",
				Data:    nil,
			})
		}
		logger.Error("Failed to find booking", err)
		return dc.sendResponseWithLog(c, fiber.StatusInternalServerError, types.ApiResponse{
			Status:  fiber.StatusInternalServerError,
			Message: "Internal server error",
			Data:    nil,
		})
	}

	// Check if booking has a delivery phone set
	if booking.DeliveryPhone == nil {
		return dc.sendResponseWithLog(c, fiber.StatusBadRequest, types.ApiResponse{
			Status:  fiber.StatusBadRequest,
			Message: "No delivery phone found for this booking",
			Data:    nil,
		})
	}

	if booking.DeliveryPhoneConfirmedVerified {
		return dc.sendResponseWithLog(c, fiber.StatusBadRequest, types.ApiResponse{
			Status:  fiber.StatusBadRequest,
			Message: "Delivery phone is already confirmed",
			Data:    nil,
		})
	}

	// Verify OTP using OTP service
	otpSvc := otpService.NewOTPService(dc.DB)
	isValid, otpRecord, err := otpSvc.VerifyOTPWithDetails(*booking.DeliveryPhone, req.OTPCode, req.Purpose)
	if err != nil {
		logger.Error("Failed to verify delivery confirmation OTP", err)

		// If we have an OTP record, we can provide more detailed error information
		if otpRecord != nil {
			remainingAttempts := otpRecord.MaxRetries - otpRecord.RetryCount
			isBlocked := otpRecord.IsCurrentlyBlocked()
			isExpired := otpRecord.IsExpired()

			// Handle OTP expiration separately
			if isExpired {
				return dc.sendResponseWithLog(c, fiber.StatusBadRequest, types.ApiResponse{
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
				return dc.sendResponseWithLog(c, fiber.StatusTooManyRequests, types.ApiResponse{
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
			return dc.sendResponseWithLog(c, fiber.StatusBadRequest, types.ApiResponse{
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
		return dc.sendResponseWithLog(c, fiber.StatusInternalServerError, types.ApiResponse{
			Status:  fiber.StatusInternalServerError,
			Message: err.Error(), // Show the actual error message instead of generic
			Data:    nil,
		})
	}

	if !isValid {
		// This case should rarely happen now since we handle specific errors above
		return dc.sendResponseWithLog(c, fiber.StatusBadRequest, types.ApiResponse{
			Status:  fiber.StatusBadRequest,
			Message: "Invalid OTP",
			Data:    nil,
		})
	}

	// Encrypt OTP data for storage
	var deliveryPhoneConfirmedOTPEncrypted string

	if otpRecord != nil {
		// Encrypt the confirmed OTP (the OTP code that was verified)
		encryptedDeliveryPhoneConfirmedOTP, err := utils.EncryptData(otpRecord.OTPCode)
		if err != nil {
			logger.Error("Failed to encrypt delivery confirmation OTP", err)
			// Continue without encryption rather than failing
			deliveryPhoneConfirmedOTPEncrypted = ""
		} else {
			deliveryPhoneConfirmedOTPEncrypted = encryptedDeliveryPhoneConfirmedOTP
		}
	}
	fmt.Println("delivery phone confirmed OTP", deliveryPhoneConfirmedOTPEncrypted)
	// Mark delivery phone as confirmed and store encrypted OTP
	booking.DeliveryPhoneConfirmedVerified = true
	// Always assign the encrypted OTP field, even if it's empty
	booking.DeliveryPhoneConfirmedOTPEncrypted = &deliveryPhoneConfirmedOTPEncrypted
	//booking.Status = bookingModel.BookingStatusDelivered

	// Save the updated booking with explicit field updates
	if err := dc.DB.Save(&booking).Error; err != nil {
		logger.Error("Failed to update delivery phone confirmation status", err)
		return dc.sendResponseWithLog(c, fiber.StatusInternalServerError, types.ApiResponse{
			Status:  fiber.StatusInternalServerError,
			Message: "Failed to update confirmation status",
			Data:    nil,
		})
	}

	// Create booking status event
	bookingStatusEvent := bookingModel.BookingStatusEvent{
		BookingID: booking.ID,
		Status:    booking.Status,
		CreatedBy: strconv.FormatUint(uint64(postmanInfo.ID), 10),
	}

	if err := dc.DB.Create(&bookingStatusEvent).Error; err != nil {
		logger.Error("Failed to create booking status event", err)
		return dc.sendResponseWithLog(c, fiber.StatusInternalServerError, types.ApiResponse{
			Status:  fiber.StatusInternalServerError,
			Message: "Failed to create booking status event",
			Data:    nil,
		})
	}

	if err := booking_event.SnapshotBookingToEvent(dc.DB, &booking, "delivery_phone_confirmed", strconv.FormatUint(uint64(postmanInfo.ID), 10)); err != nil {
		logger.Error("Failed to write booking event (delivery_phone_confirmed)", err)
	}

	logger.Success(fmt.Sprintf("Delivery confirmation verified for booking ID: %d (Barcode: %s) by postman: %s", booking.ID, req.BookingID, postmanInfo.LegalName))

	responseData := map[string]interface{}{
		"booking":      booking,
		"verified":     true,
		"postman_id":   postmanInfo.ID,
		"postman_name": postmanInfo.LegalName,
	}

	return dc.sendResponseWithLog(c, fiber.StatusOK, types.ApiResponse{
		Status:  fiber.StatusOK,
		Message: "Delivery confirmation verified successfully",
		Data:    responseData,
	})
}

// VerifyApplicationID verifies the application ID for delivery
func (dc *DeliveryController) VerifyApplicationID(c *fiber.Ctx) error {
	var req deliveryTypes.VerifyApplicationIDRequest
	if err := c.BodyParser(&req); err != nil {
		logger.Error("Failed to parse request body", err)
		return dc.sendResponseWithLog(c, fiber.StatusBadRequest, types.ApiResponse{
			Status:  fiber.StatusBadRequest,
			Message: "Invalid request body",
			Data:    nil,
		})
	}

	// Validate request using the validation method from types
	if err := req.Validate(); err != nil {
		return dc.sendResponseWithLog(c, fiber.StatusBadRequest, types.ApiResponse{
			Status:  fiber.StatusBadRequest,
			Message: err.Error(),
			Data:    nil,
		})
	}

	// Get user authentication information (postman user)
	claims, ok := c.Locals("user").(map[string]interface{})
	if !ok {
		return dc.sendResponseWithLog(c, fiber.StatusUnauthorized, types.ApiResponse{
			Message: "Invalid user claims",
			Status:  fiber.StatusUnauthorized,
			Data:    nil,
		})
	}

	userUUID, ok := claims["uuid"].(string)
	if !ok || userUUID == "" {
		return dc.sendResponseWithLog(c, fiber.StatusUnauthorized, types.ApiResponse{
			Message: "User UUID not found in token",
			Status:  fiber.StatusUnauthorized,
			Data:    nil,
		})
	}

	// Get postman user info
	postmanInfo, err := utils.GetUserByUUID(userUUID)
	if err != nil {
		logger.Error("Error finding postman by UUID", err)
		status := fiber.StatusInternalServerError
		msg := "Database error"
		if err.Error() == "user not found" {
			status = fiber.StatusUnauthorized
			msg = "Postman not found"
		}
		return dc.sendResponseWithLog(c, status, types.ApiResponse{
			Message: msg,
			Status:  status,
			Data:    nil,
		})
	}

	// Find the booking by barcode
	var booking bookingModel.Booking
	if err := dc.DB.Preload("User").Where("barcode = ?", req.BookingID).First(&booking).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return dc.sendResponseWithLog(c, fiber.StatusNotFound, types.ApiResponse{
				Status:  fiber.StatusNotFound,
				Message: "Booking not found",
				Data:    nil,
			})
		}
		logger.Error("Failed to find booking", err)
		return dc.sendResponseWithLog(c, fiber.StatusInternalServerError, types.ApiResponse{
			Status:  fiber.StatusInternalServerError,
			Message: "Internal server error",
			Data:    nil,
		})
	}

	// Validate booking status allows application ID verification
	if booking.Status != bookingModel.BookingStatusReceivedByPostman {
		return dc.sendResponseWithLog(c, fiber.StatusBadRequest, types.ApiResponse{
			Status:  fiber.StatusBadRequest,
			Message: "Booking must be received by postman before application ID verification",
			Data:    nil,
		})
	}

	// Check if application ID is already verified
	if booking.DeliveryApplicationIDVerified {
		return dc.sendResponseWithLog(c, fiber.StatusBadRequest, types.ApiResponse{
			Status:  fiber.StatusBadRequest,
			Message: "Application ID is already verified for this booking",
			Data:    nil,
		})
	}

	// Verify the application ID matches the booking's AppOrOrderID
	if booking.AppOrOrderID != req.ApplicationID {
		return dc.sendResponseWithLog(c, fiber.StatusBadRequest, types.ApiResponse{
			Status:  fiber.StatusBadRequest,
			Message: "Application ID does not match the booking record",
			Data:    nil,
		})
	}

	// Mark application ID as verified
	booking.DeliveryApplicationIDVerified = true

	// Save the updated booking
	if err := dc.DB.Save(&booking).Error; err != nil {
		logger.Error("Failed to update application ID verification status", err)
		return dc.sendResponseWithLog(c, fiber.StatusInternalServerError, types.ApiResponse{
			Status:  fiber.StatusInternalServerError,
			Message: "Failed to update verification status",
			Data:    nil,
		})
	}

	// Create booking event for application ID verification
	if err := booking_event.SnapshotBookingToEvent(dc.DB, &booking, "application_id_verified", strconv.FormatUint(uint64(postmanInfo.ID), 10)); err != nil {
		logger.Error("Failed to write booking event (application_id_verified)", err)
	}

	logger.Success(fmt.Sprintf("Application ID verified for booking ID: %d (Barcode: %s) by postman: %s", booking.ID, req.BookingID, postmanInfo.LegalName))

	responseData := map[string]interface{}{
		"booking":        booking,
		"verified":       true,
		"application_id": req.ApplicationID,
		"postman_id":     postmanInfo.ID,
		"postman_name":   postmanInfo.LegalName,
	}

	return dc.sendResponseWithLog(c, fiber.StatusOK, types.ApiResponse{
		Status:  fiber.StatusOK,
		Message: "Application ID verified successfully",
		Data:    responseData,
	})
}

// UploadDeliveryPhoto handles photo upload for a booking during delivery
func (dc *DeliveryController) UploadDeliveryPhoto(c *fiber.Ctx) error {
	// Get booking ID from form data
	bookingIDStr := c.FormValue("booking_id")
	if bookingIDStr == "" {
		return dc.sendResponseWithLog(c, fiber.StatusBadRequest, types.ApiResponse{
			Status:  fiber.StatusBadRequest,
			Message: "Booking ID is required",
			Data:    nil,
		})
	}

	// Get user authentication information (postman user)
	claims, ok := c.Locals("user").(map[string]interface{})
	if !ok {
		return dc.sendResponseWithLog(c, fiber.StatusUnauthorized, types.ApiResponse{
			Message: "Invalid user claims",
			Status:  fiber.StatusUnauthorized,
			Data:    nil,
		})
	}

	userUUID, ok := claims["uuid"].(string)
	if !ok || userUUID == "" {
		return dc.sendResponseWithLog(c, fiber.StatusUnauthorized, types.ApiResponse{
			Message: "User UUID not found in token",
			Status:  fiber.StatusUnauthorized,
			Data:    nil,
		})
	}

	// Get postman user info
	postmanInfo, err := utils.GetUserByUUID(userUUID)
	if err != nil {
		logger.Error("Error finding postman by UUID", err)
		status := fiber.StatusInternalServerError
		msg := "Database error"
		if err.Error() == "user not found" {
			status = fiber.StatusUnauthorized
			msg = "Postman not found"
		}
		return dc.sendResponseWithLog(c, status, types.ApiResponse{
			Message: msg,
			Status:  status,
			Data:    nil,
		})
	}

	// Find the booking by barcode
	var booking bookingModel.Booking
	if err := dc.DB.Preload("User").Where("barcode = ?", bookingIDStr).First(&booking).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return dc.sendResponseWithLog(c, fiber.StatusNotFound, types.ApiResponse{
				Status:  fiber.StatusNotFound,
				Message: "Booking not found",
				Data:    nil,
			})
		}
		logger.Error("Failed to find booking", err)
		return dc.sendResponseWithLog(c, fiber.StatusInternalServerError, types.ApiResponse{
			Status:  fiber.StatusInternalServerError,
			Message: "Internal server error",
			Data:    nil,
		})
	}

	// Check if photo is already uploaded
	if booking.UploadPhoto != nil && *booking.UploadPhoto != "" {
		// Check if the file actually exists on the filesystem
		if _, err := os.Stat(*booking.UploadPhoto); err == nil {
			return dc.sendResponseWithLog(c, fiber.StatusConflict, types.ApiResponse{
				Status:  fiber.StatusConflict,
				Message: "Photo already uploaded for this booking",
				Data: fiber.Map{
					"booking_id":     booking.ID,
					"existing_photo": *booking.UploadPhoto,
					"uploaded_at":    booking.UpdatedAt,
				},
			})
		} else {
			logger.Warning(fmt.Sprintf("Photo path exists in database but file not found on filesystem for booking %d: %s", booking.ID, *booking.UploadPhoto))
		}
	}

	// Get the uploaded file
	file, err := c.FormFile("photo")
	if err != nil {
		return dc.sendResponseWithLog(c, fiber.StatusBadRequest, types.ApiResponse{
			Status:  fiber.StatusBadRequest,
			Message: "Photo file is required",
			Data:    nil,
		})
	}

	// Validate file type (only allow common image formats)
	allowedTypes := map[string]bool{
		"image/jpeg": true,
		"image/jpg":  true,
		"image/png":  true,
		"image/gif":  true,
		"image/webp": true,
	}

	fileType := file.Header.Get("Content-Type")
	if !allowedTypes[fileType] {
		return dc.sendResponseWithLog(c, fiber.StatusBadRequest, types.ApiResponse{
			Status:  fiber.StatusBadRequest,
			Message: "Invalid file type. Only JPEG, PNG, GIF, and WebP images are allowed",
			Data:    nil,
		})
	}

	// Validate file size (max 10MB)
	maxSize := int64(10 << 20) // 10MB
	if file.Size > maxSize {
		return dc.sendResponseWithLog(c, fiber.StatusBadRequest, types.ApiResponse{
			Status:  fiber.StatusBadRequest,
			Message: "File size too large. Maximum size is 10MB",
			Data:    nil,
		})
	}

	// Create upload directory if it doesn't exist
	uploadDir := "./upload_photos"
	if err := os.MkdirAll(uploadDir, os.ModePerm); err != nil {
		logger.Error("Failed to create upload directory", err)
		return dc.sendResponseWithLog(c, fiber.StatusInternalServerError, types.ApiResponse{
			Status:  fiber.StatusInternalServerError,
			Message: "Failed to create upload directory",
			Data:    nil,
		})
	}

	// Generate unique filename
	fileExt := strings.ToLower(filepath.Ext(file.Filename))
	if fileExt == "" {
		// If no extension, try to determine from content type
		switch fileType {
		case "image/jpeg":
			fileExt = ".jpg"
		case "image/png":
			fileExt = ".png"
		case "image/gif":
			fileExt = ".gif"
		case "image/webp":
			fileExt = ".webp"
		default:
			fileExt = ".jpg"
		}
	}

	timestamp := time.Now().Format("20060102_150405")
	filename := fmt.Sprintf("booking_%s%s", timestamp, fileExt)
	filePath := fmt.Sprintf("%s/%s", uploadDir, filename)

	// Save the file
	if err := c.SaveFile(file, filePath); err != nil {
		logger.Error("Failed to save uploaded file", err)
		return dc.sendResponseWithLog(c, fiber.StatusInternalServerError, types.ApiResponse{
			Status:  fiber.StatusInternalServerError,
			Message: "Failed to save uploaded file",
			Data:    nil,
		})
	}

	// Update booking with photo path
	if err := dc.DB.Model(&booking).Updates(bookingModel.Booking{
		UploadPhoto: &filePath,
		UpdatedAt:   time.Now(),
	}).Error; err != nil {
		logger.Error("Failed to update booking with photo path", err)
		// Try to delete the uploaded file if database update fails
		os.Remove(filePath)
		return dc.sendResponseWithLog(c, fiber.StatusInternalServerError, types.ApiResponse{
			Status:  fiber.StatusInternalServerError,
			Message: "Failed to update booking with photo information",
			Data:    nil,
		})
	}

	// Create booking event for photo upload
	if err := booking_event.SnapshotBookingToEvent(dc.DB, &booking, "delivery_photo_uploaded", strconv.FormatUint(uint64(postmanInfo.ID), 10)); err != nil {
		logger.Error("Failed to write booking event (delivery_photo_uploaded)", err)
	}

	logger.Success(fmt.Sprintf("Delivery photo uploaded for booking ID: %d (Barcode: %s) by postman: %s", booking.ID, bookingIDStr, postmanInfo.LegalName))

	return dc.sendResponseWithLog(c, fiber.StatusOK, types.ApiResponse{
		Status:  fiber.StatusOK,
		Message: "Photo uploaded successfully",
		Data: fiber.Map{
			"booking_id":   booking.ID,
			"photo_path":   filePath,
			"filename":     filename,
			"postman_id":   postmanInfo.ID,
			"postman_name": postmanInfo.LegalName,
		},
	})
}

// ItemDetails handles POST /delivered/itemdetails
func (dc *DeliveryController) ItemDetails(c *fiber.Ctx) error {
	type request struct {
		Barcode string `json:"barcode"`
	}
	var req request
	if err := c.BodyParser(&req); err != nil {
		return dc.sendResponseWithLog(c, fiber.StatusBadRequest, types.ApiResponse{
			Status:  fiber.StatusBadRequest,
			Message: "Invalid request body",
			Data:    nil,
		})
	}
	if req.Barcode == "" {
		return dc.sendResponseWithLog(c, fiber.StatusBadRequest, types.ApiResponse{
			Status:  fiber.StatusBadRequest,
			Message: "Barcode is required",
			Data:    nil,
		})
	}
	claims, ok := c.Locals("user").(map[string]interface{})
	if !ok {
		return dc.sendResponseWithLog(c, fiber.StatusUnauthorized, types.ApiResponse{
			Status:  fiber.StatusUnauthorized,
			Message: "Invalid user claims",
			Data:    nil,
		})
	}
	userUUID, ok := claims["uuid"].(string)
	if !ok || userUUID == "" {
		return dc.sendResponseWithLog(c, fiber.StatusUnauthorized, types.ApiResponse{
			Status:  fiber.StatusUnauthorized,
			Message: "User UUID not found in token",
			Data:    nil,
		})
	}
	// Optionally, get user info if needed (e.g., for ID)
	postmanInfo, err := utils.GetUserByUUID(userUUID)
	if err != nil {
		return dc.sendResponseWithLog(c, fiber.StatusUnauthorized, types.ApiResponse{
			Status:  fiber.StatusUnauthorized,
			Message: "Postman not found",
			Data:    nil,
		})
	}
	var booking bookingModel.Booking
	// Convert postmanInfo.ID to string for updated_by comparison
	updatedByStr := fmt.Sprintf("%v", postmanInfo.ID)
	err = dc.DB.Where("barcode = ? AND status = ? AND updated_by = ?", req.Barcode, bookingModel.BookingStatusReceivedByPostman, updatedByStr).First(&booking).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return dc.sendResponseWithLog(c, fiber.StatusNotFound, types.ApiResponse{
				Status:  fiber.StatusNotFound,
				Message: "Booking not found",
				Data:    nil,
			})
		}
		return dc.sendResponseWithLog(c, fiber.StatusInternalServerError, types.ApiResponse{
			Status:  fiber.StatusInternalServerError,
			Message: "Internal server error",
			Data:    nil,
		})
	}
	return dc.sendResponseWithLog(c, fiber.StatusOK, types.ApiResponse{
		Status:  fiber.StatusOK,
		Message: "Booking details found",
		Data:    booking,
	})
}
