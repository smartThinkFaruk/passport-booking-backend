package routes

import (
	// "passport-booking/constants"
	"os"
	"passport-booking/controllers/auth"
	"passport-booking/controllers/user"
	httpServices "passport-booking/httpServices/sso"
	"passport-booking/logger"
	"passport-booking/middleware"

	//"passport-booking/middleware"
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

func SetupRoutes(app *fiber.App, db *gorm.DB) {
	ssoClient := httpServices.NewClient(os.Getenv("SSO_BASE_URL"))
	asyncLogger := logger.NewAsyncLogger(db)
	authController := auth.NewAuthController(ssoClient, db, asyncLogger)

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

	// bookingGroup := api.Group("/booking")

	// bookingGroup.Post("/create", middleware.RequirePermissions(
	// 	constants.PermAgentHasFull,
	// ), bookingController.Store)

}
