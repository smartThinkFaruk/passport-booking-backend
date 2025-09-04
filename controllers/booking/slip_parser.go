package booking

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"passport-booking/database"
	"passport-booking/logger"
	"passport-booking/models/slip_parser"
	slipParserService "passport-booking/services/slip_parser"
	"passport-booking/types"

	"github.com/gofiber/fiber/v2"
	"google.golang.org/genai"
)

// ParsePassportSlip handles the passport slip image upload and parsing using Gemini Vision API
func (bc *BookingController) ParsePassportSlip(c *fiber.Ctx) error {
	startTime := time.Now()

	// Initialize slip parser service
	service := slipParserService.NewSlipParserService(database.DB)

	// Generate unique request ID
	requestID := service.GenerateRequestID()

	// Get uploaded file
	file, err := c.FormFile("image")
	if err != nil {
		logger.Error(fmt.Sprintf("No image file provided for request %s", requestID), err)

		return bc.sendResponseWithLog(c, fiber.StatusBadRequest, types.ApiResponse{
			Message: "No image file provided",
			Status:  fiber.StatusBadRequest,
			Data:    map[string]interface{}{"request_id": requestID},
		})
	}

	// Validate file type
	mimeType := file.Header.Get("Content-Type")
	if !isValidImageType(mimeType) {
		logger.Error(fmt.Sprintf("Invalid file type %s for request %s", mimeType, requestID),
			fmt.Errorf("invalid mime type: %s", mimeType))

		return bc.sendResponseWithLog(c, fiber.StatusBadRequest, types.ApiResponse{
			Message: "Invalid file type. Only JPEG, JPG, PNG, and WebP files are allowed",
			Status:  fiber.StatusBadRequest,
			Data:    map[string]interface{}{"request_id": requestID},
		})
	}

	// Validate file size (max 10MB)
	maxSize := int64(10 * 1024 * 1024) // 10MB
	if file.Size > maxSize {
		logger.Error(fmt.Sprintf("File size %d exceeds max %d for request %s", file.Size, maxSize, requestID),
			fmt.Errorf("file size %d exceeds max %d", file.Size, maxSize))

		return bc.sendResponseWithLog(c, fiber.StatusBadRequest, types.ApiResponse{
			Message: "File size too large. Maximum size is 10MB",
			Status:  fiber.StatusBadRequest,
			Data:    map[string]interface{}{"request_id": requestID},
		})
	}

	// Create initial database request record
	_, err = service.CreateInitialRequest(c, requestID, file.Filename, file.Size, mimeType)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to create initial request %s", requestID), err)

		return bc.sendResponseWithLog(c, fiber.StatusInternalServerError, types.ApiResponse{
			Message: "Failed to initialize request",
			Status:  fiber.StatusInternalServerError,
			Data:    map[string]interface{}{"request_id": requestID},
		})
	}

	// Open the uploaded file
	src, err := file.Open()
	if err != nil {
		processingTime := time.Since(startTime).Milliseconds()
		service.SaveFailureResultAsync(requestID, "Failed to open uploaded file", processingTime)

		logger.Error(fmt.Sprintf("Failed to open uploaded file for request %s", requestID), err)

		return bc.sendResponseWithLog(c, fiber.StatusInternalServerError, types.ApiResponse{
			Message: "Failed to process uploaded file",
			Status:  fiber.StatusInternalServerError,
			Data:    map[string]interface{}{"request_id": requestID},
		})
	}
	defer src.Close()

	// Read file content
	fileBytes, err := io.ReadAll(src)
	if err != nil {
		processingTime := time.Since(startTime).Milliseconds()
		service.SaveFailureResultAsync(requestID, "Failed to read file content", processingTime)

		logger.Error(fmt.Sprintf("Failed to read file content for request %s", requestID), err)

		return bc.sendResponseWithLog(c, fiber.StatusInternalServerError, types.ApiResponse{
			Message: "Failed to read file content",
			Status:  fiber.StatusInternalServerError,
			Data:    map[string]interface{}{"request_id": requestID},
		})
	}

	// Start async file saving
	service.SaveFileAsync(requestID, fileBytes, file.Filename, mimeType)

	// Parse the passport slip using Gemini Vision API
	result, err := bc.parseSlipWithGemini(fileBytes, mimeType)
	if err != nil {
		processingTime := time.Since(startTime).Milliseconds()
		service.SaveFailureResultAsync(requestID, fmt.Sprintf("Gemini parsing failed: %s", err.Error()), processingTime)

		logger.Error(fmt.Sprintf("Failed to parse passport slip with Gemini for request %s", requestID), err)

		return bc.sendResponseWithLog(c, fiber.StatusInternalServerError, types.ApiResponse{
			Message: "Failed to parse passport slip",
			Status:  fiber.StatusInternalServerError,
			Data: map[string]interface{}{
				"error":      err.Error(),
				"request_id": requestID,
			},
		})
	}

	// Calculate processing time
	processingTime := time.Since(startTime).Milliseconds()
	result.ProcessingTimeMs = processingTime
	result.RequestID = requestID

	// Save success result asynchronously
	service.SaveSuccessResultAsync(requestID, result)

	// Log successful parsing
	logger.Success(fmt.Sprintf("Passport slip parsed successfully in %dms with ID: %s, Request ID: %s",
		processingTime, result.AppOrOrderID, requestID))

	return bc.sendResponseWithLog(c, fiber.StatusOK, types.ApiResponse{
		Status:  fiber.StatusOK,
		Message: "Passport slip parsed successfully",
		Data:    result,
	})
}

// parseSlipWithGemini uses Gemini Vision API to extract structured data from the passport delivery slip
func (bc *BookingController) parseSlipWithGemini(imageBytes []byte, mimeType string) (*slip_parser.SlipParserResponse, error) {
	ctx := context.Background()

	// Get API key from environment
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("API_KEY not found in environment variables")
	}

	// Create Gemini client
	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey:  apiKey,
		Backend: genai.BackendGeminiAPI,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create Gemini client: %w", err)
	}

	// Create the prompt for extracting passport slip data
	prompt := `Analyze this Bangladeshi passport delivery slip image and extract the following information. Return ONLY valid JSON.

			Extract these fields from the image. If a field is missing or unclear, use an empty string.

			Required JSON format:
			{
			"app_or_order_id": string,          // Application/Order ID
			"name": string,                      // Full name from "Name:" field
			"father_name": string,               // From "Father:" field  
			"mother_name": string,               // From "Mother:" field
			"phone": string,                     // Contact phone number
			"address": string,                   // Permanent address (Combine address lines into a single readable string)
			"emergency_contact_name": string,    // Find from relation
			"emergency_contact_phone": string    // Look for it in emergency contact section
			}`

	// Generate content with image and prompt
	content := &genai.Content{
		Parts: []*genai.Part{
			&genai.Part{Text: prompt},
			&genai.Part{InlineData: &genai.Blob{
				MIMEType: mimeType,
				Data:     imageBytes,
			}},
		},
	}

	result, err := client.Models.GenerateContent(
		ctx,
		"gemini-2.5-flash-lite",
		[]*genai.Content{content},
		&genai.GenerateContentConfig{
			Temperature: genai.Ptr(float32(0.1)),
		},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to generate content with OCR: %w", err)
	}

	// Extract text from result
	if len(result.Candidates) == 0 || len(result.Candidates[0].Content.Parts) == 0 {
		return nil, fmt.Errorf("no content generated by OCR")
	}

	responseText := result.Candidates[0].Content.Parts[0].Text
	if responseText == "" {
		return nil, fmt.Errorf("empty response from OCR")
	}

	// Extract JSON from markdown code blocks if present
	jsonText := extractJSONFromMarkdown(responseText)

	// Parse JSON response
	var parsedData slip_parser.SlipParserResponse
	if err := json.Unmarshal([]byte(jsonText), &parsedData); err != nil {
		return nil, fmt.Errorf("failed to parse JSON response: %w, response: %s", err, jsonText)
	}

	return &parsedData, nil
}

// extractJSONFromMarkdown extracts JSON content from markdown code blocks
func extractJSONFromMarkdown(text string) string {
	// Remove leading and trailing whitespace
	text = strings.TrimSpace(text)

	// Check if the text starts with ```json and ends with ```
	if strings.HasPrefix(text, "```json") && strings.HasSuffix(text, "```") {
		// Remove the markdown code block markers
		text = strings.TrimPrefix(text, "```json")
		text = strings.TrimSuffix(text, "```")
		text = strings.TrimSpace(text)
		return text
	}

	// Check if the text starts with ``` and ends with ``` (generic code block)
	if strings.HasPrefix(text, "```") && strings.HasSuffix(text, "```") {
		// Find the first newline after the opening ```
		lines := strings.Split(text, "\n")
		if len(lines) > 1 {
			// Join all lines except the first and last
			jsonLines := lines[1 : len(lines)-1]
			return strings.Join(jsonLines, "\n")
		}
	}

	// If no markdown code blocks found, return the text as is
	return text
}

// isValidImageType checks if the provided content type is a valid image type
func isValidImageType(contentType string) bool {
	validTypes := map[string]bool{
		"image/jpeg": true,
		"image/jpg":  true,
		"image/png":  true,
		"image/webp": true,
	}
	return validTypes[contentType]
}
