package passport_percel

import (
	"passport-booking/logger"
	"passport-booking/models/regional_passport_office"
	"passport-booking/types"
	regional_passport_office_types "passport-booking/types/regional_passport_office"
	"passport-booking/utils"
	"strings"

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

// StoreRegionalPassportOffice creates a new regional passport office
func (rpo *RegionalPassportOfficeController) StoreRegionalPassportOffice(c *fiber.Ctx) error {
	var request regional_passport_office_types.StoreRegionalPassportOffice

	// Parse the request body
	if err := c.BodyParser(&request); err != nil {
		response := types.ApiResponse{
			Status:  fiber.StatusBadRequest,
			Message: "Invalid request body",
			Data:    nil,
		}
		return rpo.sendResponseWithLog(c, fiber.StatusBadRequest, response)
	}

	claims, ok := c.Locals("user").(map[string]interface{})
	if !ok {
		return rpo.sendResponseWithLog(c, fiber.StatusUnauthorized, types.ApiResponse{
			Message: "Invalid user claims",
			Status:  fiber.StatusUnauthorized,
			Data:    nil,
		})
	}

	userUUID, ok := claims["uuid"].(string)
	if !ok || userUUID == "" {
		return rpo.sendResponseWithLog(c, fiber.StatusUnauthorized, types.ApiResponse{
			Message: "User UUID not found in token",
			Status:  fiber.StatusUnauthorized,
			Data:    nil,
		})
	}

	userInfo, err := utils.GetUserByUUID(userUUID)
	if err != nil {
		status := fiber.StatusInternalServerError
		msg := "Database error"
		if err.Error() == "user not found" {
			status = fiber.StatusUnauthorized
			msg = "User not found"
		}
		return rpo.sendResponseWithLog(c, status, types.ApiResponse{
			Message: msg,
			Status:  status,
			Data:    nil,
		})
	}

	// Create a new regional passport office
	office := regional_passport_office.RegionalPassportOffice{
		Code:      request.Code,
		Name:      request.Name,
		Address:   request.Address,
		Mobile:    request.Mobile,
		CreatedBy: userInfo.ID,
	}

	// Save the new regional passport office to the database
	result := rpo.DB.Create(&office)
	if result.Error != nil {
		if strings.Contains(result.Error.Error(), "Duplicate entry") || strings.Contains(result.Error.Error(), "UNIQUE constraint failed") {
			return rpo.sendResponseWithLog(c, fiber.StatusConflict, types.ApiResponse{
				Status:  fiber.StatusConflict,
				Message: "A regional passport office with this code already exists.",
				Data:    nil,
			})
		}
		response := types.ApiResponse{
			Status:  fiber.StatusInternalServerError,
			Message: "Failed to create regional passport office",
			Data:    result.Error.Error(),
		}
		return rpo.sendResponseWithLog(c, fiber.StatusInternalServerError, response)
	}

	response := types.ApiResponse{
		Status:  fiber.StatusCreated,
		Message: "Regional passport office created successfully",
		Data:    office,
	}

	return rpo.sendResponseWithLog(c, fiber.StatusCreated, response)
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
