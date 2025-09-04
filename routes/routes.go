package routes

import (
	"os"
	"passport-booking/constants"
	"passport-booking/controllers/auth"
	"passport-booking/controllers/bag"
	"passport-booking/controllers/booking"
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
	Bag api routes
	===============================================================================*/
	bagGroup := api.Group("/bag")

	bagGroup.Get("/branch-list", middleware.RequirePermissions(constants.PermSuperAdminFull), bag.GetBranchList)
	bagGroup.Get("/operator-list", middleware.RequirePermissions(constants.PermSuperAdminFull), bag.GetOperatorList)
	bagGroup.Post("/branch-mapping", middleware.RequirePermissions(constants.PermSuperAdminFull), bag.CreateBranchMapping)
	bagGroup.Post("/create", middleware.RequirePermissions(constants.PermOperatorFull), bag.CreateBag)
	bagGroup.Post("/item_add", middleware.RequirePermissions(constants.PermOperatorFull), bag.AddItemToBag)
	bagGroup.Post("/close", middleware.RequirePermissions(constants.PermOperatorFull), bag.CloseBag)

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
		constants.PermCustomerFull,
	), bookingController.Store)

	bookingGroup.Put("/create/update/:id", middleware.RequirePermissions(
		constants.PermAgentHasFull,
		constants.PermCustomerFull,
	), bookingController.StoreUpdate)

	bookingGroup.Get("/list", middleware.RequirePermissions(
		constants.PermAgentHasFull,
		constants.PermCustomerFull,
	), bookingController.Index)
	bookingGroup.Get("/details/:id", middleware.RequirePermissions(
		constants.PermAgentHasFull,
		constants.PermCustomerFull,
	), bookingController.Show)

	bookingGroup.Post("/parse-passport-slip", middleware.RequirePermissions(
		constants.PermAgentHasFull,
		constants.PermCustomerFull,
	), bookingController.ParsePassportSlip)

	/*=============================================================================
	| OTP Routes for Booking
	===============================================================================*/

	// Delivery phone management routes
	bookingGroup.Post("/delivery-phone", middleware.RequirePermissions(
		constants.PermAgentHasFull,
		constants.PermCustomerFull,
	), bookingController.UpdateDeliveryPhone)

	bookingGroup.Post("/verify-delivery-phone", middleware.RequirePermissions(
		constants.PermAgentHasFull,
		constants.PermCustomerFull,
	), bookingController.VerifyDeliveryPhone)

	bookingGroup.Post("/otp-retry-info", middleware.RequirePermissions(
		constants.PermAgentHasFull,
		constants.PermCustomerFull,
	), bookingController.GetOTPRetryInfo)

	bookingGroup.Post("/resend-otp", middleware.RequirePermissions(
		constants.PermAgentHasFull,
		constants.PermCustomerFull,
	), bookingController.ResendOTP)

}
