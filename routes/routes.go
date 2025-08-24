package routes

import (
	"os"
	"passport-booking/constants"
	"passport-booking/controllers/auth"
	"passport-booking/controllers/booking"
	"passport-booking/controllers/otp"
	"passport-booking/controllers/user"
	httpServices "passport-booking/httpServices/sso"
	"passport-booking/logger"
	"passport-booking/middleware"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

func SetupRoutes(app *fiber.App, db *gorm.DB) {
	ssoClient := httpServices.NewClient(os.Getenv("SSO_BASE_URL"))
	asyncLogger := logger.NewAsyncLogger(db)
	authController := auth.NewAuthController(ssoClient, db, asyncLogger)
	bookingController := booking.NewBookingController(db, asyncLogger)
	otpController := otp.NewOTPController(db, asyncLogger)

	// Start the async logger processing goroutine
	go asyncLogger.ProcessLog()

	// Index route
	app.Get("/", func(c *fiber.Ctx) error {
		return c.Render("index", fiber.Map{
			"title": "Home",
		})
	})

	/*=============================================================================
	| Public Routes
	===============================================================================*/
	api := app.Group("/api")
	api.Post("/get-service-token", authController.GetServiceToken)
	api.Post("/login", authController.Login)
	api.Post("/register", authController.Register)

	/*=============================================================================
	| Protected Routes
	===============================================================================*/
	auth := api.Group("/auth").Use(middleware.RequireAnyPermission())
	auth.Post("/register", authController.Register)
	auth.Get("/profile", user.GetUserInfo)
	auth.Post("/logout", authController.LogOut)

	/*=============================================================================
	| Booking Routes
	===============================================================================*/
	bookingGroup := api.Group("/booking")

	bookingGroup.Post("/create", middleware.RequirePermissions(
		constants.PermAgentHasFull,
	), bookingController.Store)

	// Delivery phone management routes
	bookingGroup.Post("/delivery-phone", middleware.RequirePermissions(
		constants.PermAgentHasFull,
	), bookingController.UpdateDeliveryPhone)

	bookingGroup.Post("/verify-delivery-phone", middleware.RequirePermissions(
		constants.PermAgentHasFull,
	), bookingController.VerifyDeliveryPhone)

	bookingGroup.Post("/otp-retry-info", middleware.RequirePermissions(
		constants.PermAgentHasFull,
	), bookingController.GetOTPRetryInfo)

	bookingGroup.Post("/resend-otp", middleware.RequirePermissions(
		constants.PermAgentHasFull,
	), bookingController.ResendOTP)

	// Booking-specific OTP routes (send OTP without updating phone)
	bookingGroup.Post("/send-otp", middleware.RequirePermissions(
		constants.PermAgentHasFull,
	), otpController.SendOTPForBooking)

	bookingGroup.Post("/verify-otp", middleware.RequirePermissions(
		constants.PermAgentHasFull,
	), otpController.VerifyOTPForBooking)

	bookingGroup.Post("/otp-status", middleware.RequirePermissions(
		constants.PermAgentHasFull,
	), otpController.GetBookingOTPStatus)
	/*=============================================================================
	| OTP Routes
	===============================================================================*/
	otpGroup := api.Group("/otp")

	// Public OTP routes (no authentication required for sending OTP)
	otpGroup.Post("/send", otpController.SendOTP)
	otpGroup.Post("/verify", otpController.VerifyOTP)

}
