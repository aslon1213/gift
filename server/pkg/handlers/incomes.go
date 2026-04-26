package handlers

import (
	"aslon1213/gift/pkg/repository"
	"aslon1213/gift/services"
	"context"
	"time"

	"github.com/gofiber/fiber/v3"
	"go.mongodb.org/mongo-driver/v2/bson"
)

type IncomeHandler struct {
	repo     *repository.IncomeRepository
	userRepo *repository.UserRepository
}

func NewIncomeHandler(repo *repository.IncomeRepository, userRepo *repository.UserRepository) *IncomeHandler {
	return &IncomeHandler{repo: repo, userRepo: userRepo}
}

// List lists all incomes for the authenticated user.
// @Summary      List incomes
// @Description  Returns all incomes for the authenticated user
// @Tags         incomes
// @Produce      json
// @Success      200 {object} repository.Response[[]repository.Income]
// @Failure      401 {object} repository.Response[repository.Empty]
// @Failure      500 {object} repository.Response[repository.Empty]
// @Router       /api/v1/incomes [get]
func (h *IncomeHandler) List(c fiber.Ctx) error {
	userID, err := services.GetUserIDFromContext(c)
	if err != nil {
		return repository.Unauthorized(c, "unauthorized")
	}

	incomes, err := h.repo.List(context.Background())
	if err != nil {
		return repository.Internal(c, "failed to load incomes")
	}

	userIncomes := make([]*repository.Income, 0)
	for _, inc := range incomes {
		if inc.UserID == userID {
			userIncomes = append(userIncomes, inc)
		}
	}
	return repository.OK(c, "incomes loaded", userIncomes)
}

// GetByID gets a single income by ID for the authenticated user.
// @Summary      Get income by ID
// @Description  Returns the income by its ObjectID, for the authenticated user
// @Tags         incomes
// @Produce      json
// @Param        id  path     string true "Income ID"
// @Success      200 {object} repository.Response[repository.Income]
// @Failure      400 {object} repository.Response[repository.Empty]
// @Failure      401 {object} repository.Response[repository.Empty]
// @Failure      403 {object} repository.Response[repository.Empty]
// @Failure      404 {object} repository.Response[repository.Empty]
// @Router       /api/v1/incomes/{id} [get]
func (h *IncomeHandler) GetByID(c fiber.Ctx) error {
	userID, err := services.GetUserIDFromContext(c)
	if err != nil {
		return repository.Unauthorized(c, "unauthorized")
	}

	incomeID, err := bson.ObjectIDFromHex(c.Params("id"))
	if err != nil {
		return repository.BadRequest(c, "invalid income id")
	}

	income, err := h.repo.GetByID(context.Background(), incomeID)
	if err != nil {
		return repository.NotFound(c, "income not found")
	}
	if income.UserID != userID {
		return repository.Forbidden(c, "forbidden")
	}
	return repository.OK(c, "income fetched", income)
}

// Create creates a new income for the authenticated user.
// @Summary      Create income
// @Description  Creates a new income entry for the authenticated user
// @Tags         incomes
// @Accept       json
// @Produce      json
// @Param        income body     repository.Income true "Income data"
// @Success      201    {object} repository.Response[repository.Income]
// @Failure      400    {object} repository.Response[repository.Empty]
// @Failure      401    {object} repository.Response[repository.Empty]
// @Failure      500    {object} repository.Response[repository.Empty]
// @Router       /api/v1/incomes [post]
func (h *IncomeHandler) Create(c fiber.Ctx) error {
	userID, err := services.GetUserIDFromContext(c)
	if err != nil {
		return repository.Unauthorized(c, "unauthorized")
	}

	var req repository.Income
	if err := c.Bind().Body(&req); err != nil {
		return repository.BadRequest(c, "invalid request")
	}
	req.ID = bson.NewObjectID()
	req.UserID = userID
	req.CreatedAt = time.Now()
	req.UpdatedAt = time.Now()

	if req.Date.IsZero() {
		req.Date = time.Now()
	}

	if req.Amount <= 0 {
		return repository.BadRequest(c, "amount must be positive")
	}

	if err := h.repo.Create(context.Background(), &req); err != nil {
		return repository.Internal(c, "failed to create income")
	}

	user, err := h.userRepo.GetByID(context.Background(), userID)
	if err != nil {
		return repository.Internal(c, "failed to update balance, user not found")
	}
	newBalance := user.Balance + req.Amount
	if err := h.userRepo.UpdateBalance(context.Background(), userID, newBalance); err != nil {
		return repository.Internal(c, "failed to update balance")
	}

	return repository.Created(c, "income created", req)
}

// Update updates an income by ID for the authenticated user.
// @Summary      Update income
// @Description  Updates an existing income by its ObjectID for the authenticated user
// @Tags         incomes
// @Accept       json
// @Produce      json
// @Param        id     path     string             true "Income ID"
// @Param        income body     repository.Income  true "Income data"
// @Success      200    {object} repository.Response[repository.Income]
// @Failure      400    {object} repository.Response[repository.Empty]
// @Failure      401    {object} repository.Response[repository.Empty]
// @Failure      403    {object} repository.Response[repository.Empty]
// @Failure      404    {object} repository.Response[repository.Empty]
// @Failure      500    {object} repository.Response[repository.Empty]
// @Router       /api/v1/incomes/{id} [put]
func (h *IncomeHandler) Update(c fiber.Ctx) error {
	userID, err := services.GetUserIDFromContext(c)
	if err != nil {
		return repository.Unauthorized(c, "unauthorized")
	}

	incomeID, err := bson.ObjectIDFromHex(c.Params("id"))
	if err != nil {
		return repository.BadRequest(c, "invalid income id")
	}

	existing, err := h.repo.GetByID(context.Background(), incomeID)
	if err != nil {
		return repository.NotFound(c, "income not found")
	}
	if existing.UserID != userID {
		return repository.Forbidden(c, "forbidden")
	}
	var req repository.Income
	if err := c.Bind().Body(&req); err != nil {
		return repository.BadRequest(c, "invalid request")
	}

	amountBefore := existing.Amount

	existing.Amount = req.Amount
	existing.Currency = req.Currency
	existing.Source = req.Source
	existing.Description = req.Description
	existing.Date = req.Date
	existing.UpdatedAt = time.Now()

	if err := h.repo.Update(context.Background(), incomeID, existing); err != nil {
		return repository.Internal(c, "failed to update income")
	}

	if req.Amount != amountBefore {
		user, err := h.userRepo.GetByID(context.Background(), userID)
		if err != nil {
			return repository.Internal(c, "failed to update balance, user not found")
		}
		newBalance := user.Balance + (req.Amount - amountBefore)
		if err := h.userRepo.UpdateBalance(context.Background(), userID, newBalance); err != nil {
			return repository.Internal(c, "failed to update balance")
		}
	}

	return repository.OK(c, "income updated", existing)
}

// Delete deletes an income by ID for the authenticated user.
// @Summary      Delete income
// @Description  Deletes an income entry for the authenticated user by its ObjectID
// @Tags         incomes
// @Produce      json
// @Param        id  path     string true "Income ID"
// @Success      200 {object} repository.Response[repository.Empty]
// @Failure      400 {object} repository.Response[repository.Empty]
// @Failure      401 {object} repository.Response[repository.Empty]
// @Failure      403 {object} repository.Response[repository.Empty]
// @Failure      404 {object} repository.Response[repository.Empty]
// @Failure      500 {object} repository.Response[repository.Empty]
// @Router       /api/v1/incomes/{id} [delete]
func (h *IncomeHandler) Delete(c fiber.Ctx) error {
	userID, err := services.GetUserIDFromContext(c)
	if err != nil {
		return repository.Unauthorized(c, "unauthorized")
	}

	incomeID, err := bson.ObjectIDFromHex(c.Params("id"))
	if err != nil {
		return repository.BadRequest(c, "invalid income id")
	}

	income, err := h.repo.GetByID(context.Background(), incomeID)
	if err != nil {
		return repository.NotFound(c, "income not found")
	}
	if income.UserID != userID {
		return repository.Forbidden(c, "forbidden")
	}

	if err := h.repo.Delete(context.Background(), incomeID); err != nil {
		return repository.Internal(c, "failed to delete income")
	}

	user, err := h.userRepo.GetByID(context.Background(), userID)
	if err != nil {
		return repository.Internal(c, "failed to update balance, user not found")
	}
	newBalance := user.Balance - income.Amount
	if newBalance < 0 {
		newBalance = 0
	}
	if err := h.userRepo.UpdateBalance(context.Background(), userID, newBalance); err != nil {
		return repository.Internal(c, "failed to update balance")
	}

	return repository.Ack(c, "income deleted")
}
