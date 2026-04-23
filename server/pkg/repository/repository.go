package repository

import "go.mongodb.org/mongo-driver/v2/mongo"

type Repository struct {
	Users         *UserRepository
	Groups        *GroupRepository
	Spendings     *SpendingRepository
	Incomes       *IncomeRepository
	Goals         *GoalRepository
	Budgets       *BudgetRepository
	Alerts        *AlertRepository
	RefreshTokens *RefreshTokenRepository
}

func NewRepository(db *mongo.Database) *Repository {
	return &Repository{
		Users:         NewUserRepository(db),
		Groups:        NewGroupRepository(db),
		Spendings:     NewSpendingRepository(db),
		Incomes:       NewIncomeRepository(db),
		Goals:         NewGoalRepository(db),
		Budgets:       NewBudgetRepository(db),
		Alerts:        NewAlertRepository(db),
		RefreshTokens: NewRefreshTokenRepository(db),
	}
}
