package passport_percel

import (
	"passport-booking/logger"
	"passport-booking/models/regional_passport_office"
	"passport-booking/types"
	"passport-booking/utils"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// RegionalPassportOfficeController handles regional passport office related HTTP requests
type RegionalPassportOfficeController struct {
	DB     *gorm.DB
	Logger *logger.AsyncLogger
}

// NewRegionalPassportOfficeController creates a new regional passport office controller
func NewRegionalPassportOfficeController(db *gorm.DB, asyncLogger *logger.AsyncLogger) *RegionalPassportOfficeController {
	return &RegionalPassportOfficeController{
		DB:     db,
		Logger: asyncLogger,
	}
}

// Helper function to log API requests and responses
func (rpo *RegionalPassportOfficeController) logAPIRequest(c *fiber.Ctx) {
	logEntry := utils.CreateSanitizedLogEntry(c)
	rpo.Logger.Log(logEntry)
}

// Helper function to send response and log in one call
func (rpo *RegionalPassportOfficeController) sendResponseWithLog(c *fiber.Ctx, status int, response types.ApiResponse) error {
	result := c.Status(status).JSON(response)
	rpo.logAPIRequest(c)
	return result
}

// GetRegionalPassportOffices returns a list of all regional passport offices
func (rpo *RegionalPassportOfficeController) GetRegionalPassportOffices(c *fiber.Ctx) error {
	var offices []regional_passport_office.RegionalPassportOffice

	// Get all regional passport offices
	result := rpo.DB.Find(&offices)
	if result.Error != nil {
		response := types.ApiResponse{
			Status:  fiber.StatusInternalServerError,
			Message: "Failed to retrieve regional passport offices",
			Data:    nil,
		}
		return rpo.sendResponseWithLog(c, fiber.StatusInternalServerError, response)
	}

	response := types.ApiResponse{
		Status:  fiber.StatusOK,
		Message: "Regional passport offices retrieved successfully",
		Data:    offices,
	}

	return rpo.sendResponseWithLog(c, fiber.StatusOK, response)
}
