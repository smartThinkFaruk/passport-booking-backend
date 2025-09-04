package slip_parser

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"passport-booking/logger"
	"passport-booking/models/slip_parser"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// SlipParserService handles slip parser operations
type SlipParserService struct {
	DB        *gorm.DB
	UploadDir string
}

// NewSlipParserService creates a new slip parser service
func NewSlipParserService(db *gorm.DB) *SlipParserService {
	uploadDir := "uploaded_slips"
	return &SlipParserService{
		DB:        db,
		UploadDir: uploadDir,
	}
}

// GenerateRequestID generates a 24 character unique request ID
func (s *SlipParserService) GenerateRequestID() string {
	// Generate 12 random bytes (which will become 24 hex characters)
	bytes := make([]byte, 12)
	rand.Read(bytes)

	// Convert to hex string (24 characters)
	requestID := hex.EncodeToString(bytes)

	// Add timestamp prefix to ensure uniqueness
	timestamp := time.Now().Unix()

	// Use last 6 characters of timestamp + 18 characters of random hex
	return fmt.Sprintf("%06x%s", timestamp&0xffffff, requestID[:18])
}

// CreateInitialRequest creates an initial request record in the database
func (s *SlipParserService) CreateInitialRequest(c *fiber.Ctx, requestID, originalFileName string, fileSize int64, mimeType string) (*slip_parser.SlipParserRequest, error) {
	// Get client IP address
	ipAddress := c.IP()
	if ipAddress == "" {
		ipAddress = "unknown"
	}

	// Get user agent
	userAgent := c.Get("User-Agent")

	request := &slip_parser.SlipParserRequest{
		RequestID:        requestID,
		OriginalFileName: originalFileName,
		FileSize:         fileSize,
		MimeType:         mimeType,
		Status:           "processing",
		IPAddress:        ipAddress,
		UserAgent:        &userAgent,
	}

	if err := s.DB.Create(request).Error; err != nil {
		return nil, fmt.Errorf("failed to create initial request: %w", err)
	}

	return request, nil
}

// SaveFileAsync saves the uploaded file asynchronously
func (s *SlipParserService) SaveFileAsync(requestID string, fileBytes []byte, originalFileName, mimeType string) {
	go func() {
		if err := s.saveFile(requestID, fileBytes, originalFileName, mimeType); err != nil {
			logger.Error(fmt.Sprintf("Failed to save file for request %s", requestID), err)
			// Update request with error
			s.updateRequestWithFileError(requestID, err.Error())
		}
	}()
}

// saveFile saves the file to disk and updates the database record
func (s *SlipParserService) saveFile(requestID string, fileBytes []byte, originalFileName, mimeType string) error {
	// Ensure upload directory exists
	if err := s.ensureUploadDir(); err != nil {
		return fmt.Errorf("failed to create upload directory: %w", err)
	}

	// Generate file hash
	hash := sha256.Sum256(fileBytes)
	fileHash := hex.EncodeToString(hash[:])

	// Generate unique filename
	ext := filepath.Ext(originalFileName)
	savedFileName := fmt.Sprintf("%s_%d%s", requestID, time.Now().Unix(), ext)
	filePath := filepath.Join(s.UploadDir, savedFileName)

	// Save file to disk
	if err := os.WriteFile(filePath, fileBytes, 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	// Update database record
	updates := map[string]interface{}{
		"saved_file_name": savedFileName,
		"file_hash":       fileHash,
		"file_path":       filePath,
	}

	if err := s.DB.Model(&slip_parser.SlipParserRequest{}).Where("request_id = ?", requestID).Updates(updates).Error; err != nil {
		// If database update fails, try to clean up the file
		os.Remove(filePath)
		return fmt.Errorf("failed to update request with file info: %w", err)
	}

	logger.Success(fmt.Sprintf("File saved successfully for request %s: %s", requestID, savedFileName))
	return nil
}

// SaveSuccessResultAsync saves the parsing result asynchronously
func (s *SlipParserService) SaveSuccessResultAsync(requestID string, result *slip_parser.SlipParserResponse) {
	go func() {
		if err := s.saveSuccessResult(requestID, result); err != nil {
			logger.Error(fmt.Sprintf("Failed to save success result for request %s", requestID), err)
		}
	}()
}

// saveSuccessResult saves the successful parsing result
func (s *SlipParserService) saveSuccessResult(requestID string, result *slip_parser.SlipParserResponse) error {
	var request slip_parser.SlipParserRequest
	if err := s.DB.Where("request_id = ?", requestID).First(&request).Error; err != nil {
		return fmt.Errorf("failed to find request: %w", err)
	}

	if err := request.MarkAsSuccess(s.DB, result); err != nil {
		return fmt.Errorf("failed to mark request as success: %w", err)
	}

	logger.Success(fmt.Sprintf("Parsing result saved successfully for request %s", requestID))
	return nil
}

// SaveFailureResultAsync saves the failure result asynchronously
func (s *SlipParserService) SaveFailureResultAsync(requestID string, errorMsg string, processingTime int64) {
	go func() {
		if err := s.saveFailureResult(requestID, errorMsg, processingTime); err != nil {
			logger.Error(fmt.Sprintf("Failed to save failure result for request %s", requestID), err)
		}
	}()
}

// saveFailureResult saves the failure result
func (s *SlipParserService) saveFailureResult(requestID string, errorMsg string, processingTime int64) error {
	var request slip_parser.SlipParserRequest
	if err := s.DB.Where("request_id = ?", requestID).First(&request).Error; err != nil {
		return fmt.Errorf("failed to find request: %w", err)
	}

	if err := request.MarkAsFailed(s.DB, errorMsg, processingTime); err != nil {
		return fmt.Errorf("failed to mark request as failed: %w", err)
	}

	logger.Info(fmt.Sprintf("Failure result saved for request %s: %s", requestID, errorMsg))
	return nil
}

// updateRequestWithFileError updates the request with file saving error
func (s *SlipParserService) updateRequestWithFileError(requestID string, errorMsg string) {
	errorMessage := fmt.Sprintf("File saving error: %s", errorMsg)
	updates := map[string]interface{}{
		"status":        "failed",
		"error_message": errorMessage,
	}

	if err := s.DB.Model(&slip_parser.SlipParserRequest{}).Where("request_id = ?", requestID).Updates(updates).Error; err != nil {
		logger.Error(fmt.Sprintf("Failed to update request %s with file error", requestID), err)
	}
}

// ensureUploadDir creates the upload directory if it doesn't exist
func (s *SlipParserService) ensureUploadDir() error {
	if _, err := os.Stat(s.UploadDir); os.IsNotExist(err) {
		if err := os.MkdirAll(s.UploadDir, 0755); err != nil {
			return err
		}
		logger.Info(fmt.Sprintf("Created upload directory: %s", s.UploadDir))
	}
	return nil
}

// GetRequestByID retrieves a request by ID
func (s *SlipParserService) GetRequestByID(requestID string) (*slip_parser.SlipParserRequest, error) {
	var request slip_parser.SlipParserRequest
	if err := s.DB.Where("request_id = ?", requestID).First(&request).Error; err != nil {
		return nil, err
	}
	return &request, nil
}

// GetRequestsByStatus retrieves requests by status
func (s *SlipParserService) GetRequestsByStatus(status string, limit int) ([]slip_parser.SlipParserRequest, error) {
	var requests []slip_parser.SlipParserRequest
	query := s.DB.Where("status = ?", status).Order("created_at DESC")

	if limit > 0 {
		query = query.Limit(limit)
	}

	if err := query.Find(&requests).Error; err != nil {
		return nil, err
	}

	return requests, nil
}

// CleanupOldFiles removes old files (older than specified days)
func (s *SlipParserService) CleanupOldFiles(daysOld int) error {
	cutoffDate := time.Now().AddDate(0, 0, -daysOld)

	var oldRequests []slip_parser.SlipParserRequest
	if err := s.DB.Where("created_at < ? AND file_path != ''", cutoffDate).Find(&oldRequests).Error; err != nil {
		return err
	}

	for _, request := range oldRequests {
		// Remove file from disk
		if request.FilePath != "" {
			if err := os.Remove(request.FilePath); err != nil && !os.IsNotExist(err) {
				logger.Error(fmt.Sprintf("Failed to remove old file: %s", request.FilePath), err)
			} else {
				logger.Info(fmt.Sprintf("Removed old file: %s", request.FilePath))
			}
		}

		// Clear file path in database
		if err := s.DB.Model(&request).Update("file_path", "").Error; err != nil {
			logger.Error(fmt.Sprintf("Failed to clear file path for request %s", request.RequestID), err)
		}
	}

	return nil
}
