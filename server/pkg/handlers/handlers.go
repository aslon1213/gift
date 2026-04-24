package handlers

import (
	"aslon1213/gift/configs"
	"aslon1213/gift/pkg/repository"
	"aslon1213/gift/services"
	"time"

	"go.mongodb.org/mongo-driver/v2/mongo"
)

// Version is the public build tag used by the settings endpoint.
// Overridable at build time via -ldflags "-X aslon1213/gift/pkg/handlers.Version=v0.4.3".
var Version = "v0.4.2"

type Handlers struct {
	UserHandler     *UserHandler
	GroupHandler    *GroupHandler
	SpendingHandler *SpendingHandler
	IncomeHandler   *IncomeHandler
	GoalHandler     *GoalHandler
	BudgetHandler   *BudgetHandler
	AlertHandler    *AlertHandler
	SettingsHandler *SettingsHandler
	AuthHandler     *AuthHandler
}

func NewHandlers(repo *repository.Repository, db *mongo.Database, startedAt time.Time) *Handlers {

	config := configs.GetConfig()

	return &Handlers{
		UserHandler:     NewUserHandler(repo.Users),
		GroupHandler:    NewGroupHandler(repo.Groups),
		SpendingHandler: NewSpendingHandler(repo.Spendings, repo.Groups, repo.Users),
		IncomeHandler:   NewIncomeHandler(repo.Incomes, repo.Users),
		GoalHandler:     NewGoalHandler(repo.Goals),
		BudgetHandler:   NewBudgetHandler(repo.Budgets),
		AlertHandler:    NewAlertHandler(repo.Alerts),
		SettingsHandler: NewSettingsHandler(repo.Users, repo.Groups, repo.Budgets, repo.Goals, db, startedAt, Version),
		AuthHandler:     NewAuthHandler(services.NewAuthService(repo.Users, repo.RefreshTokens, config.Auth.JwtSecret, config.Auth.JwtExpiresIn), repo.Users),
	}
}
