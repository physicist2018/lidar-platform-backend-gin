package route

import (
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	"github.com/kshmirko/lidar-platform-go/internal/delivery/http/controller"
	"github.com/kshmirko/lidar-platform-go/internal/delivery/http/middleware"
)

type RouteConfig struct {
	App            *gin.Engine
	JWTSecret      string
	UserController *controller.UserController
	AuthController *controller.AuthController
}

func NewRouteConfig(app *gin.Engine, jwtSecret string, uc *controller.UserController, ac *controller.AuthController) *RouteConfig {
	return &RouteConfig{
		App:            app,
		JWTSecret:      jwtSecret,
		UserController: uc,
		AuthController: ac,
	}
}

func (rc *RouteConfig) Setup() {
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
	}
}

func (rc *RouteConfig) SetupUserRoutes(rg *gin.RouterGroup) {
	userRoutes := rg.Group("/users")
	{
		userRoutes.GET("/", rc.UserController.GetAll)
		userRoutes.GET("/:id", rc.UserController.GetByID)

		// Admin-only routes
		admin := userRoutes.Group("")
		admin.Use(middleware.AdminOnly())
		{
			admin.POST("/", rc.UserController.Create)
			admin.PUT("/:id", rc.UserController.Update)
			admin.DELETE("/:id", rc.UserController.Delete)
		}
	}
}
