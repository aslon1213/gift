package handlers

import (
	"aslon1213/gift/configs"
	"aslon1213/gift/pkg/repository"
	"aslon1213/gift/services"
)

type Handlers struct {
	UserHandler     *UserHandler
	GroupHandler    *GroupHandler
	SpendingHandler *SpendingHandler
	IncomeHandler   *IncomeHandler
	GoalHandler     *GoalHandler
	BudgetHandler   *BudgetHandler
	AlertHandler    *AlertHandler
	AuthHandler     *AuthHandler
}

func NewHandlers(repo *repository.Repository) *Handlers {

	config := configs.GetConfig()

	return &Handlers{
		UserHandler:     NewUserHandler(repo.Users),
		GroupHandler:    NewGroupHandler(repo.Groups),
		SpendingHandler: NewSpendingHandler(repo.Spendings, repo.Groups, repo.Users),
		IncomeHandler:   NewIncomeHandler(repo.Incomes, repo.Users),
		GoalHandler:     NewGoalHandler(repo.Goals),
		BudgetHandler:   NewBudgetHandler(repo.Budgets),
		AlertHandler:    NewAlertHandler(repo.Alerts),
		AuthHandler:     NewAuthHandler(services.NewAuthService(repo.Users, repo.RefreshTokens, config.Auth.JwtSecret, config.Auth.JwtExpiresIn)),
	}
}
