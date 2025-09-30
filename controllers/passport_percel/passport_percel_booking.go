package passport_percel

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"passport-booking/logger"
	"passport-booking/models/parcel_booking"
	"passport-booking/types"
	parcel_booking_types "passport-booking/types/parcel_booking"
	"passport-booking/utils"
	"time"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// ParcelBookingController handles parcel booking related HTTP requests
type ParcelBookingController struct {
	DB     *gorm.DB
	Logger *logger.AsyncLogger
}

// NewParcelBookingController creates a new parcel booking controller
func NewParcelBookingController(db *gorm.DB, asyncLogger *logger.AsyncLogger) *ParcelBookingController {
	return &ParcelBookingController{
		DB:     db,
		Logger: asyncLogger,
	}
}

// Helper function to log API requests and responses
func (pbc *ParcelBookingController) logAPIRequest(c *fiber.Ctx) {
	logEntry := utils.CreateSanitizedLogEntry(c)
	pbc.Logger.Log(logEntry)
}

// Helper function to send response and log in one call
func (pbc *ParcelBookingController) sendResponseWithLog(c *fiber.Ctx, status int, response types.ApiResponse) error {
	result := c.Status(status).JSON(response)
	pbc.logAPIRequest(c)
	return result
}

// Store handles creating a new parcel booking or returning existing one
func (pbc *ParcelBookingController) Store(c *fiber.Ctx) error {
	var request parcel_booking_types.StoreParcelBookingRequest

	// Parse request body
	if err := c.BodyParser(&request); err != nil {
		response := types.ApiResponse{
			Status:  fiber.StatusBadRequest,
			Message: "Invalid request format",
			Data:    nil,
		}
		return pbc.sendResponseWithLog(c, fiber.StatusBadRequest, response)
	}

	// Get user authentication information (following booking.go pattern)
	claims, ok := c.Locals("user").(map[string]interface{})
	if !ok {
		response := types.ApiResponse{
			Status:  fiber.StatusUnauthorized,
			Message: "Invalid user claims",
			Data:    nil,
		}
		return pbc.sendResponseWithLog(c, fiber.StatusUnauthorized, response)
	}

	userUUID, ok := claims["uuid"].(string)
	if !ok || userUUID == "" {
		response := types.ApiResponse{
			Status:  fiber.StatusUnauthorized,
			Message: "User UUID not found in token",
			Data:    nil,
		}
		return pbc.sendResponseWithLog(c, fiber.StatusUnauthorized, response)
	}

	userInfo, err := utils.GetUserByUUID(userUUID)
	if err != nil {
		status := fiber.StatusInternalServerError
		msg := "Database error"
		if err.Error() == "user not found" {
			status = fiber.StatusUnauthorized
			msg = "User not found"
		}
		response := types.ApiResponse{
			Status:  status,
			Message: msg,
			Data:    nil,
		}
		return pbc.sendResponseWithLog(c, status, response)
	}

	userID := uint(userInfo.ID)

	// Check if there's already an existing parcel with initial or pending status for this RPO
	var existingParcel parcel_booking.ParcelBooking
	result := pbc.DB.Where("user_id = ? AND post_code = ? AND current_status IN ?",
		userID, request.PostCode, []string{string(parcel_booking.ParcelBookingStatusInitial), string(parcel_booking.ParcelBookingStatusPending)}).
		First(&existingParcel)

	// If found existing parcel with initial or pending status, return it
	if result.Error == nil {
		response := types.ApiResponse{
			Status:  fiber.StatusOK,
			Message: "Existing parcel booking found",
			Data:    existingParcel,
		}
		return pbc.sendResponseWithLog(c, fiber.StatusOK, response)
	}

	// Generate barcode from API before creating the parcel booking
	var barcode string
	authHeader := c.Get("Authorization")
	if authHeader != "" {
		generatedBarcode, err := pbc.getBarcodeFromAPI(authHeader)
		if err != nil {
			// Log the error but don't fail the entire operation
			logger.Error("Failed to generate barcode", err)
		} else {
			barcode = generatedBarcode
		}
	}

	// If no existing parcel found, create a new one with barcode
	newParcel := parcel_booking.ParcelBooking{
		UserID:        uint(userID),
		RpoAddress:    request.RpoAddress,
		Phone:         request.Phone,
		PostCode:      request.PostCode,
		RpoName:       request.RpoName,
		Barcode:       barcode, // Include barcode in initial creation
		CurrentStatus: string(parcel_booking.ParcelBookingStatusInitial),
		ServiceType:   "passport", // Default service type
		Insured:       false,
		PushStatus:    0,
	}

	// Create the new parcel booking
	if err := pbc.DB.Create(&newParcel).Error; err != nil {
		response := types.ApiResponse{
			Status:  fiber.StatusInternalServerError,
			Message: "Failed to create parcel booking",
			Data:    nil,
		}
		return pbc.sendResponseWithLog(c, fiber.StatusInternalServerError, response)
	}

	// Create initial parcel booking status event
	initialEvent := parcel_booking.ParcelBookingStatusEvent{
		ParcelBookingID: newParcel.ID,
		Status:          string(parcel_booking.ParcelBookingStatusInitial),
		CreatedBy:       userID,
	}

	if err := pbc.DB.Create(&initialEvent).Error; err != nil {
		// Log the error but don't fail the entire operation
		// since the parcel booking was created successfully
		logger.Error(fmt.Sprintf("Failed to create initial parcel booking status event for parcel_booking_id: %d", newParcel.ID), err)
	}

	// Load the user relationship
	pbc.DB.Preload("User").First(&newParcel, newParcel.ID)

	response := types.ApiResponse{
		Status:  fiber.StatusCreated,
		Message: "Parcel booking created successfully",
		Data:    newParcel,
	}

	return pbc.sendResponseWithLog(c, fiber.StatusCreated, response)
}

// getBarcodeFromAPI generates a barcode by calling the external DMS API
func (pbc *ParcelBookingController) getBarcodeFromAPI(authHeader string) (string, error) {
	baseURL := os.Getenv("DMS_BASE_URL")
	url := fmt.Sprintf("%s/dms/api/get-barcode/", baseURL)

	payload := map[string]interface{}{
		"service_type": "letter",
	}

	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("failed to marshal payload: %v", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonPayload))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", authHeader)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to call barcode API: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %v", err)
	}

	// Accept both 200 and 201 as success status codes
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return "", fmt.Errorf("barcode API returned status %d: %s", resp.StatusCode, string(body))
	}

	var barcodeResp map[string]interface{}
	if err := json.Unmarshal(body, &barcodeResp); err != nil {
		return "", fmt.Errorf("failed to parse barcode response: %v", err)
	}

	// Extract barcode from response
	barcode, ok := barcodeResp["barcode"].(string)
	if !ok {
		return "", fmt.Errorf("barcode not found in response")
	}

	return barcode, nil
}

// StorePendingBooking handles updating a parcel booking status to pending
func (pbc *ParcelBookingController) StorePendingBooking(c *fiber.Ctx) error {
	var request parcel_booking_types.StorePendingBookingRequest

	// Parse request body
	if err := c.BodyParser(&request); err != nil {
		response := types.ApiResponse{
			Status:  fiber.StatusBadRequest,
			Message: "Invalid request format",
			Data:    nil,
		}
		return pbc.sendResponseWithLog(c, fiber.StatusBadRequest, response)
	}

	// Get user authentication information
	claims, ok := c.Locals("user").(map[string]interface{})
	if !ok {
		response := types.ApiResponse{
			Status:  fiber.StatusUnauthorized,
			Message: "Invalid user claims",
			Data:    nil,
		}
		return pbc.sendResponseWithLog(c, fiber.StatusUnauthorized, response)
	}

	userUUID, ok := claims["uuid"].(string)
	if !ok || userUUID == "" {
		response := types.ApiResponse{
			Status:  fiber.StatusUnauthorized,
			Message: "User UUID not found in token",
			Data:    nil,
		}
		return pbc.sendResponseWithLog(c, fiber.StatusUnauthorized, response)
	}

	userInfo, err := utils.GetUserByUUID(userUUID)
	if err != nil {
		status := fiber.StatusInternalServerError
		msg := "Database error"
		if err.Error() == "user not found" {
			status = fiber.StatusUnauthorized
			msg = "User not found"
		}
		response := types.ApiResponse{
			Status:  status,
			Message: msg,
			Data:    nil,
		}
		return pbc.sendResponseWithLog(c, status, response)
	}

	userID := uint(userInfo.ID)

	// Find the parcel booking by barcode
	var parcelBooking parcel_booking.ParcelBooking
	result := pbc.DB.Where("barcode = ?", request.Barcode).First(&parcelBooking)
	if result.Error != nil {
		response := types.ApiResponse{
			Status:  fiber.StatusNotFound,
			Message: "Parcel booking not found",
			Data:    nil,
		}
		return pbc.sendResponseWithLog(c, fiber.StatusNotFound, response)
	}

	// Check if the parcel booking is already in pending status
	var message string
	var statusCode int
	if parcelBooking.CurrentStatus == string(parcel_booking.ParcelBookingStatusPending) {
		message = "Already pending this item"
		statusCode = fiber.StatusOK
	} else {
		// Update the parcel booking status to pending and set pending date
		now := time.Now()
		parcelBooking.CurrentStatus = string(parcel_booking.ParcelBookingStatusPending)
		parcelBooking.PendingDate = &now

		if err := pbc.DB.Save(&parcelBooking).Error; err != nil {
			response := types.ApiResponse{
				Status:  fiber.StatusInternalServerError,
				Message: "Failed to update parcel booking",
				Data:    nil,
			}
			return pbc.sendResponseWithLog(c, fiber.StatusInternalServerError, response)
		}
		message = "Parcel booking updated to pending status successfully"
		statusCode = fiber.StatusOK
	}

	// Always create parcel booking status event (whether already pending or newly updated)
	statusEvent := parcel_booking.ParcelBookingStatusEvent{
		ParcelBookingID: parcelBooking.ID,
		Status:          string(parcel_booking.ParcelBookingStatusPending),
		CreatedBy:       userID,
	}

	if err := pbc.DB.Create(&statusEvent).Error; err != nil {
		// Log the error but don't fail the entire operation
		logger.Error(fmt.Sprintf("Failed to create parcel booking status event for parcel_booking_id: %d", parcelBooking.ID), err)
	}

	// Load the user relationship for response
	pbc.DB.Preload("User").First(&parcelBooking, parcelBooking.ID)

	response := types.ApiResponse{
		Status:  statusCode,
		Message: message,
		Data:    parcelBooking,
	}

	return pbc.sendResponseWithLog(c, statusCode, response)
}
