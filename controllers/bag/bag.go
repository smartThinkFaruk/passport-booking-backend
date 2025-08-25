package bag

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/gofiber/fiber/v2"
	"io/ioutil"
	"net/http"
	"os"
	"passport-booking/database"
	bookingModel "passport-booking/models/booking"
	"passport-booking/models/user"
	bagType "passport-booking/types/bag"
)

func GetBranchList(c *fiber.Ctx) error {
	authHeader := c.Get("Authorization")
	if authHeader == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Authorization header is required",
		})
	}
	baseURL := os.Getenv("EKDAK_BASE_URL")
	if baseURL == "" {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "DMS_BASE_URL not set in environment"})
	}

	// Forward query params from user request
	query := c.Context().QueryArgs().String()
	url := fmt.Sprintf("%s/v1/dms-legacy-core-logs/search-dms-branch/", baseURL)
	if query != "" {
		url = fmt.Sprintf("%s?%s", url, query)
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to create request"})
	}
	req.Header.Set("Authorization", authHeader)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return c.Status(fiber.StatusBadGateway).JSON(fiber.Map{"error": "Failed to call external API"})
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to read response"})
	}

	return c.Status(resp.StatusCode).Send(body)
}

func GetOperatorList(c *fiber.Ctx) error {
	authHeader := c.Get("Authorization")
	if authHeader == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Authorization header is required",
		})
	}

	db := database.DB
	if db == nil {
		fmt.Println("DEBUG: db not found in context")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Database connection not found in context",
		})
	}

	var users []user.User
	if err := db.Find(&users).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to fetch operators"})
	}
	return c.JSON(users)
}

func CreateBranchMapping(c *fiber.Ctx) error {
	var reqBody bagType.BranchMappingRequest

	authHeader := c.Get("Authorization")
	if authHeader == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Authorization header is required",
		})
	}

	if err := c.BodyParser(&reqBody); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	payload := map[string]interface{}{
		"username":     reqBody.Username,
		"branch_code":  reqBody.BranchCode,
		"relationship": reqBody.Relationship,
	}

	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to marshal payload"})
	}

	baseURL := os.Getenv("DMS_BASE_URL")
	//baseURL := "http://192.168.1.78:8002"

	if baseURL == "" {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "DMS_BASE_URL not set in environment"})
	}

	url := fmt.Sprintf("%s/user/branch-user-mapping/", baseURL)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonPayload))
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to create request"})
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", authHeader)
	req.Header.Set("Accept", "application/json")

	fmt.Println(req)
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return c.Status(fiber.StatusBadGateway).JSON(fiber.Map{"error": "Failed to call external API"})
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to read response"})
	}

	return c.Status(resp.StatusCode).Send(body)
}

func CreateBag(c *fiber.Ctx) error {
	var reqBody bagType.CreateBagRequest

	authHeader := c.Get("Authorization")
	if authHeader == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Authorization header is required",
		})
	}

	if err := c.BodyParser(&reqBody); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

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
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to marshal payload"})
	}

	// Rest of the function remains the same...
	baseURL := os.Getenv("DMS_BASE_URL")
	//baseURL := "http://192.168.1.78:8002"
	if baseURL == "" {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "EKDAK_BASE_URL not set in environment"})
	}

	url := fmt.Sprintf("%s/rms/bag/create/", baseURL)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonPayload))
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to create request"})
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", authHeader)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return c.Status(fiber.StatusBadGateway).JSON(fiber.Map{"error": "Failed to call external API"})
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to read response"})
	}

	return c.Status(resp.StatusCode).Send(body)
}

func AddItemToBag(c *fiber.Ctx) error {
	var reqBody bagType.AddItemRequest

	if err := c.BodyParser(&reqBody); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	authHeader := c.Get("Authorization")
	if authHeader == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Authorization header is required",
		})
	}

	db := database.DB
	var booking bookingModel.Booking
	err := db.Where("app_or_order_id = ?", reqBody.OrderId).First(&booking).Error
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": fmt.Sprintf("Order ID %s not found in our records", reqBody.OrderId),
		})
	}

	if booking.Status == bookingModel.BookingStatusBooked {
		// Already booked, just add article
		return callAddArticleAPI(c, authHeader, reqBody, strPtrToStr(booking.Barcode), os.Getenv("DMS_BASE_URL"))
	}

	barcode, err := getBarcodeFromAPI(authHeader)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fmt.Sprintf("Failed to get barcode: %v", err),
		})
	}

	bookingResponse, statusCode, err := BookingDms(authHeader, barcode, reqBody.OrderId)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fmt.Sprintf("Failed to book article: %v", err),
		})
	}

	if statusCode < 200 || statusCode >= 300 {
		var errorResp map[string]interface{}
		if jsonErr := json.Unmarshal(bookingResponse, &errorResp); jsonErr == nil {
			return c.Status(statusCode).JSON(fiber.Map{
				"error":   "Booking failed",
				"details": errorResp,
			})
		}
		return c.Status(statusCode).JSON(fiber.Map{
			"error":   "Booking failed",
			"details": string(bookingResponse),
		})
	}

	// Update booking status to booked and save barcode
	booking.Status = bookingModel.BookingStatusBooked
	booking.Barcode = &barcode
	db.Save(&booking)

	return callAddArticleAPI(c, authHeader, reqBody, barcode, os.Getenv("DMS_BASE_URL"))
}

// Helper function to call add-article API
func callAddArticleAPI(c *fiber.Ctx, authHeader string, reqBody bagType.AddItemRequest, barcode, baseURL string) error {
	if baseURL == "" {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Base URL not set in environment"})
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
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to marshal payload"})
	}
	url := fmt.Sprintf("%s/rms/bag/add-article/", baseURL)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonPayload))
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to create request"})
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", authHeader)
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return c.Status(fiber.StatusBadGateway).JSON(fiber.Map{"error": "Failed to call external API"})
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to read response"})
	}
	return c.Status(resp.StatusCode).Send(body)
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

	body, err := ioutil.ReadAll(resp.Body)
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
		Preload("AddressInfo").
		Where("app_or_order_id = ?", orderID).
		First(&booking).Error; err != nil {
		return nil, 0, fmt.Errorf("booking not found: %v", err)
	}
	//bookingJSON, err := json.Marshal(booking)
	//if err != nil {
	//	return nil, 0, fmt.Errorf("failed to marshal booking details: %v", err)
	//}
	//return bookingJSON, http.StatusOK, nil

	//fmt.Println(barcode)
	// Create the booking payload with the provided structure
	payload := bagType.BookingRequest{
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
		Receiver: bagType.Address{
			AddressType:   booking.AddressInfo.AddressType,
			Country:       "Bangladesh",
			District:      strPtrToStr(booking.AddressInfo.District),
			Division:      strPtrToStr(booking.AddressInfo.Division),
			PhoneNumber:   booking.Phone,
			PoliceStation: strPtrToStr(booking.AddressInfo.PoliceStation),
			PostOffice:    strPtrToStr(booking.AddressInfo.PostOffice),
			StreetAddress: strPtrToStr(booking.AddressInfo.StreetAddress),
			UserUUID:      booking.User.Uuid,
			Username:      booking.User.Username,
			Zone:          "Zone 1",
		},
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

	body, err := ioutil.ReadAll(resp.Body)
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
