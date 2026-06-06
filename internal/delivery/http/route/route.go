package route

import (
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	"github.com/kshmirko/lidar-platform-go/docs"
	"github.com/kshmirko/lidar-platform-go/internal/delivery/http/controller"
	"github.com/kshmirko/lidar-platform-go/internal/delivery/http/middleware"
)

type RouteConfig struct {
	App                  *gin.Engine
	JWTSecret            string
	UserController       *controller.UserController
	AuthController       *controller.AuthController
	ExperimentController *controller.ExperimentController
}

func NewRouteConfig(
	app *gin.Engine,
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
	docs.SwaggerInfo.Host = "lidarbackup.dvo.ru:18080"

	rc.App.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

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

func (rc *RouteConfig) SetupUserRoutes(rg *gin.RouterGroup) {
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

func (rc *RouteConfig) SetupTaskRoutes(rg *gin.RouterGroup) {
	rg.GET("/tasks/:taskID", rc.ExperimentController.GetTaskStatus)
}

func (rc *RouteConfig) SetupExperimentRoutes(rg *gin.RouterGroup) {
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
