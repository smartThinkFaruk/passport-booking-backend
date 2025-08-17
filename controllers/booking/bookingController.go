package booking

import (
	"fmt"
	"passport-booking/database"
	"passport-booking/logger"
	addressModel "passport-booking/models/address"
	bookingModel "passport-booking/models/booking"
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
