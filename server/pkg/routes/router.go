package routes

import (
	"aslon1213/gift/docs"
	_ "aslon1213/gift/docs"
	"aslon1213/gift/middleware"
	"aslon1213/gift/pkg/handlers"

	"github.com/gofiber/fiber/v3"
	"github.com/yokeTH/gofiber-scalar/scalar/v3"
)

type Router struct {
	handlers *handlers.Handlers
}

func NewRouter(h *handlers.Handlers) *Router {
	return &Router{handlers: h}
}

func (r *Router) RegisterRoutes(app *fiber.App) {

	app.Get("/docs/*", scalar.New(scalar.Config{
		Path:              "/docs",
		FileContentString: docs.SwaggerInfo.ReadDoc(),
	}))

	app.Get(
		"/health",
		func(c fiber.Ctx) error {
			return c.SendString("OK")
		},
	)

	api := app.Group("/api/v1")

	// Auth routes
	auth := api.Group("/auth")
	auth.Post("/login", r.handlers.AuthHandler.Login)
	auth.Post("/register", r.handlers.AuthHandler.Register)
	auth.Post("/refresh", r.handlers.AuthHandler.RefreshToken)
	auth.Post("/logout", middleware.Protected(), r.handlers.AuthHandler.Logout)
	auth.Get("/me", middleware.Protected(), r.handlers.AuthHandler.GetUserInfo)

	// add protected middleware to all routes

	// User routes

	users := api.Group("/users", middleware.Protected())
	users.Get("/", r.handlers.UserHandler.Query)
	users.Get("/:id", r.handlers.UserHandler.GetByID)
	users.Put("/:id", r.handlers.UserHandler.Update)
	users.Delete("/:id", r.handlers.UserHandler.Delete)

	// Groups
	groups := api.Group("/groups", middleware.Protected())
	groups.Get("/", r.handlers.GroupHandler.Query)
	groups.Post("/", r.handlers.GroupHandler.Create)
	groups.Get("/:id", r.handlers.GroupHandler.GetByID)
	groups.Put("/:id", r.handlers.GroupHandler.Update)
	groups.Delete("/:id", r.handlers.GroupHandler.Delete)
	groups.Post("/:id/invite", r.handlers.GroupHandler.InviteMember)
	groups.Post("/:id/remove", r.handlers.GroupHandler.RemoveMember)

	// Spending Routes
	spendings := api.Group("/spendings", middleware.Protected())
	spendings.Get("/", r.handlers.SpendingHandler.Query)
	spendings.Post("/", r.handlers.SpendingHandler.Create)
	spendings.Get("/:id", r.handlers.SpendingHandler.GetByID)
	spendings.Put("/:id", r.handlers.SpendingHandler.Update)
	spendings.Delete("/:id", r.handlers.SpendingHandler.Delete)

	// Income Routes
	incomes := api.Group("/incomes", middleware.Protected())
	incomes.Get("/", r.handlers.IncomeHandler.List)
	incomes.Post("/", r.handlers.IncomeHandler.Create)
	incomes.Get("/:id", r.handlers.IncomeHandler.GetByID)
	incomes.Put("/:id", r.handlers.IncomeHandler.Update)
	incomes.Delete("/:id", r.handlers.IncomeHandler.Delete)

	// goals
	goals := api.Group("/goals", middleware.Protected())
	goals.Get("/", r.handlers.GoalHandler.List)
	goals.Post("/", r.handlers.GoalHandler.Create)
	goals.Get("/:id", r.handlers.GoalHandler.GetByID)
	goals.Put("/:id", r.handlers.GoalHandler.Update)
	goals.Delete("/:id", r.handlers.GoalHandler.Delete)
	goals.Post("/:id/contribute", r.handlers.GoalHandler.Contribute)

	// settings
	settings := api.Group("/settings", middleware.Protected())
	settings.Get("/", r.handlers.SettingsHandler.Get)
	settings.Post("/export_data", r.handlers.SettingsHandler.ExportData)

	// budgets
	budgets := api.Group("/budgets", middleware.Protected())
	budgets.Get("/", r.handlers.BudgetHandler.List)
	budgets.Post("/", r.handlers.BudgetHandler.Create)
	budgets.Get("/:id", r.handlers.BudgetHandler.GetByID)
	budgets.Put("/:id", r.handlers.BudgetHandler.Update)
	budgets.Delete("/:id", r.handlers.BudgetHandler.Delete)

	// alerts
	alerts := api.Group("/alerts", middleware.Protected())
	alerts.Get("/", r.handlers.AlertHandler.List)
	alerts.Post("/", r.handlers.AlertHandler.Create)
	alerts.Get("/:id", r.handlers.AlertHandler.GetByID)
	alerts.Put("/:id", r.handlers.AlertHandler.Update)
	alerts.Delete("/:id", r.handlers.AlertHandler.Delete)

}
