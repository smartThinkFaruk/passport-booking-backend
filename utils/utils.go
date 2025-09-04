package utils

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"passport-booking/database"
	"passport-booking/models/user"
	"passport-booking/types"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"github.com/jinzhu/now"
	"gorm.io/gorm"
)

// Mutex for safe concurrent access
var mu sync.Mutex

// Global variable to store decoded token data
var GlobalTokenData map[string]interface{}

// BarcodeRequest represents the request payload for barcode generation
type BarcodeRequest struct {
	ServiceType string `json:"service_type"`
}

// BarcodeResponse represents the response from the barcode generation API
type BarcodeResponse struct {
	Status  string `json:"status"`
	Barcode string `json:"barcode"`
	Message string `json:"message"`
}

// SetTokenData sets the global token data
func SetTokenData(data map[string]interface{}) {
	mu.Lock()
	defer mu.Unlock()
	GlobalTokenData = data
}

// GetTokenData gets the global token data
func GetTokenData() map[string]interface{} {
	mu.Lock()
	defer mu.Unlock()
	return GlobalTokenData
}

// Function to calculate age in Years, Months, and Days
func CalculateAge(dob time.Time) (int, int, int) {
	currentTime := time.Now()

	// Extract year, month, and day
	years := currentTime.Year() - dob.Year()
	months := int(currentTime.Month()) - int(dob.Month())
	days := currentTime.Day() - dob.Day()

	// Adjust for negative months (if birthday hasn't occurred this year)
	if months < 0 {
		years--
		months += 12
	}

	// Adjust for negative days (if birthday day hasn't occurred this month)
	if days < 0 {
		previousMonth := now.With(currentTime).BeginningOfMonth().AddDate(0, 0, -1) // Get last day of the previous month
		days += previousMonth.Day()
		months--
	}

	return years, months, days
}

func ExtractUUIDFromToken(c *fiber.Ctx) (string, error) {
	authHeader := c.Get("Authorization")
	if authHeader == "" {
		return "", fmt.Errorf("authorization header missing")
	}

	// Split "Bearer <token>"
	tokenParts := strings.Split(authHeader, " ")
	if len(tokenParts) != 2 || tokenParts[0] != "Bearer" {
		return "", fmt.Errorf("invalid token format")
	}

	tokenString := tokenParts[1]

	// Parse the token
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// Validate the signing method (adjust as per your JWT configuration)
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method")
		}
		// Replace with your secret key
		return []byte("your_secret_key"), nil
	})

	if err != nil {
		return "", err
	}

	// Extract claims
	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		uid, ok := claims["Uid"].(string)
		if !ok {
			return "", fmt.Errorf("uuid not found in token")
		}
		return uid, nil
	}

	return "", fmt.Errorf("invalid token")
}

// GetUserByUUID retrieves a user by their UUID from the database
func GetUserByUUID(uuid string) (*user.User, error) {
	if uuid == "" {
		return nil, errors.New("UUID cannot be empty")
	}

	var userModel user.User
	if err := database.DB.Where("uuid = ?", uuid).First(&userModel).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("user not found")
		}
		return nil, fmt.Errorf("database error: %w", err)
	}

	return &userModel, nil
}

func GenerateBarcode(serviceName, authHeader string) (string, error) {
	serviceName = strings.TrimSpace(serviceName)
	if serviceName == "" {
		return "", fmt.Errorf("serviceName is empty")
	}

	base := strings.TrimRight(os.Getenv("DMS_BASE_URL"), "/")
	if base == "" {
		return "", fmt.Errorf("DMS_BASE_URL is not set")
	}
	url := base + "/dms/create-new-barcode/"

	// payload
	reqBody, err := json.Marshal(BarcodeRequest{ServiceType: serviceName})
	if err != nil {
		return "", fmt.Errorf("marshal payload: %w", err)
	}

	// normalize Authorization (support either raw token or full "Bearer <token>")
	auth := strings.TrimSpace(authHeader)
	if auth != "" && !strings.HasPrefix(strings.ToLower(auth), "bearer ") {
		auth = "Bearer " + auth
	}

	client := &http.Client{Timeout: 30 * time.Second}
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(reqBody))
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if auth != "" {
		req.Header.Set("Authorization", auth)
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read body: %w", err)
	}

	// Accept ANY 2xx status (200, 201, etc.)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("api error: status=%d body=%s", resp.StatusCode, string(body))
	}

	var bResp BarcodeResponse
	if err := json.Unmarshal(body, &bResp); err != nil {
		return "", fmt.Errorf("unmarshal response: %w", err)
	}

	if strings.ToLower(bResp.Status) != "success" {
		// API said not success even though 2xx
		return "", fmt.Errorf("barcode generation failed: %s", bResp.Message)
	}
	if strings.TrimSpace(bResp.Barcode) == "" {
		return "", fmt.Errorf("empty barcode in successful response")
	}

	return bResp.Barcode, nil
}

func GetServiceCost(serviceName string, weight int, additionalService string, trackingNumber string, isInternational bool, countryName string, authHeader string) (float64, error) {
	base := strings.TrimRight(os.Getenv("DMS_BASE_URL"), "/")
	if base == "" {
		return 0, fmt.Errorf("DMS_BASE_URL is not set")
	}
	url := base + "/dms/get_calculate_service_cost/"

	payload := map[string]interface{}{
		"service_name":       serviceName,
		"weight_gm":          weight,
		"additional_service": additionalService,
		"tracking_number":    trackingNumber,
		"is_international":   isInternational,
		"country_name":       countryName,
	}

	reqBody, err := json.Marshal(payload)
	if err != nil {
		return 0, fmt.Errorf("marshal payload: %w", err)
	}

	auth := strings.TrimSpace(authHeader)
	if auth != "" && !strings.HasPrefix(strings.ToLower(auth), "bearer ") {
		auth = "Bearer " + auth
	}

	client := &http.Client{Timeout: 30 * time.Second}
	req, err := http.NewRequest(http.MethodGet, url, bytes.NewBuffer(reqBody))
	if err != nil {
		return 0, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if auth != "" {
		req.Header.Set("Authorization", auth)
	}

	resp, err := client.Do(req)
	if err != nil {
		return 0, fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, fmt.Errorf("read body: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return 0, fmt.Errorf("api error: status=%d body=%s", resp.StatusCode, string(body))
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return 0, fmt.Errorf("unmarshal response: %w", err)
	}

	cost, ok := result["total_cost"].(float64)
	if !ok {
		return 0, fmt.Errorf("invalid cost in response")
	}

	return cost, nil
}

// ValidatePhoneNumber validates phone number using the specified regex pattern
// Pattern: /^(?:\+88)?01[0-9]{9}$/
// Allows: 01xxxxxxxxx or +8801xxxxxxxxx (where x is any digit 0-9)
func ValidatePhoneNumber(phone string) bool {
	// Remove any whitespace
	phone = strings.TrimSpace(phone)

	// Define the regex pattern
	pattern := `^(?:\+88)?01[0-9]{9}$`

	// Compile the regular expression
	re := regexp.MustCompile(pattern)

	// Check if the phone matches the pattern
	return re.MatchString(phone)
}

// sanitizeRequestBody sanitizes request body for file uploads and large content
func sanitizeRequestBody(c *fiber.Ctx) string {
	// Check if this is a multipart form (file upload)
	contentType := c.Get("Content-Type")
	if strings.Contains(contentType, "multipart/form-data") {
		// For multipart requests, create a sanitized representation
		formData := make(map[string]interface{})

		// Get form values (non-file fields)
		if form, err := c.MultipartForm(); err == nil {
			// Add text fields
			for key, values := range form.Value {
				if len(values) > 0 {
					formData[key] = values[0] // Take first value
				}
			}

			// Add file field information without content
			for key, files := range form.File {
				fileInfo := make([]map[string]interface{}, len(files))
				for i, file := range files {
					fileInfo[i] = map[string]interface{}{
						"filename": file.Filename,
						"size":     file.Size,
						"content":  "[FILE_CONTENT_REMOVED]",
					}
				}
				formData[key] = fileInfo
			}
		}

		// Convert to JSON string
		if jsonBytes, err := json.Marshal(formData); err == nil {
			return string(jsonBytes)
		}
		return "[MULTIPART_FORM_DATA]"
	}

	// For regular requests, return the body but check for base64 encoded content
	body := string(c.Body())
	if len(body) > 1000 && (strings.Contains(body, "data:image/") ||
		strings.Contains(body, "base64") ||
		isLikelyBase64(body)) {
		return "[LARGE_REQUEST_BODY_WITH_POSSIBLE_FILE_CONTENT]"
	}

	return body
}

// isLikelyBase64 detects if content looks like base64
func isLikelyBase64(content string) bool {
	// Simple heuristic: if more than 50% of content is base64 characters and it's long
	if len(content) < 100 {
		return false
	}

	base64Chars := 0
	for _, char := range content {
		if (char >= 'A' && char <= 'Z') ||
			(char >= 'a' && char <= 'z') ||
			(char >= '0' && char <= '9') ||
			char == '+' || char == '/' || char == '=' {
			base64Chars++
		}
	}

	return float64(base64Chars)/float64(len(content)) > 0.8
}

// CreateSanitizedLogEntry creates a deep copied and sanitized log entry for logging
// This function handles file uploads, large content, and creates safe copies of all data
func CreateSanitizedLogEntry(c *fiber.Ctx) types.LogEntry {
	// Create deep copies of all data to prevent memory reference issues
	method := string([]byte(c.Method()))
	url := string([]byte(c.OriginalURL()))
	requestBody := sanitizeRequestBody(c) // Use sanitized request body
	responseBody := string(append([]byte(nil), c.Response().Body()...))

	// Deep copy headers
	requestHeaders := make([]byte, len(c.Request().Header.Header()))
	copy(requestHeaders, c.Request().Header.Header())

	responseHeaders := make([]byte, len(c.Response().Header.Header()))
	copy(responseHeaders, c.Response().Header.Header())

	return types.LogEntry{
		Method:          method,
		URL:             url,
		RequestBody:     requestBody,
		ResponseBody:    responseBody,
		RequestHeaders:  string(requestHeaders),
		ResponseHeaders: string(responseHeaders),
		StatusCode:      c.Response().StatusCode(),
		CreatedAt:       time.Now(),
	}
}

// CreateSanitizedLogEntryWithCustomBody creates a sanitized log entry with custom request and response bodies
// Useful for cases where you want to provide pre-processed body content
func CreateSanitizedLogEntryWithCustomBody(c *fiber.Ctx, requestBody, responseBody string) types.LogEntry {
	// Create deep copies of all data to prevent memory reference issues
	method := string([]byte(c.Method()))
	url := string([]byte(c.OriginalURL()))
	requestBodyCopy := string(append([]byte(nil), []byte(requestBody)...))
	responseBodyCopy := string(append([]byte(nil), []byte(responseBody)...))

	// Deep copy headers
	requestHeaders := make([]byte, len(c.Request().Header.Header()))
	copy(requestHeaders, c.Request().Header.Header())

	responseHeaders := make([]byte, len(c.Response().Header.Header()))
	copy(responseHeaders, c.Response().Header.Header())

	return types.LogEntry{
		Method:          method,
		URL:             url,
		RequestBody:     requestBodyCopy,
		ResponseBody:    responseBodyCopy,
		RequestHeaders:  string(requestHeaders),
		ResponseHeaders: string(responseHeaders),
		StatusCode:      c.Response().StatusCode(),
		CreatedAt:       time.Now(),
	}
}
