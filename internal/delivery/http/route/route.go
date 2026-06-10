package route

import (
	"net/http"

	"github.com/labstack/echo/v5"
	swaggerFiles "github.com/swaggo/files/v2"

	"github.com/kshmirko/lidar-platform-go/docs"
	"github.com/kshmirko/lidar-platform-go/internal/delivery/http/controller"
	"github.com/kshmirko/lidar-platform-go/internal/delivery/http/middleware"
)

type RouteConfig struct {
	App                  *echo.Echo
	JWTSecret            string
	UserController       *controller.UserController
	AuthController       *controller.AuthController
	ExperimentController *controller.ExperimentController
}

func NewRouteConfig(
	app *echo.Echo,
	jwtSecret string,
	uc *controller.UserController,
	ac *controller.AuthController,
	ec *controller.ExperimentController,
) *RouteConfig {
	return &RouteConfig{
		App:                  app,
		JWTSecret:            jwtSecret,
		UserController:       uc,
		AuthController:       ac,
		ExperimentController: ec,
	}
}

func (rc *RouteConfig) Setup() {

	// Exact routes first (must be registered before catch-all)
	rc.App.GET("/swagger/swagger.json", docs.SwaggerJSONHandler)
	rc.App.GET("/swagger/swagger.yaml", docs.SwaggerYAMLHandler)

	// Serve Swagger UI static files (catch-all — must be after exact routes)
	rc.App.GET("/swagger/*", echo.WrapHandler(http.StripPrefix("/swagger/", http.FileServer(http.FS(swaggerFiles.FS)))))

	// Public routes
	auth := rc.App.Group("/auth")
	{
		auth.POST("/login", rc.AuthController.Login)
	}

	// Protected routes (authenticated)
	protected := rc.App.Group("")
	protected.Use(middleware.AuthMiddleware(rc.JWTSecret))
	{
		rc.SetupUserRoutes(protected)
		rc.SetupExperimentRoutes(protected)
		rc.SetupTaskRoutes(protected)
	}
}

func (rc *RouteConfig) SetupUserRoutes(rg *echo.Group) {
	userRoutes := rg.Group("/users")
	{
		userRoutes.GET("", rc.UserController.GetAll)
		userRoutes.GET("/:id", rc.UserController.GetByID)

		// Admin-only routes
		admin := userRoutes.Group("")
		admin.Use(middleware.AdminOnly())
		{
			admin.POST("", rc.UserController.Create)
			admin.PUT("/:id", rc.UserController.Update)
			admin.DELETE("/:id", rc.UserController.Delete)
		}
	}
}

func (rc *RouteConfig) SetupTaskRoutes(rg *echo.Group) {
	rg.GET("/tasks/:taskID", rc.ExperimentController.GetTaskStatus)
}

func (rc *RouteConfig) SetupExperimentRoutes(rg *echo.Group) {
	expRoutes := rg.Group("/experiments")
	{
		expRoutes.GET("", rc.ExperimentController.GetAll)
		expRoutes.GET("/:id", rc.ExperimentController.GetByID)
		expRoutes.GET("/:id/channels", rc.ExperimentController.GetChannels)

		// Admin-only routes
		admin := expRoutes.Group("")
		admin.Use(middleware.AdminOnly())
		{
			admin.POST("", rc.ExperimentController.Create)
		}

		// Admin+Manager routes
		adminManager := expRoutes.Group("")
		adminManager.Use(middleware.AdminOrManager())
		{
			adminManager.POST("/:id/prepare", rc.ExperimentController.Prepare)
			adminManager.POST("/:id/glue", rc.ExperimentController.Glue)
		}

		// Prepared experiment visualization (admin+manager)
		prepRoutes := rg.Group("/prepared")
		prepRoutes.Use(middleware.AdminOrManager())
		{
			prepRoutes.GET("/:id", rc.ExperimentController.Visualize)
		}
	}
}
