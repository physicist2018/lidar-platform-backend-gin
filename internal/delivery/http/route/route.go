package route

import (
	"github.com/go-chi/chi/v5"

	"github.com/physicist2018/lidar-platform-go/internal/delivery/http/controller"
	"github.com/physicist2018/lidar-platform-go/internal/delivery/http/middleware"
	"github.com/physicist2018/lidar-platform-go/internal/delivery/http/swagger"
)

type RouteConfig struct {
	App                  *chi.Mux
	JWTSecret            string
	UserController       *controller.UserController
	AuthController       *controller.AuthController
	ExperimentController *controller.ExperimentController
}

func NewRouteConfig(
	app *chi.Mux,
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
	// Swagger UI (custom index.html pointing to local spec)
	rc.App.Get("/swagger/*", swagger.NewHandler().ServeHTTP)

	// Public routes
	rc.App.Post("/auth/login", rc.AuthController.Login)

	// Protected routes (authenticated)
	rc.App.Group(func(r chi.Router) {
		r.Use(middleware.AuthMiddleware(rc.JWTSecret))

		rc.SetupUserRoutes(r)
		rc.SetupExperimentRoutes(r)
	})
}

func (rc *RouteConfig) SetupUserRoutes(rg chi.Router) {
	rg.Route("/users", func(r chi.Router) {
		r.Get("/", rc.UserController.GetAll)
		r.Get("/{id}", rc.UserController.GetByID)

		// Admin-only routes
		r.Group(func(admin chi.Router) {
			admin.Use(middleware.AdminOnly)
			admin.Post("/", rc.UserController.Create)
			admin.Put("/{id}", rc.UserController.Update)
			admin.Delete("/{id}", rc.UserController.Delete)
		})
	})
}

func (rc *RouteConfig) SetupExperimentRoutes(rg chi.Router) {
	rg.Route("/experiments", func(r chi.Router) {
		r.Get("/", rc.ExperimentController.GetAll)
		r.Get("/{id}", rc.ExperimentController.GetByID)
		r.Get("/{id}/channels", rc.ExperimentController.GetChannels)

		// Admin-only routes
		r.Group(func(admin chi.Router) {
			admin.Use(middleware.AdminOnly)
			admin.Post("/", rc.ExperimentController.Create)
		})

		// Admin+Manager routes
		r.Group(func(am chi.Router) {
			am.Use(middleware.AdminOrManager)
			am.Post("/{id}/process", rc.ExperimentController.Process)
		})
	})

	// Processing runs (admin+manager)
	rg.Route("/processing", func(r chi.Router) {
		r.Use(middleware.AdminOrManager)
		r.Get("/{id}", rc.ExperimentController.GetProcessingStatus)
	})

	// Results endpoints (admin+manager)
	rg.Route("/results", func(r chi.Router) {
		r.Use(middleware.AdminOrManager)
		r.Post("/{stage}/data", rc.ExperimentController.GetStage0Data)
	})

}
