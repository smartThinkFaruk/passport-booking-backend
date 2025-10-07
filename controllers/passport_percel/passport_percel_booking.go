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
			// Log the error and return the actual error message - don't create parcel without barcode
			logger.Error("Failed to generate barcode", err)
			response := types.ApiResponse{
				Status:  fiber.StatusInternalServerError,
				Message: fmt.Sprintf("Failed to generate barcode: %v", err),
				Data:    nil,
			}
			return pbc.sendResponseWithLog(c, fiber.StatusInternalServerError, response)
		}
		barcode = generatedBarcode
	} else {
		// No authorization header provided
		response := types.ApiResponse{
			Status:  fiber.StatusUnauthorized,
			Message: "Authorization header required for barcode generation",
			Data:    nil,
		}
		return pbc.sendResponseWithLog(c, fiber.StatusUnauthorized, response)
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
		UpdatedBy:     fmt.Sprintf("%d", userID), // Convert uint to string
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
		parcelBooking.UpdatedBy = fmt.Sprintf("%d", userID)

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

// StoreSubmit handles submitting a parcel booking to external DMS API
func (pbc *ParcelBookingController) StoreSubmit(c *fiber.Ctx) error {
	var request parcel_booking_types.StoreSubmitRequest

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

	// Check if the parcel booking is in pending status
	if parcelBooking.CurrentStatus != string(parcel_booking.ParcelBookingStatusPending) {
		response := types.ApiResponse{
			Status:  fiber.StatusBadRequest,
			Message: "Parcel booking must be in pending status to submit",
			Data:    nil,
		}
		return pbc.sendResponseWithLog(c, fiber.StatusBadRequest, response)
	}

	// Check if the parcel booking is already booked
	if parcelBooking.CurrentStatus == string(parcel_booking.ParcelBookingStatusBooked) {
		response := types.ApiResponse{
			Status:  fiber.StatusOK,
			Message: "Parcel booking is already submitted",
			Data:    parcelBooking,
		}
		return pbc.sendResponseWithLog(c, fiber.StatusOK, response)
	}

	// Call external DMS API for booking
	authHeader := c.Get("Authorization")
	if authHeader == "" {
		response := types.ApiResponse{
			Status:  fiber.StatusUnauthorized,
			Message: "Authorization header required",
			Data:    nil,
		}
		return pbc.sendResponseWithLog(c, fiber.StatusUnauthorized, response)
	}

	dmsBody, dmsStatusCode, err := pbc.BookingDms(authHeader, request.Barcode, parcelBooking.ID)
	if err != nil {
		// Log the error with more details
		//logger.Error(fmt.Sprintf("DMS booking failed for barcode %s: %v", request.Barcode, err))
		logger.Error("DMS booking failed", err)
		response := types.ApiResponse{
			Status:  fiber.StatusInternalServerError,
			Message: fmt.Sprintf("Failed to call external booking API: %v", err),
			Data:    nil,
		}
		return pbc.sendResponseWithLog(c, fiber.StatusInternalServerError, response)
	}

	// Handle DMS API response
	if dmsStatusCode != http.StatusOK && dmsStatusCode != http.StatusCreated {
		// Log the DMS response for debugging
		//logger.Error(fmt.Sprintf("DMS API returned status %d for barcode %s. Response: %s", dmsStatusCode, request.Barcode, string(dmsBody)))
		logger.Error("Failed to Booking", err)
		response := types.ApiResponse{
			Status:  fiber.StatusInternalServerError,
			Message: fmt.Sprintf("DMS API returned status %d", dmsStatusCode),
			Data:    string(dmsBody),
		}
		return pbc.sendResponseWithLog(c, fiber.StatusInternalServerError, response)
	}

	// Log successful DMS response
	logger.Info(fmt.Sprintf("DMS booking successful for barcode %s. Status: %d", request.Barcode, dmsStatusCode))

	// Update parcel booking status to booked
	now := time.Now()
	parcelBooking.CurrentStatus = string(parcel_booking.ParcelBookingStatusBooked)
	parcelBooking.BookingDate = &now
	parcelBooking.UpdatedBy = fmt.Sprintf("%d", userID)

	if err := pbc.DB.Save(&parcelBooking).Error; err != nil {
		response := types.ApiResponse{
			Status:  fiber.StatusInternalServerError,
			Message: "Failed to update parcel booking status",
			Data:    nil,
		}
		return pbc.sendResponseWithLog(c, fiber.StatusInternalServerError, response)
	}

	// Create parcel booking status event for booked status
	statusEvent := parcel_booking.ParcelBookingStatusEvent{
		ParcelBookingID: parcelBooking.ID,
		Status:          string(parcel_booking.ParcelBookingStatusBooked),
		CreatedBy:       userID,
	}

	if err := pbc.DB.Create(&statusEvent).Error; err != nil {
		// Log the error but don't fail the entire operation
		logger.Error(fmt.Sprintf("Failed to create parcel booking status event for parcel_booking_id: %d", parcelBooking.ID), err)
	}

	// Load the user relationship for response
	pbc.DB.Preload("User").First(&parcelBooking, parcelBooking.ID)

	response := types.ApiResponse{
		Status:  fiber.StatusOK,
		Message: "Parcel booking submitted successfully",
		Data:    parcelBooking,
	}

	return pbc.sendResponseWithLog(c, fiber.StatusOK, response)
}

// BookingDms calls the external DMS API to book a parcel
func (pbc *ParcelBookingController) BookingDms(authHeader, barcode string, parcelBookingID uint) ([]byte, int, error) {
	baseURL := os.Getenv("DMS_BASE_URL")
	url := fmt.Sprintf("%s/dms/book/article/", baseURL)

	// Find the parcel booking by ID with user relationship
	var parcelBooking parcel_booking.ParcelBooking
	if err := pbc.DB.
		Preload("User").
		Where("id = ?", parcelBookingID).
		First(&parcelBooking).Error; err != nil {
		return nil, 0, fmt.Errorf("parcel booking not found: %v", err)
	}

	// Check if parcel booking exists and is in pending status
	if parcelBooking.ID == 0 {
		return nil, 0, fmt.Errorf("parcel booking not found")
	}

	if parcelBooking.CurrentStatus != string(parcel_booking.ParcelBookingStatusPending) {
		return nil, 0, fmt.Errorf("parcel booking is not in pending status")
	}

	// Check if required user data is loaded
	if parcelBooking.User.Uuid == "" {
		return nil, 0, fmt.Errorf("user information not found for parcel booking")
	}

	// Create the actual request body structure
	payload := map[string]interface{}{
		"ad_pod_id":        "1",
		"article_desc":     "Passport Delivery",
		"article_price":    100,
		"barcode":          barcode,
		"city_post_status": "No",
		"delivery_branch":  "100000",
		"emts_branch_code": "100000",
		"height":           10,
		"hnddevice":        "web",
		"image_pod":        "0",
		"image_src":        "No",
		"insurance_price":  "0",
		"is_bulk_mail":     "No",
		"isCharge":         "Yes",
		"is_city_post":     "No",
		"is_international": false,
		"isStation":        "No",
		"length":           10,
		"receiver": map[string]interface{}{
			"address_type":   "home",
			"country":        "Bangladesh",
			"district":       parcelBooking.RpoName, // Using RpoName as district
			"division":       "",                    // Can be enhanced if needed
			"phone_number":   parcelBooking.Phone,
			"police_station": "",
			"post_office":    parcelBooking.PostCode,
			"street_address": parcelBooking.RpoAddress,
			"user_uuid":      parcelBooking.User.Uuid,
			"username":       parcelBooking.User.Username,
			"zone":           "Zone 1",
		},
		"sender": map[string]interface{}{
			"address_type":   "office",
			"country":        "Bangladesh",
			"district":       "Dhaka",
			"division":       "Dhaka",
			"phone_number":   "018XXXXXXXX",
			"police_station": "Gulshan",
			"post_office":    "Gulshan",
			"street_address": "456, Gulshan, Dhaka",
			"user_uuid":      parcelBooking.User.Uuid,
			"username":       "passport-office",
			"zone":           "Zone 2",
		},
		"service_name": "letter",
		"set_ad":       "No",
		"vas_type":     "Registry",
		"vp_amount":    "0",
		"vp_service":   "No",
		"weight":       100,
		"width":        10,
	}

	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to marshal payload: %v", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonPayload))
	if err != nil {
		return nil, 0, fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", authHeader)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to call booking API: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to read response: %v", err)
	}

	return body, resp.StatusCode, nil
}

// Index handles listing parcel bookings with pagination and filtering
func (pbc *ParcelBookingController) Index(c *fiber.Ctx) error {
	// Default pagination
	page := c.QueryInt("page", 1)
	limit := c.QueryInt("limit", 10)
	offset := (page - 1) * limit

	// Get query parameters for filtering
	barcode := c.Query("barcode")
	startDateStr := c.Query("start_date")
	endDateStr := c.Query("end_date")
	status := c.Query("status")

	// Build the query
	query := pbc.DB.Model(&parcel_booking.ParcelBooking{})

	if barcode != "" {
		query = query.Where("barcode LIKE ?", "%"+barcode+"%")
	}

	if startDateStr != "" && endDateStr != "" {
		startDate, err1 := time.Parse("2006-01-02", startDateStr)
		endDate, err2 := time.Parse("2006-01-02", endDateStr)
		if err1 == nil && err2 == nil {
			query = query.Where("created_at BETWEEN ? AND ?", startDate, endDate.Add(24*time.Hour))
		}
	}

	if status != "" {
		query = query.Where("current_status = ?", status)
	}

	// Get total count for pagination
	var total int64
	if err := query.Count(&total).Error; err != nil {
		response := types.ApiResponse{
			Status:  fiber.StatusInternalServerError,
			Message: "Failed to count parcel bookings",
			Data:    nil,
		}
		return pbc.sendResponseWithLog(c, fiber.StatusInternalServerError, response)
	}

	// Get parcel bookings with pagination
	var parcelBookings []parcel_booking.ParcelBooking
	if err := query.Preload("User").Offset(offset).Limit(limit).Order("created_at desc").Find(&parcelBookings).Error; err != nil {
		response := types.ApiResponse{
			Status:  fiber.StatusInternalServerError,
			Message: "Failed to retrieve parcel bookings",
			Data:    nil,
		}
		return pbc.sendResponseWithLog(c, fiber.StatusInternalServerError, response)
	}

	response := types.ApiResponse{
		Status:  fiber.StatusOK,
		Message: "Parcel bookings retrieved successfully",
		Data: fiber.Map{
			"data":      parcelBookings,
			"total":     total,
			"page":      page,
			"limit":     limit,
			"last_page": (total + int64(limit) - 1) / int64(limit),
		},
	}

	return pbc.sendResponseWithLog(c, fiber.StatusOK, response)
}
