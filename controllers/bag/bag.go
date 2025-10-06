package bag

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
	"io"
	"net/http"
	"os"
	"passport-booking/database"
	"passport-booking/logger"
	bookingModel "passport-booking/models/booking"
	"passport-booking/models/user"
	"passport-booking/services/booking_event"
	"passport-booking/types"
	bagType "passport-booking/types/bag"
	"passport-booking/utils"
	"time"
)

type BagController struct {
	DB             *gorm.DB
	Logger         *logger.AsyncLogger
	loggerInstance *logger.AsyncLogger
}

// NewBagController creates a new bag controller
func NewBagController(db *gorm.DB, asyncLogger *logger.AsyncLogger) *BagController {
	return &BagController{
		DB:             db,
		Logger:         asyncLogger,
		loggerInstance: asyncLogger,
	}
}

// Helper function to log API requests and responses
func (bc *BagController) logAPIRequest(c *fiber.Ctx) {
	logEntry := utils.CreateSanitizedLogEntry(c)
	bc.loggerInstance.Log(logEntry)
}

// Helper function to send response and log in one call
func (bc *BagController) sendResponseWithLog(c *fiber.Ctx, status int, response types.ApiResponse) error {
	result := c.Status(status).JSON(response)
	bc.logAPIRequest(c)
	return result
}

func logRequest(c *fiber.Ctx, responseBody string, requestBody string) {
	// Create AsyncLogger instance and start processor
	asyncLogger := logger.NewAsyncLogger(database.DB)
	go asyncLogger.ProcessLog()

	logEntry := utils.CreateSanitizedLogEntryWithCustomBody(c, requestBody, responseBody)
	asyncLogger.Log(logEntry)
}

func GetBranchList(c *fiber.Ctx) error {
	// Capture request body before any processing
	requestBody := string(c.Body())

	authHeader := c.Get("Authorization")
	if authHeader == "" {
		errorResponse := fiber.Map{"error": "Authorization header is required"}
		c.Status(fiber.StatusUnauthorized).JSON(errorResponse)
		logRequest(c, "", requestBody)
		return nil
	}
	baseURL := os.Getenv("EKDAK_BASE_URL")
	if baseURL == "" {
		errorResponse := fiber.Map{"error": "DMS_BASE_URL not set in environment"}
		c.Status(fiber.StatusInternalServerError).JSON(errorResponse)
		logRequest(c, "", requestBody)
		return nil
	}

	// Forward query params from user request
	query := c.Context().QueryArgs().String()
	url := fmt.Sprintf("%s/v1/dms-legacy-core-logs/search-dms-branch/", baseURL)
	if query != "" {
		url = fmt.Sprintf("%s?%s", url, query)
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		errorResponse := fiber.Map{"error": "Failed to create request"}
		c.Status(fiber.StatusInternalServerError).JSON(errorResponse)
		logRequest(c, "", requestBody)
		return nil
	}
	req.Header.Set("Authorization", authHeader)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		errorResponse := fiber.Map{"error": "Failed to call external API"}
		c.Status(fiber.StatusBadGateway).JSON(errorResponse)
		logRequest(c, "", requestBody)
		return nil
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		errorResponse := fiber.Map{"error": "Failed to read response"}
		c.Status(fiber.StatusInternalServerError).JSON(errorResponse)
		logRequest(c, "", requestBody)
		return nil
	}

	// Send successful response and log it
	c.Status(resp.StatusCode).Send(body)
	logRequest(c, string(body), requestBody)
	return nil
}

func GetOperatorList(c *fiber.Ctx) error {
	// Capture request body before any processing
	requestBody := string(c.Body())

	authHeader := c.Get("Authorization")
	if authHeader == "" {
		errorResponse := types.ApiResponse{
			Message: "Authorization header is required",
			Status:  fiber.StatusUnauthorized,
		}
		c.Status(fiber.StatusUnauthorized).JSON(errorResponse)
		logRequest(c, "", requestBody)
		return nil
	}

	db := database.DB
	if db == nil {
		fmt.Println("DEBUG: db not found in context")
		errorResponse := types.ApiResponse{
			Message: "Database connection not found in context",
			Status:  fiber.StatusInternalServerError,
		}
		c.Status(fiber.StatusInternalServerError).JSON(errorResponse)
		logRequest(c, "", requestBody)
		return nil
	}

	var users []user.User
	if err := db.Find(&users).Error; err != nil {
		errorResponse := types.ApiResponse{
			Message: "Failed to fetch operators",
			Status:  fiber.StatusInternalServerError,
		}
		c.Status(fiber.StatusInternalServerError).JSON(errorResponse)
		logRequest(c, "", requestBody)
		return nil
	}

	// Filter users who have the "passport-booking.operator.full-permit" permission
	var operators []user.User
	targetPermission := "passport-booking.operator.full-permit"

	for _, u := range users {
		for _, permission := range u.Permissions {
			if permission == targetPermission {
				operators = append(operators, u)
				break // Found the permission, no need to check other permissions for this user
			}
		}
	}

	successResponse := types.ApiResponse{
		Message: "Operators retrieved successfully",
		Status:  fiber.StatusOK,
		Data:    operators,
	}
	c.JSON(successResponse)

	// Serialize the response properly for logging
	responseBytes, _ := json.Marshal(successResponse)
	logRequest(c, string(responseBytes), requestBody)
	return nil
}

func CreateBranchMapping(c *fiber.Ctx) error {
	// Capture the raw request body first
	rawRequestBody := string(c.Body())

	var reqBody bagType.BranchMappingRequest

	authHeader := c.Get("Authorization")
	if authHeader == "" {
		errorResponse := types.ApiResponse{
			Message: "Authorization header is required",
			Status:  fiber.StatusUnauthorized,
		}
		c.Status(fiber.StatusUnauthorized).JSON(errorResponse)
		logRequest(c, "", rawRequestBody)
		return nil
	}

	if err := c.BodyParser(&reqBody); err != nil {
		errorResponse := types.ApiResponse{
			Message: "Invalid request body",
			Status:  fiber.StatusBadRequest,
		}
		c.Status(fiber.StatusBadRequest).JSON(errorResponse)
		logRequest(c, "", rawRequestBody)
		return nil
	}

	// Convert parsed reqBody to JSON string for logging (with actual values)
	requestBodyBytes, _ := json.Marshal(reqBody)
	requestBody := string(requestBodyBytes)

	payload := map[string]interface{}{
		"username":     reqBody.Username,
		"branch_code":  reqBody.BranchCode,
		"relationship": reqBody.Relationship,
	}

	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		errorResponse := types.ApiResponse{
			Message: "Failed to marshal payload",
			Status:  fiber.StatusInternalServerError,
		}
		c.Status(fiber.StatusInternalServerError).JSON(errorResponse)
		logRequest(c, "", requestBody)
		return nil
	}

	baseURL := os.Getenv("DMS_BASE_URL")
	//baseURL := "http://192.168.1.78:8002"

	if baseURL == "" {
		errorResponse := types.ApiResponse{
			Message: "DMS_BASE_URL not set in environment",
			Status:  fiber.StatusInternalServerError,
		}
		c.Status(fiber.StatusInternalServerError).JSON(errorResponse)
		logRequest(c, "", requestBody)
		return nil
	}

	url := fmt.Sprintf("%s/user/branch-user-mapping/", baseURL)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonPayload))
	if err != nil {
		errorResponse := types.ApiResponse{
			Message: "Failed to create request",
			Status:  fiber.StatusInternalServerError,
		}
		c.Status(fiber.StatusInternalServerError).JSON(errorResponse)
		logRequest(c, "", requestBody)
		return nil
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", authHeader)
	req.Header.Set("Accept", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		errorResponse := types.ApiResponse{
			Message: "Failed to call external API",
			Status:  fiber.StatusBadGateway,
		}
		c.Status(fiber.StatusBadGateway).JSON(errorResponse)
		logRequest(c, "", requestBody)
		return nil
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		errorResponse := types.ApiResponse{
			Message: "Failed to read response",
			Status:  fiber.StatusInternalServerError,
		}
		c.Status(fiber.StatusInternalServerError).JSON(errorResponse)
		logRequest(c, "", requestBody)
		return nil
	}

	// Parse the response to include it in our standardized format
	var responseData interface{}
	if jsonErr := json.Unmarshal(body, &responseData); jsonErr == nil {
		// Check if the response indicates an authentication error
		if resp.StatusCode == 401 {
			errorResponse := types.ApiResponse{
				Message: "Authentication failed - Invalid or missing credentials",
				Status:  resp.StatusCode,
				Data:    responseData,
			}
			c.Status(resp.StatusCode).JSON(errorResponse)
			// Serialize the response properly for logging
			responseBytes, _ := json.Marshal(errorResponse)
			logRequest(c, string(responseBytes), requestBody)
			return nil
		}

		// Check for other client errors (4xx) and server errors (5xx)
		if resp.StatusCode >= 400 {
			errorResponse := types.ApiResponse{
				Message: "Branch mapping creation failed",
				Status:  resp.StatusCode,
				Data:    responseData,
			}
			c.Status(resp.StatusCode).JSON(errorResponse)
			// Serialize the response properly for logging
			responseBytes, _ := json.Marshal(errorResponse)
			logRequest(c, string(responseBytes), requestBody)
			return nil
		}

		// Success response (2xx status codes)
		successResponse := types.ApiResponse{
			Message: "Branch mapping created successfully",
			Status:  resp.StatusCode,
			Data:    responseData,
		}
		c.Status(resp.StatusCode).JSON(successResponse)
		// Serialize the response properly for logging
		responseBytes, _ := json.Marshal(successResponse)
		logRequest(c, string(responseBytes), requestBody)
		return nil
	}

	// If JSON parsing fails, handle based on status code
	if resp.StatusCode == 401 {
		errorResponse := types.ApiResponse{
			Message: "Authentication failed - Invalid or missing credentials",
			Status:  resp.StatusCode,
			Data:    string(body),
		}
		c.Status(resp.StatusCode).JSON(errorResponse)
		responseBytes, _ := json.Marshal(errorResponse)
		logRequest(c, string(responseBytes), requestBody)
		return nil
	}

	if resp.StatusCode >= 400 {
		errorResponse := types.ApiResponse{
			Message: "Branch mapping creation failed",
			Status:  resp.StatusCode,
			Data:    string(body),
		}
		c.Status(resp.StatusCode).JSON(errorResponse)
		responseBytes, _ := json.Marshal(errorResponse)
		logRequest(c, string(responseBytes), requestBody)
		return nil
	}

	// If JSON parsing fails but status is success, return the raw response
	finalResponse := types.ApiResponse{
		Message: "Branch mapping processed",
		Status:  resp.StatusCode,
		Data:    string(body),
	}
	c.Status(resp.StatusCode).JSON(finalResponse)
	// Serialize the response properly for logging
	responseBytes, _ := json.Marshal(finalResponse)
	logRequest(c, string(responseBytes), requestBody)
	return nil
}

func CreateBag(c *fiber.Ctx) error {
	// Capture the raw request body first
	rawRequestBody := string(c.Body())

	var reqBody bagType.CreateBagRequest

	authHeader := c.Get("Authorization")
	if authHeader == "" {
		errorResponse := types.ApiResponse{
			Message: "Authorization header is required",
			Status:  fiber.StatusUnauthorized,
		}
		c.Status(fiber.StatusUnauthorized).JSON(errorResponse)
		logRequest(c, "", rawRequestBody)
		return nil
	}

	if err := c.BodyParser(&reqBody); err != nil {
		errorResponse := types.ApiResponse{
			Message: "Invalid request body",
			Status:  fiber.StatusBadRequest,
		}
		c.Status(fiber.StatusBadRequest).JSON(errorResponse)
		logRequest(c, "", rawRequestBody)
		return nil
	}

	// Convert parsed reqBody to JSON string for logging (with actual values)
	requestBodyBytes, _ := json.Marshal(reqBody)
	requestBody := string(requestBodyBytes)

	// Prepare payload using data from request
	payload := map[string]interface{}{
		"bag_category":     reqBody.BagCategory,
		"bag_id":           reqBody.BagID,
		"bag_type":         reqBody.BagType,
		"dest_office_code": reqBody.DestOfficeCode,
		"rms_instruction":  reqBody.RMSInstruction,
	}

	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		errorResponse := types.ApiResponse{
			Message: "Failed to marshal payload",
			Status:  fiber.StatusInternalServerError,
		}
		c.Status(fiber.StatusInternalServerError).JSON(errorResponse)
		logRequest(c, "", requestBody)
		return nil
	}

	baseURL := os.Getenv("DMS_BASE_URL")
	//baseURL := "http://192.168.1.78:8002"
	if baseURL == "" {
		errorResponse := types.ApiResponse{
			Message: "DMS_BASE_URL not set in environment",
			Status:  fiber.StatusInternalServerError,
		}
		c.Status(fiber.StatusInternalServerError).JSON(errorResponse)
		logRequest(c, "", requestBody)
		return nil
	}

	url := fmt.Sprintf("%s/rms/bag/create/", baseURL)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonPayload))
	if err != nil {
		errorResponse := types.ApiResponse{
			Message: "Failed to create request",
			Status:  fiber.StatusInternalServerError,
		}
		c.Status(fiber.StatusInternalServerError).JSON(errorResponse)
		logRequest(c, "", requestBody)
		return nil
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", authHeader)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		errorResponse := types.ApiResponse{
			Message: "Failed to call external API",
			Status:  fiber.StatusBadGateway,
		}
		c.Status(fiber.StatusBadGateway).JSON(errorResponse)
		logRequest(c, "", requestBody)
		return nil
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		errorResponse := types.ApiResponse{
			Message: "Failed to read response",
			Status:  fiber.StatusInternalServerError,
		}
		c.Status(fiber.StatusInternalServerError).JSON(errorResponse)
		logRequest(c, "", requestBody)
		return nil
	}

	// Parse the response to include it in our standardized format
	var responseData interface{}
	if jsonErr := json.Unmarshal(body, &responseData); jsonErr == nil {
		// Check if this is a success response (2xx status codes)
		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			successResponse := types.ApiResponse{
				Message: "Bag created successfully",
				Status:  resp.StatusCode,
				Data:    responseData,
			}
			c.Status(resp.StatusCode).JSON(successResponse)
			// Serialize the response properly for logging
			responseBytes, _ := json.Marshal(successResponse)
			logRequest(c, string(responseBytes), requestBody)
			return nil
		} else {
			// For error responses, extract the message from the response data if available
			var message string = "Bag creation failed"

			// Try to extract message from response data
			if respMap, ok := responseData.(map[string]interface{}); ok {
				if detail, exists := respMap["detail"]; exists {
					if detailStr, ok := detail.(string); ok {
						message = detailStr
					}
				} else if respMessage, exists := respMap["message"]; exists {
					if msgStr, ok := respMessage.(string); ok {
						message = msgStr
					}
				}
			}

			errorResponse := types.ApiResponse{
				Message: message,
				Status:  resp.StatusCode,
				Data:    responseData,
			}
			c.Status(resp.StatusCode).JSON(errorResponse)
			// Serialize the response properly for logging
			responseBytes, _ := json.Marshal(errorResponse)
			logRequest(c, string(responseBytes), requestBody)
			return nil
		}
	}

	// If JSON parsing fails, return the raw response
	finalResponse := types.ApiResponse{
		Message: "Bag creation processed",
		Status:  resp.StatusCode,
		Data:    string(body),
	}
	c.Status(resp.StatusCode).JSON(finalResponse)
	// Serialize the response properly for logging
	responseBytes, _ := json.Marshal(finalResponse)
	logRequest(c, string(responseBytes), requestBody)
	return nil
}

func AddItemToBag(c *fiber.Ctx) error {
	// Capture the raw request body first
	rawRequestBody := string(c.Body())

	var reqBody bagType.AddItemRequest

	if err := c.BodyParser(&reqBody); err != nil {
		errorResponse := types.ApiResponse{
			Message: "Invalid request body",
			Status:  fiber.StatusBadRequest,
		}
		c.Status(fiber.StatusBadRequest).JSON(errorResponse)
		logRequest(c, "", rawRequestBody)
		return nil
	}

	// Convert parsed reqBody to JSON string for logging (with actual values)
	requestBodyBytes, _ := json.Marshal(reqBody)
	requestBody := string(requestBodyBytes)

	authHeader := c.Get("Authorization")
	if authHeader == "" {
		errorResponse := types.ApiResponse{
			Message: "Authorization header is required",
			Status:  fiber.StatusUnauthorized,
		}
		c.Status(fiber.StatusUnauthorized).JSON(errorResponse)
		logRequest(c, "", requestBody)
		return nil
	}

	db := database.DB
	var booking bookingModel.Booking
	err := db.Where("app_or_order_id = ?", reqBody.OrderId).First(&booking).Error
	if err != nil {
		errorResponse := types.ApiResponse{
			Message: fmt.Sprintf("Order ID %s not found in our records", reqBody.OrderId),
			Status:  fiber.StatusBadRequest,
		}
		c.Status(fiber.StatusBadRequest).JSON(errorResponse)
		// Serialize the error response properly for logging
		responseBytes, _ := json.Marshal(errorResponse)
		logRequest(c, string(responseBytes), requestBody)
		return nil
	}

	// Safely extract user ID from JWT claims
	var userID string
	if userClaims := c.Locals("user"); userClaims != nil {
		if claims, ok := userClaims.(map[string]interface{}); ok {
			if username, exists := claims["username"]; exists {
				if usernameStr, ok := username.(string); ok {
					// Query the database to get the actual user ID
					var authUser user.User
					if err := db.Where("username = ?", usernameStr).First(&authUser).Error; err == nil {
						// Convert user ID to string
						userID = fmt.Sprintf("%d", authUser.ID)
					}
				}
			}
		}
	}

	// Fallback to empty string if userID couldn't be extracted
	if userID == "" {
		userID = "system" // or handle this case as appropriate for your application
	}

	if booking.Status == bookingModel.BookingStatusBooked {
		// Already booked, create event for adding item to bag
		if err := booking_event.SnapshotBookingToEvent(db, &booking, "item_added_to_bag", userID); err != nil {
			// Log the error but don't fail the operation
			fmt.Printf("Failed to create booking event: %v\n", err)
		}
		// Already booked, just add article
		return callAddArticleAPI(c, authHeader, reqBody, strPtrToStr(booking.Barcode), os.Getenv("DMS_BASE_URL"), requestBody)
	}

	barcode, err := getBarcodeFromAPI(authHeader)
	if err != nil {
		errorResponse := types.ApiResponse{
			Message: fmt.Sprintf("Failed to get barcode: %v", err),
			Status:  fiber.StatusInternalServerError,
		}
		c.Status(fiber.StatusInternalServerError).JSON(errorResponse)
		logRequest(c, "", requestBody)
		return nil
	}

	bookingResponse, statusCode, err := BookingDms(authHeader, barcode, reqBody.OrderId)
	if err != nil {
		errorResponse := types.ApiResponse{
			Message: fmt.Sprintf("Failed to book article: %v", err),
			Status:  fiber.StatusInternalServerError,
		}
		c.Status(fiber.StatusInternalServerError).JSON(errorResponse)
		logRequest(c, "", requestBody)
		return nil
	}

	if statusCode < 200 || statusCode >= 300 {
		var errorResp map[string]interface{}
		if jsonErr := json.Unmarshal(bookingResponse, &errorResp); jsonErr == nil {
			errorResponse := types.ApiResponse{
				Message: "Booking failed",
				Status:  statusCode,
				Data:    errorResp,
			}
			c.Status(statusCode).JSON(errorResponse)
			responseBytes, _ := json.Marshal(errorResponse)
			logRequest(c, string(responseBytes), requestBody)
			return nil
		}
		errorResponse := types.ApiResponse{
			Message: "Booking failed",
			Status:  statusCode,
			Data:    string(bookingResponse),
		}
		c.Status(statusCode).JSON(errorResponse)
		responseBytes, _ := json.Marshal(errorResponse)
		logRequest(c, string(responseBytes), requestBody)
		return nil
	}

	// Update booking status to booked and save barcode
	booking.Status = bookingModel.BookingStatusBooked
	booking.Barcode = &barcode
	booking.CurrentBagID = &reqBody.BagID
	booking.BookingDate = time.Now()
	booking.UpdatedBy = userID

	// Use transaction to ensure both booking update and event creation succeed together
	tx := db.Begin()
	if err := tx.Save(&booking).Error; err != nil {
		tx.Rollback()
		errorResponse := types.ApiResponse{
			Message: "Failed to update booking status",
			Status:  fiber.StatusInternalServerError,
		}
		c.Status(fiber.StatusInternalServerError).JSON(errorResponse)
		logRequest(c, "", requestBody)
		return nil
	}

	// Create booking status event for status change to booked
	bookingStatusEvent := bookingModel.BookingStatusEvent{
		BookingID: booking.ID,
		Status:    booking.Status,
		CreatedBy: userID,
	}

	if err := tx.Create(&bookingStatusEvent).Error; err != nil {
		tx.Rollback()
		errorResponse := types.ApiResponse{
			Message: "Failed to create booking status event",
			Status:  fiber.StatusInternalServerError,
		}
		c.Status(fiber.StatusInternalServerError).JSON(errorResponse)
		logRequest(c, "", requestBody)
		return nil
	}

	// Create booking event for status change to booked and item added to bag
	if err := booking_event.SnapshotBookingToEvent(tx, &booking, "booking_confirmed_and_item_added_to_bag", userID); err != nil {
		tx.Rollback()
		errorResponse := types.ApiResponse{
			Message: "Failed to create booking event",
			Status:  fiber.StatusInternalServerError,
		}
		c.Status(fiber.StatusInternalServerError).JSON(errorResponse)
		logRequest(c, "", requestBody)
		return nil
	}

	// Commit the transaction
	if err := tx.Commit().Error; err != nil {
		errorResponse := types.ApiResponse{
			Message: "Failed to commit booking changes",
			Status:  fiber.StatusInternalServerError,
		}
		c.Status(fiber.StatusInternalServerError).JSON(errorResponse)
		logRequest(c, "", requestBody)
		return nil
	}

	return callAddArticleAPI(c, authHeader, reqBody, barcode, os.Getenv("DMS_BASE_URL"), requestBody)
}

// Helper function to call add-article API
func callAddArticleAPI(c *fiber.Ctx, authHeader string, reqBody bagType.AddItemRequest, barcode, baseURL string, requestBody string) error {
	if baseURL == "" {
		errorResponse := types.ApiResponse{
			Message: "Base URL not set in environment",
			Status:  fiber.StatusInternalServerError,
		}
		c.Status(fiber.StatusInternalServerError).JSON(errorResponse)
		logRequest(c, "", requestBody)
		return nil
	}
	//fmt.Println(barcode)
	payload := map[string]interface{}{
		"bag_type": reqBody.BagType,
		"bag_id":   reqBody.BagID,
		"index":    reqBody.Index,
		"item_id":  barcode,
	}
	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		errorResponse := types.ApiResponse{
			Message: "Failed to marshal payload",
			Status:  fiber.StatusInternalServerError,
		}
		c.Status(fiber.StatusInternalServerError).JSON(errorResponse)
		logRequest(c, "", requestBody)
		return nil
	}
	url := fmt.Sprintf("%s/rms/bag/add-article/", baseURL)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonPayload))
	if err != nil {
		errorResponse := types.ApiResponse{
			Message: "Failed to create request",
			Status:  fiber.StatusInternalServerError,
		}
		c.Status(fiber.StatusInternalServerError).JSON(errorResponse)
		logRequest(c, "", requestBody)
		return nil
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", authHeader)
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		errorResponse := types.ApiResponse{
			Message: "Failed to call external API",
			Status:  fiber.StatusBadGateway,
		}
		c.Status(fiber.StatusBadGateway).JSON(errorResponse)
		logRequest(c, "", requestBody)
		return nil
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		errorResponse := types.ApiResponse{
			Message: "Failed to read response",
			Status:  fiber.StatusInternalServerError,
		}
		c.Status(fiber.StatusInternalServerError).JSON(errorResponse)
		logRequest(c, "", requestBody)
		return nil
	}

	// Parse the response to include it in our standardized format
	var responseData interface{}
	if jsonErr := json.Unmarshal(body, &responseData); jsonErr == nil {
		// Check if this is a success response (2xx status codes)
		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			successResponse := types.ApiResponse{
				Message: "Item added to bag successfully",
				Status:  resp.StatusCode,
				Data:    responseData,
			}
			c.Status(resp.StatusCode).JSON(successResponse)
			// Serialize the response properly for logging
			responseBytes, _ := json.Marshal(successResponse)
			logRequest(c, string(responseBytes), requestBody)
			return nil
		} else {
			// For error responses, extract the message from the response data if available
			var message string = "Failed to add item to bag"

			// Try to extract message from response data
			//if respMap, ok := responseData.(map[string]interface{}); ok {
			//	if respMessage, exists := respMap["message"]; exists {
			//		if msgStr, ok := respMessage.(string); ok {
			//			message = msgStr
			//		}
			//	}
			//}

			errorResponse := types.ApiResponse{
				Message: message,
				Status:  resp.StatusCode,
				Data:    responseData,
			}
			c.Status(resp.StatusCode).JSON(errorResponse)
			// Serialize the response properly for logging
			responseBytes, _ := json.Marshal(errorResponse)
			logRequest(c, string(responseBytes), requestBody)
			return nil
		}
	}

	// If JSON parsing fails, return the raw response
	finalResponse := types.ApiResponse{
		Message: "Item addition processed",
		Status:  resp.StatusCode,
		Data:    string(body),
	}
	c.Status(resp.StatusCode).JSON(finalResponse)
	// Serialize the response properly for logging
	responseBytes, _ := json.Marshal(finalResponse)
	logRequest(c, string(responseBytes), requestBody)
	return nil
}

func getBarcodeFromAPI(authHeader string) (string, error) {
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

func BookingDms(authHeader, barcode, orderID string) ([]byte, int, error) {
	baseURL := os.Getenv("DMS_BASE_URL")
	url := fmt.Sprintf("%s/dms/book/article/", baseURL)

	db := database.DB
	var booking bookingModel.Booking
	// Preload related data (adjust field names as per your model)
	if err := db.
		Preload("User").
		Preload("DeliveryAddress").
		Where("app_or_order_id = ?", orderID).
		Where("status = ?", bookingModel.BookingStatusPreBooked).
		First(&booking).Error; err != nil {
		return nil, 0, fmt.Errorf("booking not found: %v", err)
	}
	//if booking blank return error
	if booking.ID == 0 {
		return nil, 0, fmt.Errorf("booking not found or already booked")
	}

	// Check if required data is loaded
	if booking.User.Uuid == "" {
		return nil, 0, fmt.Errorf("user information not found for booking")
	}

	// Initialize receiver address with safe nil checks
	receiverAddress := bagType.Address{
		AddressType:   "home", // default value
		Country:       "Bangladesh",
		District:      strPtrToStr(booking.DeliveryAddress.District),
		Division:      strPtrToStr(booking.DeliveryAddress.Division),
		PhoneNumber:   booking.Phone,
		PoliceStation: strPtrToStr(booking.DeliveryAddress.PoliceStation),
		PostOffice:    strPtrToStr(booking.DeliveryAddress.PostOffice),
		StreetAddress: booking.Address,
		UserUUID:      booking.User.Uuid,
		Username:      booking.User.Username,
		Zone:          "Zone 1",
	}
	// Safely populate address info if it exists
	if booking.DeliveryAddress != nil {
		receiverAddress.District = strPtrToStr(booking.DeliveryAddress.District)
		receiverAddress.Division = strPtrToStr(booking.DeliveryAddress.Division)
		receiverAddress.PoliceStation = strPtrToStr(booking.DeliveryAddress.PoliceStation)
		receiverAddress.PostOffice = strPtrToStr(booking.DeliveryAddress.PostOffice)
		receiverAddress.StreetAddress = strPtrToStr(booking.DeliveryAddress.StreetAddress)
	}

	payload := bagType.BookingRequest{
		FromNumber:      "",
		AdPodID:         "1",
		ArticleDesc:     "Sample Article",
		ArticlePrice:    100,
		Barcode:         barcode,
		CityPostStatus:  "N",
		DeliveryBranch:  "100000",
		EmtsBranchCode:  "",
		Height:          10,
		HndDevice:       "web",
		ImagePod:        "",
		ImageSrc:        "",
		InsurancePrice:  "0",
		IsBulkMail:      "N",
		IsCharge:        "Y",
		IsCityPost:      "N",
		IsStation:       "N",
		IsInternational: false,
		Length:          10,
		ServiceName:     "letter",
		SetAd:           "N",
		VasType:         "N",
		VpAmount:        "0",
		VpService:       "N",
		Weight:          100,
		Width:           10,
		Receiver:        receiverAddress,
		Sender: bagType.Address{
			AddressType:   "office",
			Country:       "Bangladesh",
			District:      "Dhaka",
			Division:      "Dhaka",
			PhoneNumber:   "018XXXXXXXX",
			PoliceStation: "Gulshan",
			PostOffice:    "Gulshan",
			StreetAddress: "456, Gulshan, Dhaka",
			UserUUID:      booking.User.Uuid,
			Username:      "sender-username-placeholder",
			Zone:          "Zone 2",
		},
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

func strPtrToStr(s *string) string {
	if s != nil {
		return *s
	}
	return ""
}

// Helper function Ends here

func CloseBag(c *fiber.Ctx) error {
	// Capture the raw request body first
	rawRequestBody := string(c.Body())

	var reqBody bagType.CloseBagRequest

	authHeader := c.Get("Authorization")
	if authHeader == "" {
		errorResponse := types.ApiResponse{
			Message: "Authorization header is required",
			Status:  fiber.StatusUnauthorized,
		}
		c.Status(fiber.StatusUnauthorized).JSON(errorResponse)
		logRequest(c, "", rawRequestBody)
		return nil
	}

	if err := c.BodyParser(&reqBody); err != nil {
		errorResponse := types.ApiResponse{
			Message: "Invalid request body",
			Status:  fiber.StatusBadRequest,
		}
		c.Status(fiber.StatusBadRequest).JSON(errorResponse)
		logRequest(c, "", rawRequestBody)
		return nil
	}

	// Convert parsed reqBody to JSON string for logging (with actual values)
	requestBodyBytes, _ := json.Marshal(reqBody)
	requestBody := string(requestBodyBytes)

	// Prepare payload using data from request
	payload := map[string]interface{}{
		"bag_id": reqBody.BagID,
	}

	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		errorResponse := types.ApiResponse{
			Message: "Failed to marshal payload",
			Status:  fiber.StatusInternalServerError,
		}
		c.Status(fiber.StatusInternalServerError).JSON(errorResponse)
		logRequest(c, "", requestBody)
		return nil
	}

	baseURL := os.Getenv("DMS_BASE_URL")
	if baseURL == "" {
		errorResponse := types.ApiResponse{
			Message: "DMS_BASE_URL not set in environment",
			Status:  fiber.StatusInternalServerError,
		}
		c.Status(fiber.StatusInternalServerError).JSON(errorResponse)
		logRequest(c, "", requestBody)
		return nil
	}

	url := fmt.Sprintf("%s/rms/close-bag/", baseURL)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonPayload))
	if err != nil {
		errorResponse := types.ApiResponse{
			Message: "Failed to create request",
			Status:  fiber.StatusInternalServerError,
		}
		c.Status(fiber.StatusInternalServerError).JSON(errorResponse)
		logRequest(c, "", requestBody)
		return nil
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", authHeader)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		errorResponse := types.ApiResponse{
			Message: "Failed to call external API",
			Status:  fiber.StatusBadGateway,
		}
		c.Status(fiber.StatusBadGateway).JSON(errorResponse)
		logRequest(c, "", requestBody)
		return nil
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		errorResponse := types.ApiResponse{
			Message: "Failed to read response",
			Status:  fiber.StatusInternalServerError,
		}
		c.Status(fiber.StatusInternalServerError).JSON(errorResponse)
		logRequest(c, "", requestBody)
		return nil
	}

	// Parse the response to include it in our standardized format
	var responseData interface{}
	if jsonErr := json.Unmarshal(body, &responseData); jsonErr == nil {
		// Check if this is a success response (2xx status codes)
		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			successResponse := types.ApiResponse{
				Message: "Bag closed successfully",
				Status:  resp.StatusCode,
				Data:    responseData,
			}
			c.Status(resp.StatusCode).JSON(successResponse)
			// Serialize the response properly for logging
			responseBytes, _ := json.Marshal(successResponse)
			logRequest(c, string(responseBytes), requestBody)
			return nil
		} else {
			// For error responses, extract the message from the response data if available
			var message string = "Bag closure failed"

			// Try to extract message from response data
			if respMap, ok := responseData.(map[string]interface{}); ok {
				if respMessage, exists := respMap["message"]; exists {
					if msgStr, ok := respMessage.(string); ok {
						message = msgStr
					}
				}
			}

			errorResponse := types.ApiResponse{
				Message: message,
				Status:  resp.StatusCode,
				Data:    responseData,
			}
			c.Status(resp.StatusCode).JSON(errorResponse)
			// Serialize the response properly for logging
			responseBytes, _ := json.Marshal(errorResponse)
			logRequest(c, string(responseBytes), requestBody)
			return nil
		}
	}

	// If JSON parsing fails, return the raw response
	finalResponse := types.ApiResponse{
		Message: "Bag closure processed",
		Status:  resp.StatusCode,
		Data:    string(body),
	}
	c.Status(resp.StatusCode).JSON(finalResponse)
	// Serialize the response properly for logging
	responseBytes, _ := json.Marshal(finalResponse)
	logRequest(c, string(responseBytes), requestBody)
	return nil
}

func (bc *BagController) ReceiveBag(c *fiber.Ctx) error {
	// Capture the raw request body first
	rawRequestBody := string(c.Body())
	var reqBody bagType.ReceiveBagRequest

	authHeader := c.Get("Authorization")

	if authHeader == "" {
		errorResponse := types.ApiResponse{
			Message: "Authorization header is required",
			Status:  fiber.StatusUnauthorized,
		}
		c.Status(fiber.StatusUnauthorized).JSON(errorResponse)
		logRequest(c, "", rawRequestBody)
		return nil
	}
	if err := c.BodyParser(&reqBody); err != nil {
		errorResponse := types.ApiResponse{
			Message: "Invalid request body",
			Status:  fiber.StatusBadRequest,
		}
		c.Status(fiber.StatusBadRequest).JSON(errorResponse)
		logRequest(c, "", rawRequestBody)
		return nil
	}
	requestBodyBytes, _ := json.Marshal(reqBody)
	requestBody := string(requestBodyBytes)
	payload := map[string]interface{}{
		"bag_id":           reqBody.BagID,
		"recv_instruction": reqBody.RecvInstruction,
		"line_id":          reqBody.LineID,
		"receive_items":    reqBody.ReceiveItems,
	}
	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		errorResponse := types.ApiResponse{
			Message: "Failed to marshal payload",
			Status:  fiber.StatusInternalServerError,
		}
		c.Status(fiber.StatusInternalServerError).JSON(errorResponse)
		logRequest(c, "", requestBody)
		return nil
	}
	baseURL := os.Getenv("DMS_BASE_URL")
	if baseURL == "" {
		errorResponse := types.ApiResponse{
			Message: "DMS_BASE_URL environment variable is not set",
			Status:  fiber.StatusInternalServerError,
		}
		c.Status(fiber.StatusInternalServerError).JSON(errorResponse)
		logRequest(c, "", requestBody)
		return nil
	}
	url := fmt.Sprintf("%s/rms/receive-bag/", baseURL)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonPayload))
	if err != nil {
		errorResponse := types.ApiResponse{
			Message: "Failed to create request",
			Status:  fiber.StatusInternalServerError,
		}
		c.Status(fiber.StatusInternalServerError).JSON(errorResponse)
		logRequest(c, "", requestBody)
		return nil
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", authHeader)
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		errorResponse := types.ApiResponse{
			Message: "Failed to send request",
			Status:  fiber.StatusInternalServerError,
		}
		c.Status(fiber.StatusInternalServerError).JSON(errorResponse)
		logRequest(c, "", requestBody)
		return nil
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		errorResponse := types.ApiResponse{
			Message: "Failed to read response",
			Status:  fiber.StatusInternalServerError,
		}
		c.Status(fiber.StatusInternalServerError).JSON(errorResponse)
		logRequest(c, "", requestBody)
		return nil
	}

	var responseData interface{}
	if err := json.Unmarshal(body, &responseData); err != nil {
		// If JSON decoding fails, return the raw response
		finalResponse := types.ApiResponse{
			Message: "Failed to decode response",
			Status:  resp.StatusCode,
			Data:    string(body),
		}
		c.Status(resp.StatusCode).JSON(finalResponse)
		// Serialize the response properly for logging
		responseBytes, _ := json.Marshal(finalResponse)
		logRequest(c, string(responseBytes), requestBody)
		return nil
	}

	// Check the response status code
	if resp.StatusCode == http.StatusOK {
		// Successfully received bag - now update all bookings with this bag ID
		if err := bc.updateBookingsAfterBagReceived(reqBody.BagID, c); err != nil {
			// Log the error but don't fail the main operation since bag was successfully received
			fmt.Printf("Failed to update bookings after bag received: %v\n", err)
		}

		finalResponse := types.ApiResponse{
			Message: "Bag received successfully",
			Status:  resp.StatusCode,
			Data:    responseData,
		}
		c.Status(resp.StatusCode).JSON(finalResponse)
		// Serialize the response properly for logging
		responseBytes, _ := json.Marshal(finalResponse)
		logRequest(c, string(responseBytes), requestBody)
		return nil
	} else {
		// For error responses, extract the message from the response data if available
		var message string = "Bag reception failed"

		// Try to extract message from response data
		if respMap, ok := responseData.(map[string]interface{}); ok {
			if respMessage, exists := respMap["message"]; exists {
				if msgStr, ok := respMessage.(string); ok {
					message = msgStr
				}
			}
		}

		errorResponse := types.ApiResponse{
			Message: message,
			Status:  resp.StatusCode,
			Data:    responseData,
		}
		c.Status(resp.StatusCode).JSON(errorResponse)
		// Serialize the response properly for logging
		responseBytes, _ := json.Marshal(errorResponse)
		logRequest(c, string(responseBytes), requestBody)
		return nil
	}
}

// and creates booking status events and booking snapshots for each booking
func (bc *BagController) updateBookingsAfterBagReceived(bagID string, c *fiber.Ctx) error {
	db := database.DB
	if db == nil {
		return fmt.Errorf("database connection not found")
	}

	// Get user authentication information
	claims, ok := c.Locals("user").(map[string]interface{})
	if !ok {
		return bc.sendResponseWithLog(c, fiber.StatusUnauthorized, types.ApiResponse{
			Message: "Invalid user claims",
			Status:  fiber.StatusUnauthorized,
			Data:    nil,
		})
	}

	userUUID, ok := claims["uuid"].(string)
	if !ok || userUUID == "" {
		return bc.sendResponseWithLog(c, fiber.StatusUnauthorized, types.ApiResponse{
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
		return bc.sendResponseWithLog(c, status, types.ApiResponse{
			Message: msg,
			Status:  status,
			Data:    nil,
		})
	}

	userID := uint(userInfo.ID)

	userPermission := userInfo.Permissions

	// Find all bookings with the current bag ID
	var bookings []bookingModel.Booking
	if err := db.Where("current_bag_id = ?", bagID).Find(&bookings).Error; err != nil {
		return fmt.Errorf("failed to find bookings with bag ID %s: %v", bagID, err)
	}

	if len(bookings) == 0 {
		fmt.Printf("No bookings found with bag ID: %s\n", bagID)
		return nil
	}

	// Use transaction to ensure all updates succeed together
	tx := db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Update each booking status and create events
	for _, booking := range bookings {
		// Update booking status based on user permissions
		hasPostMasterPermission := false
		for _, permission := range userPermission {
			if permission == "passport-booking.postmaster.full-permit" {
				hasPostMasterPermission = true
				break
			}
		}

		if hasPostMasterPermission {
			// User has full permission, allow status update
			booking.Status = bookingModel.BookingStatusReceivedByPostMaster
		} else {
			booking.Status = bookingModel.BookingStatusReceivedByPostman
		}
		booking.UpdatedBy = fmt.Sprintf("%d", userID)

		if err := tx.Save(&booking).Error; err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to update booking ID %d: %v", booking.ID, err)
		}

		// Create booking status event
		bookingStatusEvent := bookingModel.BookingStatusEvent{
			BookingID: booking.ID,
			Status:    booking.Status,
			CreatedBy: fmt.Sprintf("%d", userID),
		}

		if err := tx.Create(&bookingStatusEvent).Error; err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to create booking status event for booking ID %d: %v", booking.ID, err)
		}

		// Create booking snapshot event
		if err := booking_event.SnapshotBookingToEvent(tx, &booking, "bag_received_by_postman", fmt.Sprintf("%d", userID)); err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to create booking event for booking ID %d: %v", booking.ID, err)
		}
	}

	// Commit the transaction
	if err := tx.Commit().Error; err != nil {
		return fmt.Errorf("failed to commit booking updates: %v", err)
	}

	fmt.Printf("Successfully updated %d bookings for bag ID %s to item_received_by_postman status\n", len(bookings), bagID)
	return nil
}
