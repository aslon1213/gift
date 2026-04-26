package handlers

import (
	"aslon1213/gift/pkg/repository"
	"aslon1213/gift/services"
	"context"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v3"
	"go.mongodb.org/mongo-driver/v2/bson"
)

type BudgetHandler struct {
	repo *repository.BudgetRepository
}

func NewBudgetHandler(repo *repository.BudgetRepository) *BudgetHandler {
	return &BudgetHandler{repo: repo}
}

// List lists all budgets for the authenticated user.
// @Summary      List budgets
// @Description  Returns all budgets for the authenticated user
// @Tags         budgets
// @Produce      json
// @Success      200 {object} repository.Response[[]repository.Budget]
// @Failure      401 {object} repository.Response[repository.Empty]
// @Failure      500 {object} repository.Response[repository.Empty]
// @Router       /api/v1/budgets [get]
func (h *BudgetHandler) List(c fiber.Ctx) error {
	userID, err := services.GetUserIDFromContext(c)
	if err != nil {
		return repository.Unauthorized(c, "unauthorized")
	}

	budgets, err := h.repo.ListByUser(context.Background(), userID)
	if err != nil {
		return repository.Internal(c, "failed to load budgets")
	}
	return repository.OK(c, "budgets loaded successfully", budgets)
}

// GetByID gets a single budget by ID for the authenticated user.
// @Summary      Get budget by ID
// @Tags         budgets
// @Produce      json
// @Param        id  path     string true "Budget ID"
// @Success      200 {object} repository.Response[repository.Budget]
// @Failure      400 {object} repository.Response[repository.Empty]
// @Failure      401 {object} repository.Response[repository.Empty]
// @Failure      403 {object} repository.Response[repository.Empty]
// @Failure      404 {object} repository.Response[repository.Empty]
// @Router       /api/v1/budgets/{id} [get]
func (h *BudgetHandler) GetByID(c fiber.Ctx) error {
	userID, err := services.GetUserIDFromContext(c)
	if err != nil {
		return repository.Unauthorized(c, "unauthorized")
	}

	budgetID, err := bson.ObjectIDFromHex(c.Params("id"))
	if err != nil {
		return repository.BadRequest(c, "invalid budget id")
	}

	b, err := h.repo.GetByID(context.Background(), budgetID)
	if err != nil {
		return repository.NotFound(c, "budget not found")
	}
	if b.UserID != userID {
		return repository.Forbidden(c, "forbidden")
	}
	return repository.OK(c, "budget fetched successfully", b)
}

// Create creates a new budget for the authenticated user.
// @Summary      Create budget
// @Tags         budgets
// @Accept       json
// @Produce      json
// @Param        budget body     repository.Budget true "Budget data"
// @Success      201    {object} repository.Response[repository.Budget]
// @Failure      400    {object} repository.Response[repository.Empty]
// @Failure      401    {object} repository.Response[repository.Empty]
// @Failure      500    {object} repository.Response[repository.Empty]
// @Router       /api/v1/budgets [post]
func (h *BudgetHandler) Create(c fiber.Ctx) error {
	userID, err := services.GetUserIDFromContext(c)
	if err != nil {
		return repository.Unauthorized(c, "unauthorized")
	}

	var req repository.Budget
	if err := c.Bind().Body(&req); err != nil {
		return repository.BadRequest(c, "invalid request")
	}
	if req.Amount <= 0 {
		return repository.BadRequest(c, "amount must be positive")
	}
	if req.Limit <= 0 {
		return repository.BadRequest(c, "limit must be positive")
	}
	if req.Category == "" {
		return repository.BadRequest(c, "category is required")
	}

	req.ID = bson.NewObjectID()
	req.UserID = userID
	if req.Currency == "" {
		req.Currency = "$"
	}
	if req.Period == "" {
		req.Period = "monthly"
	}
	if req.StartDate.IsZero() {
		req.StartDate = time.Now()
	}
	req.CreatedAt = time.Now()
	req.UpdatedAt = time.Now()

	if err := h.repo.Create(context.Background(), &req); err != nil {
		return repository.Internal(c, "failed to create budget")
	}
	return repository.Created(c, "budget created successfully", req)
}

// Update updates a budget by ID for the authenticated user.
// @Summary      Update budget
// @Tags         budgets
// @Accept       json
// @Produce      json
// @Param        id     path     string             true "Budget ID"
// @Param        budget body     repository.Budget  true "Budget data"
// @Success      200    {object} repository.Response[repository.Budget]
// @Failure      400    {object} repository.Response[repository.Empty]
// @Failure      401    {object} repository.Response[repository.Empty]
// @Failure      403    {object} repository.Response[repository.Empty]
// @Failure      404    {object} repository.Response[repository.Empty]
// @Router       /api/v1/budgets/{id} [put]
func (h *BudgetHandler) Update(c fiber.Ctx) error {
	userID, err := services.GetUserIDFromContext(c)
	if err != nil {
		return repository.Unauthorized(c, "unauthorized")
	}

	budgetID, err := bson.ObjectIDFromHex(c.Params("id"))
	if err != nil {
		return repository.BadRequest(c, "invalid budget id")
	}

	existing, err := h.repo.GetByID(context.Background(), budgetID)
	if err != nil {
		return repository.NotFound(c, "budget not found")
	}
	if existing.UserID != userID {
		return repository.Forbidden(c, "forbidden")
	}

	var req repository.Budget
	if err := c.Bind().Body(&req); err != nil {
		return repository.BadRequest(c, "invalid request")
	}

	existing.Category = req.Category
	existing.Amount = req.Amount
	existing.Limit = req.Limit
	existing.Currency = req.Currency
	existing.Period = req.Period
	existing.StartDate = req.StartDate
	existing.EndDate = req.EndDate
	existing.UpdatedAt = time.Now()

	if err := h.repo.Update(context.Background(), budgetID, existing); err != nil {
		return repository.Internal(c, "failed to update budget")
	}
	return repository.OK(c, "budget updated successfully", existing)
}

// Delete deletes a budget by ID for the authenticated user.
// @Summary      Delete budget
// @Tags         budgets
// @Produce      json
// @Param        id  path     string true "Budget ID"
// @Success      200 {object} repository.Response[repository.Empty]
// @Failure      400 {object} repository.Response[repository.Empty]
// @Failure      401 {object} repository.Response[repository.Empty]
// @Failure      403 {object} repository.Response[repository.Empty]
// @Failure      404 {object} repository.Response[repository.Empty]
// @Router       /api/v1/budgets/{id} [delete]
func (h *BudgetHandler) Delete(c fiber.Ctx) error {
	userID, err := services.GetUserIDFromContext(c)
	if err != nil {
		return repository.Unauthorized(c, "unauthorized")
	}

	budgetID, err := bson.ObjectIDFromHex(c.Params("id"))
	if err != nil {
		return repository.BadRequest(c, "invalid budget id")
	}

	existing, err := h.repo.GetByID(context.Background(), budgetID)
	if err != nil {
		return repository.NotFound(c, "budget not found")
	}
	if existing.UserID != userID {
		return repository.Forbidden(c, "forbidden")
	}

	if err := h.repo.Delete(context.Background(), budgetID); err != nil {
		return repository.Internal(c, "failed to delete budget")
	}
	return repository.Ack(c, "budget deleted successfully")
}

// IncreaseAmount godoc
// @Summary      Increase the amount of a budget
// @Description  Increases the amount value for the specified budget of the authenticated user
// @Tags         budgets
// @Accept       json
// @Produce      json
// @Param        id     path  string  true  "Budget ID"
// @Param        amount query float64 false "Amount to increase" default(0)
// @Success      200 {object} repository.Response[repository.Budget]
// @Failure      400 {object} repository.Response[repository.Empty]
// @Failure      401 {object} repository.Response[repository.Empty]
// @Failure      403 {object} repository.Response[repository.Empty]
// @Failure      404 {object} repository.Response[repository.Empty]
// @Failure      500 {object} repository.Response[repository.Empty]
// @Router       /api/v1/budgets/{id}/increase [post]
func (h *BudgetHandler) IncreaseAmount(c fiber.Ctx) error {
	return h.adjustAmount(c, +1)
}

// DecreaseAmount godoc
// @Summary      Decrease the amount of a budget
// @Description  Decreases the amount value for the specified budget of the authenticated user
// @Tags         budgets
// @Accept       json
// @Produce      json
// @Param        id     path  string  true  "Budget ID"
// @Param        amount query float64 false "Amount to decrease" default(0)
// @Success      200 {object} repository.Response[repository.Budget]
// @Failure      400 {object} repository.Response[repository.Empty]
// @Failure      401 {object} repository.Response[repository.Empty]
// @Failure      403 {object} repository.Response[repository.Empty]
// @Failure      404 {object} repository.Response[repository.Empty]
// @Failure      500 {object} repository.Response[repository.Empty]
// @Router       /api/v1/budgets/{id}/decrease [post]
func (h *BudgetHandler) DecreaseAmount(c fiber.Ctx) error {
	return h.adjustAmount(c, -1)
}

func (h *BudgetHandler) adjustAmount(c fiber.Ctx, sign float64) error {
	userID, err := services.GetUserIDFromContext(c)
	if err != nil {
		return repository.Unauthorized(c, "unauthorized")
	}

	budgetID, err := bson.ObjectIDFromHex(c.Params("id"))
	if err != nil {
		return repository.BadRequest(c, "invalid budget id")
	}

	existing, err := h.repo.GetByID(context.Background(), budgetID)
	if err != nil {
		return repository.NotFound(c, "budget not found")
	}
	if existing.UserID != userID {
		return repository.Forbidden(c, "forbidden")
	}

	amountFloat, err := strconv.ParseFloat(c.Query("amount", "0"), 64)
	if err != nil {
		return repository.BadRequest(c, "invalid amount")
	}
	if amountFloat <= 0 {
		return repository.BadRequest(c, "amount must be positive")
	}

	existing.Amount += sign * amountFloat
	existing.UpdatedAt = time.Now()
	if err := h.repo.Update(context.Background(), budgetID, existing); err != nil {
		if sign > 0 {
			return repository.Internal(c, "failed to increase amount")
		}
		return repository.Internal(c, "failed to decrease amount")
	}
	if sign > 0 {
		return repository.OK(c, "amount increased successfully", existing)
	}
	return repository.OK(c, "amount decreased successfully", existing)
}
