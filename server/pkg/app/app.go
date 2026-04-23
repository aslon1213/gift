package app

import (
	"aslon1213/gift/pkg/handlers"
	"aslon1213/gift/pkg/repository"
	"aslon1213/gift/pkg/routes"
	"aslon1213/gift/platform"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/cors"
	"github.com/gofiber/fiber/v3/middleware/logger"
)

type App struct {
	Router *routes.Router
	fiber  *fiber.App
}

func (a *App) SetupRoutes() {
	a.Router.RegisterRoutes(a.fiber)
}

func (a *App) SetupFiber() {
	a.fiber = fiber.New()

	// add logger middleware
	a.fiber.Use(logger.New())

	// Permissive CORS so web clients can configure an arbitrary server URL.
	a.fiber.Use(cors.New(cors.Config{
		AllowOriginsFunc: func(string) bool { return true },
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
	}))
}

func NewApp() *App {
	db := platform.NewDB()
	app := &App{
		Router: routes.NewRouter(handlers.NewHandlers(repository.NewRepository(db.Database("gift")))),
	}
	app.SetupFiber()
	return app
}

func (a *App) Start() error {
	a.SetupRoutes()
	return a.fiber.Listen(":3000")
}
