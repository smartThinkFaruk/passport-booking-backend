package auth

import (
	"encoding/json"
	"fmt"
	"os"
	"passport-booking/database"
	httpServices "passport-booking/httpServices/sso"
	"passport-booking/logger"
	"passport-booking/models/user"
	"passport-booking/types"
	"passport-booking/utils"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

type AuthController struct {
	db             *gorm.DB
	httpService    *httpServices.SSOClient
	loggerInstance *logger.AsyncLogger
}

func NewAuthController(service *httpServices.SSOClient, db *gorm.DB, async_logger *logger.AsyncLogger) *AuthController {
	return &AuthController{httpService: service, db: db, loggerInstance: async_logger}
}

// Helper function to set secure cookies based on environment
func (h *AuthController) setSecureCookie(c *fiber.Ctx, name, value string, maxAge int) {
	isProduction := os.Getenv("APP_ENV") == "production"

	c.Cookie(&fiber.Cookie{
		Name:     name,
		Value:    value,
		HTTPOnly: false,
		Secure:   isProduction, // Only secure in production (HTTPS)
		SameSite: "Strict",
		MaxAge:   maxAge,
		Path:     "/",
	})
}

func (h *AuthController) Register(c *fiber.Ctx) error {
	// Parse the request body as JSON
	var req types.RegisterUserRequest
	if err := c.BodyParser(&req); err != nil {
		logger.Error("Error parsing request body", err)
		response := types.ErrorResponse{
			Message: fmt.Sprintf("Error parsing request body: %v", err),
			Status:  fiber.StatusBadRequest,
		}
		return c.Status(fiber.StatusBadRequest).JSON(response)
	}

	// Get the access token from Authorization header
	authHeader := c.Get("Authorization")
	if authHeader == "" {
		logger.Error("Authorization header missing", nil)
		return c.Status(fiber.StatusUnauthorized).JSON(types.ErrorResponse{
			Message: "Authorization token required",
			Status:  fiber.StatusUnauthorized,
		})
	}

	// Extract Bearer token
	tokenParts := strings.Split(authHeader, " ")
	if len(tokenParts) != 2 || tokenParts[0] != "Bearer" {
		logger.Error("Invalid authorization header format", nil)
		return c.Status(fiber.StatusUnauthorized).JSON(types.ErrorResponse{
			Message: "Invalid authorization header format",
			Status:  fiber.StatusUnauthorized,
		})
	}

	accessToken := tokenParts[1] // Extract the actual token

	// Validate request
	if validationErr := req.Validate(); validationErr != "" {
		logger.Error(validationErr, nil)
		response := types.ErrorResponse{
			Message: validationErr,
			Status:  fiber.StatusBadRequest,
		}
		return c.Status(fiber.StatusBadRequest).JSON(response)
	}
	// Make call to external API through the service
	registerResponse, err := h.httpService.RequestRegisterUser(types.RegisterUserRequest{
		PhoneNumber: req.PhoneNumber,
		Token:       req.Token,
		Password:    req.Password,
		Username:    req.Username,
		Access:      accessToken, // Pass the extracted access token
	})
	// fmt.Println("Register Response: ", registerResponse)
	if err != nil {
		logger.Error("Failed to login user", err)
		return c.Status(fiber.StatusBadGateway).JSON(types.ErrorResponse{
			Message: "Failed to login user",
			Status:  fiber.StatusBadGateway,
		})
	}

	currentTime := time.Now().Format("2006-01-02 03:04:05 PM")

	// If registration was successful, create user in local database
	if registerResponse.Status == "success" && registerResponse.User.UUID != "" {
		// Create user in local database
		newUser := user.User{
			Uuid:          registerResponse.User.UUID,
			Username:      registerResponse.User.Username,
			Phone:         registerResponse.User.PhoneNumber,
			PhoneVerified: false, // Set to false initially as SMS is sent for verification
			EmailVerified: false,
			LegalName:     "",                 // Set to empty string if null in response
			Avatar:        "",                 // Set to empty string if null in response
			Nonce:         0,                  // Default value
			Permissions:   user.StringSlice{}, // Empty permissions array
		}

		// Handle nullable fields
		if registerResponse.User.Email != nil && *registerResponse.User.Email != "" {
			newUser.Email = registerResponse.User.Email
		}
		// Email remains nil if not provided or empty
		if registerResponse.User.LegalName != nil {
			newUser.LegalName = *registerResponse.User.LegalName
		}
		if registerResponse.User.Avatar != nil {
			newUser.Avatar = *registerResponse.User.Avatar
		}

		// Create user in database
		if err := database.DB.Create(&newUser).Error; err != nil {
			logger.Error("Failed to create user in local database", err)
			// Note: We still return success since external registration succeeded
			// This is just a local database sync issue
		} else {
			logger.Success("User created in local database successfully. UUID: " + newUser.Uuid)
		}
	}

	logEntry := utils.CreateSanitizedLogEntry(c)
	h.loggerInstance.Log(logEntry)

	logger.Success("User registered in successfully." + " at " + currentTime)
	return c.Status(fiber.StatusOK).JSON(registerResponse)
	// // Start Transaction
	// tx := database.DB.Begin()

	// // Create user
	// createUser := models.User{
	// 	Uuid:          uuid.NewString(),
	// 	Username:      req.Username,
	// 	LegalName:     req.LegalName,
	// 	Phone:         req.Phone,
	// 	PhoneVerified: false,
	// 	Email:         req.Email,
	// 	EmailVerified: false,
	// 	Avatar:        "", // or req.Avatar if available
	// 	Nonce:         0,  // default value, update as needed
	// 	CreatedBy:     nil,
	// 	ApprovedBy:    nil,
	// 	Permissions:   []string{},
	// }

	// if err := tx.Create(&createUser).Error; err != nil {
	// 	tx.Rollback()
	// 	logger.Error("Failed to create user", err)
	// 	return c.Status(fiber.StatusInternalServerError).JSON(types.ApiResponse{
	// 		Message: fmt.Sprintf("Failed to create user: %v", err),
	// 		Status:  fiber.StatusInternalServerError,
	// 	})
	// }

	// tx.Commit()

}

func (h *AuthController) Login(c *fiber.Ctx) error {
	var req types.LoginDMSRequest
	if err := c.BodyParser(&req); err != nil {
		logger.Error("Error parsing request body", err)
		response := types.ApiResponse{
			Message: fmt.Errorf("Error parsing request body: %v", err).Error(),
			Status:  fiber.StatusBadRequest,
			Data:    nil,
		}
		return c.Status(fiber.StatusBadRequest).JSON(response)
	}

	// Validate request
	//if validationError := req.Validate(); validationError != "" {
	//	logger.Error(validationError, nil)
	//	response := types.ApiResponse{
	//		Message: validationError,
	//		Status:  fiber.StatusBadRequest,
	//		Data:    nil,
	//	}
	//	return c.Status(fiber.StatusBadRequest).JSON(response)
	//}

	// Make call to external API through the service
	loginResponse, err := h.httpService.RequestDMSLoginUser(types.LoginDMSRequest{
		UserName: req.UserName,
		Password: req.Password,
	})
	if err != nil {
		logger.Error("Failed to login user", err)
		return c.Status(fiber.StatusBadGateway).JSON(types.ApiResponse{
			Message: "Failed to login user",
			Status:  fiber.StatusBadGateway,
		})
	}

	currentTime := time.Now().Format("2006-01-02 03:04:05 PM")

	// Check if user exists in local database, create if not exists
	if loginResponse.Status == "success" && loginResponse.Data.UUID != "" {
		fmt.Println("Login Response Data: ")
		var existingUser user.User
		result := database.DB.Where("uuid = ?", loginResponse.Data.UUID).First(&existingUser)

		if result.Error != nil {
			// User doesn't exist, create new user
			newUser := user.User{
				Uuid:          loginResponse.Data.UUID,
				Username:      loginResponse.Data.Username,
				Phone:         loginResponse.Data.Phone,
				PhoneVerified: loginResponse.Data.PhoneVerified,
				EmailVerified: loginResponse.Data.EmailVerified,
				Avatar:        loginResponse.Data.Avatar,
				Nonce:         loginResponse.Data.Nonce,
				Permissions:   user.StringSlice(loginResponse.Data.Permissions),
			}

			// Handle nullable fields
			if loginResponse.Data.LegalName != nil {
				newUser.LegalName = *loginResponse.Data.LegalName
			}
			if loginResponse.Data.Email != nil && *loginResponse.Data.Email != "" {
				newUser.Email = loginResponse.Data.Email
			}
			// Email remains nil if not provided or empty

			// Handle CreatedBy and ApprovedBy if they exist in the response
			// For now, we'll just store the UUIDs if needed
			// You might want to implement logic to find and link existing users

			// Create user in database
			if err := database.DB.Create(&newUser).Error; err != nil {
				logger.Error("Failed to create user in local database", err)
				// Continue with login even if local database sync fails
			} else {
				logger.Success("User created in local database successfully. UUID: " + newUser.Uuid)
			}
		} else {
			// User exists, optionally update their information
			fmt.Printf("User already exists in local database. UUID: %s\n", existingUser.Uuid)
		}
	}

	fmt.Println("Login Response Data: Milon ", loginResponse)
	// Set HTTP-only secure cookies for access and refresh tokens
	if loginResponse.Access != "" {
		h.setSecureCookie(c, "access", loginResponse.Access, 8*60*60) // 8 hours
	}

	if loginResponse.Refresh != "" {
		h.setSecureCookie(c, "refresh", loginResponse.Refresh, 7*24*60*60) // 7 days
	}

	// Marshal loginResponse to JSON string for logging
	responseBodyStr := ""
	if loginResponse != nil {
		if b, err := json.Marshal(loginResponse); err == nil {
			responseBodyStr = string(b)
		}
	}

	logEntry := utils.CreateSanitizedLogEntryWithCustomBody(c, string(c.Body()), responseBodyStr)
	h.loggerInstance.Log(logEntry)

	logger.Success("User logged in successfully. uuid: " + loginResponse.Data.UUID + " at " + currentTime)
	return c.Status(fiber.StatusOK).JSON(loginResponse)
}

func (h *AuthController) GetServiceToken(c *fiber.Ctx) error {
	var req types.GetServiceTokenRequest
	if err := c.BodyParser(&req); err != nil {
		logger.Error("Error parsing request body", err)
		return c.Status(fiber.StatusBadRequest).JSON(types.ApiResponse{
			Message: "Invalid request payload",
			Status:  fiber.StatusBadRequest,
		})
	}

	if validationErr := req.Validate(); validationErr != "" {
		logger.Error(validationErr, nil)
		return c.Status(fiber.StatusBadRequest).JSON(types.ApiResponse{
			Message: validationErr,
			Status:  fiber.StatusBadRequest,
		})
	}

	// Make call to external API through the service
	redirectToken, err := h.httpService.RequestRedirectToken(httpServices.ServiceUserRequest{
		InternalIdentifier: req.InternalIdentifier,
		RedirectURL:        req.RedirectURL,
		UserType:           req.UserType,
	})
	if err != nil {
		logger.Error("Failed to retrieve redirect token", err)
		return c.Status(fiber.StatusBadGateway).JSON(types.ApiResponse{
			Message: "Failed to communicate with external service",
			Status:  fiber.StatusBadGateway,
		})
	}

	currentTime := time.Now().Format("2006-01-02 03:04:05 PM")

	// Generate your actual response
	response := types.ApiResponse{
		Message: "Got redirect token Successfully!!!",
		Status:  fiber.StatusOK,
		Data: map[string]interface{}{
			"redirect_token": redirectToken,
		},
	}

	logger.Success("User token got successfully. Redirect token: " + redirectToken + " at " + currentTime)
	return c.Status(fiber.StatusOK).JSON(response)
}

func (h *AuthController) LogOut(c *fiber.Ctx) error {
	// Get the token from the Authorization header
	tokenStr := c.Get("Authorization")
	tokenStr = strings.TrimPrefix(tokenStr, "Bearer ")

	// Clear the access and refresh cookies
	h.setSecureCookie(c, "access", "", -1)  // Expire immediately
	h.setSecureCookie(c, "refresh", "", -1) // Expire immediately

	response := types.ApiResponse{
		Message: "Logout successful",
		Status:  fiber.StatusOK,
		Data:    nil,
	}
	logger.Success("Logout successful")
	return c.Status(fiber.StatusOK).JSON(response)
}
