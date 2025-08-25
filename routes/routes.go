package routes

import (
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
	"os"
	"passport-booking/constants"
	"passport-booking/controllers/auth"
	"passport-booking/controllers/bag"
	"passport-booking/controllers/booking"
	"passport-booking/controllers/user"
	httpServices "passport-booking/httpServices/sso"
	"passport-booking/logger"
	"passport-booking/middleware"
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

}
