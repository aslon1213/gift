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
// @Summary List budgets
// @Description Returns all budgets for the authenticated user
// @Tags budgets
// @Produce json
// @Success 200 {array} repository.Budget
// @Failure 401 {object} repository.Response
// @Failure 500 {object} repository.Response
// @Router /api/v1/budgets [get]
func (h *BudgetHandler) List(c fiber.Ctx) error {
	userID, err := services.GetUserIDFromContext(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(repository.Response{
			Status:  "error",
			Message: "unauthorized",
			Data:    nil,
		})
	}

	budgets, err := h.repo.ListByUser(context.Background(), userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(repository.Response{
			Status:  "error",
			Message: "failed to load budgets",
			Data:    nil,
		})
	}
	return c.Status(fiber.StatusOK).JSON(repository.Response{
		Status:  "success",
		Message: "budgets loaded successfully",
		Data:    budgets,
	})
}

// GetByID gets a single budget by ID for the authenticated user.
// @Summary Get budget by ID
// @Tags budgets
// @Produce json
// @Param id path string true "Budget ID"
// @Success 200 {object} repository.Budget
// @Failure 400 {object} repository.Response
// @Failure 401 {object} repository.Response
// @Failure 403 {object} repository.Response
// @Failure 404 {object} repository.Response
// @Router /api/v1/budgets/{id} [get]
func (h *BudgetHandler) GetByID(c fiber.Ctx) error {
	userID, err := services.GetUserIDFromContext(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(repository.Response{
			Status:  "error",
			Message: "unauthorized",
			Data:    nil,
		})
	}

	budgetID, err := bson.ObjectIDFromHex(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(repository.Response{
			Status:  "error",
			Message: "invalid budget id",
			Data:    nil,
		})
	}

	b, err := h.repo.GetByID(context.Background(), budgetID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(repository.Response{
			Status:  "error",
			Message: "budget not found",
			Data:    nil,
		})
	}
	if b.UserID != userID {
		return c.Status(fiber.StatusForbidden).JSON(repository.Response{
			Status:  "error",
			Message: "forbidden",
			Data:    nil,
		})
	}
	return c.Status(fiber.StatusOK).JSON(repository.Response{
		Status:  "success",
		Message: "budget fetched successfully",
		Data:    b,
	})
}

// Create creates a new budget for the authenticated user.
// @Summary Create budget
// @Tags budgets
// @Accept json
// @Produce json
// @Param budget body repository.Budget true "Budget data"
// @Success 201 {object} repository.Budget
// @Failure 400 {object} repository.Response
// @Failure 401 {object} repository.Response
// @Failure 500 {object} repository.Response
// @Router /api/v1/budgets [post]
func (h *BudgetHandler) Create(c fiber.Ctx) error {
	userID, err := services.GetUserIDFromContext(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(repository.Response{
			Status:  "error",
			Message: "unauthorized",
			Data:    nil,
		})
	}

	var req repository.Budget
	if err := c.Bind().Body(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(repository.Response{
			Status:  "error",
			Message: "invalid request",
			Data:    nil,
		})
	}
	if req.Amount <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(repository.Response{
			Status:  "error",
			Message: "amount must be positive",
			Data:    nil,
		})
	}
	if req.Limit <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(repository.Response{
			Status:  "error",
			Message: "limit must be positive",
			Data:    nil,
		})
	}

	if req.Category == "" {
		return c.Status(fiber.StatusBadRequest).JSON(repository.Response{
			Status:  "error",
			Message: "category is required",
			Data:    nil,
		})
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
		return c.Status(fiber.StatusInternalServerError).JSON(repository.Response{
			Status:  "error",
			Message: "failed to create budget",
			Data:    nil,
		})
	}
	return c.Status(fiber.StatusCreated).JSON(repository.Response{
		Status:  "success",
		Message: "budget created successfully",
		Data:    req,
	})
}

// Update updates a budget by ID for the authenticated user.
// @Summary Update budget
// @Tags budgets
// @Accept json
// @Produce json
// @Param id path string true "Budget ID"
// @Param budget body repository.Budget true "Budget data"
// @Success 200 {object} repository.Budget
// @Failure 400 {object} repository.Response
// @Failure 401 {object} repository.Response
// @Failure 403 {object} repository.Response
// @Failure 404 {object} repository.Response
// @Router /api/v1/budgets/{id} [put]
func (h *BudgetHandler) Update(c fiber.Ctx) error {
	userID, err := services.GetUserIDFromContext(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(repository.Response{
			Status:  "error",
			Message: "unauthorized",
			Data:    nil,
		})
	}

	budgetID, err := bson.ObjectIDFromHex(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(repository.Response{
			Status:  "error",
			Message: "invalid budget id",
			Data:    nil,
		})
	}

	existing, err := h.repo.GetByID(context.Background(), budgetID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(repository.Response{
			Status:  "error",
			Message: "budget not found",
			Data:    nil,
		})
	}
	if existing.UserID != userID {
		return c.Status(fiber.StatusForbidden).JSON(repository.Response{
			Status:  "error",
			Message: "forbidden",
			Data:    nil,
		})
	}

	var req repository.Budget
	if err := c.Bind().Body(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(repository.Response{
			Status:  "error",
			Message: "invalid request",
			Data:    nil,
		})
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
		return c.Status(fiber.StatusInternalServerError).JSON(repository.Response{
			Status:  "error",
			Message: "failed to update budget",
			Data:    nil,
		})
	}
	return c.Status(fiber.StatusOK).JSON(repository.Response{
		Status:  "success",
		Message: "budget updated successfully",
		Data:    existing,
	})
}

// Delete deletes a budget by ID for the authenticated user.
// @Summary Delete budget
// @Tags budgets
// @Produce json
// @Param id path string true "Budget ID"
// @Success 200 {object} repository.Response
// @Failure 400 {object} repository.Response
// @Failure 401 {object} repository.Response
// @Failure 403 {object} repository.Response
// @Failure 404 {object} repository.Response
// @Router /api/v1/budgets/{id} [delete]
func (h *BudgetHandler) Delete(c fiber.Ctx) error {
	userID, err := services.GetUserIDFromContext(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(repository.Response{
			Status:  "error",
			Message: "unauthorized",
			Data:    nil,
		})
	}

	budgetID, err := bson.ObjectIDFromHex(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(repository.Response{
			Status:  "error",
			Message: "invalid budget id",
			Data:    nil,
		})
	}

	existing, err := h.repo.GetByID(context.Background(), budgetID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(repository.Response{
			Status:  "error",
			Message: "budget not found",
			Data:    nil,
		})
	}
	if existing.UserID != userID {
		return c.Status(fiber.StatusForbidden).JSON(repository.Response{
			Status:  "error",
			Message: "forbidden",
			Data:    nil,
		})
	}

	if err := h.repo.Delete(context.Background(), budgetID); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(repository.Response{
			Status:  "error",
			Message: "failed to delete budget",
			Data:    nil,
		})
	}
	return c.Status(fiber.StatusOK).JSON(repository.Response{
		Status:  "success",
		Message: "budget deleted successfully",
		Data:    map[string]bool{"deleted": true},
	})
}

// IncreaseAmount godoc
// @Summary      Increase the amount of a budget
// @Description  Increases the amount value for the specified budget of the authenticated user
// @Tags         budgets
// @Accept       json
// @Produce      json
// @Param        id     path      string                     true   "Budget ID"
// @Param        amount query   float64                      false  "Amount to increase"  default(0)
// @Success      200    {object}  repository.Response        "Amount increased successfully"
// @Failure      400    {object}  repository.Response        "Invalid budget id, request, or amount"
// @Failure      401    {object}  repository.Response        "Unauthorized"
// @Failure      403    {object}  repository.Response        "Forbidden"
// @Failure      404    {object}  repository.Response        "Budget not found"
// @Failure      500    {object}  repository.Response        "Failed to increase amount"
// @Router       /api/v1/budgets/{id}/increase [post]
func (h *BudgetHandler) IncreaseAmount(c fiber.Ctx) error {
	userID, err := services.GetUserIDFromContext(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(repository.Response{
			Status:  "error",
			Message: "unauthorized",
			Data:    nil,
		})
	}

	budgetID, err := bson.ObjectIDFromHex(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(repository.Response{
			Status:  "error",
			Message: "invalid budget id",
			Data:    nil,
		})
	}

	existing, err := h.repo.GetByID(context.Background(), budgetID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(repository.Response{
			Status:  "error",
			Message: "budget not found",
			Data:    nil,
		})
	}
	if existing.UserID != userID {
		return c.Status(fiber.StatusForbidden).JSON(repository.Response{
			Status:  "error",
			Message: "forbidden",
			Data:    nil,
		})
	}

	amount := c.Query("amount", "0")
	// parse amount float
	amountFloat, err := strconv.ParseFloat(amount, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(repository.Response{
			Status:  "error",
			Message: "invalid amount",
			Data:    nil,
		})
	}
	if amountFloat <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(repository.Response{
			Status:  "error",
			Message: "amount must be positive",
			Data:    nil,
		})
	}

	existing.Amount += amountFloat
	existing.UpdatedAt = time.Now()
	if err := h.repo.Update(context.Background(), budgetID, existing); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(repository.Response{
			Status:  "error",
			Message: "failed to increase amount",
			Data:    nil,
		})
	}
	return c.Status(fiber.StatusOK).JSON(repository.Response{
		Status:  "success",
		Message: "amount increased successfully",
		Data:    existing,
	})
}

// DecreaseAmount godoc
// @Summary      Decrease the amount of a budget
// @Description  Decreases the amount value for the specified budget of the authenticated user
// @Tags         budgets
// @Accept       json
// @Produce      json
// @Param        id     path      string                     true   "Budget ID"
// @Param        amount query   float64                      false  "Amount to decrease"  default(0)
// @Success      200    {object}  repository.Response        "Amount decreased successfully"
// @Failure      400    {object}  repository.Response        "Invalid budget id, request, or amount"
// @Failure      401    {object}  repository.Response        "Unauthorized"
// @Failure      403    {object}  repository.Response        "Forbidden"
// @Failure      404    {object}  repository.Response        "Budget not found"
// @Failure      500    {object}  repository.Response        "Failed to decrease amount"
// @Router       /api/v1/budgets/{id}/decrease [post]
func (h *BudgetHandler) DecreaseAmount(c fiber.Ctx) error {
	userID, err := services.GetUserIDFromContext(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(repository.Response{
			Status:  "error",
			Message: "unauthorized",
			Data:    nil,
		})
	}

	budgetID, err := bson.ObjectIDFromHex(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(repository.Response{
			Status:  "error",
			Message: "invalid budget id",
			Data:    nil,
		})
	}

	existing, err := h.repo.GetByID(context.Background(), budgetID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(repository.Response{
			Status:  "error",
			Message: "budget not found",
			Data:    nil,
		})
	}
	if existing.UserID != userID {
		return c.Status(fiber.StatusForbidden).JSON(repository.Response{
			Status:  "error",
			Message: "forbidden",
			Data:    nil,
		})
	}

	amount := c.Query("amount", "0")
	// parse amount float
	amountFloat, err := strconv.ParseFloat(amount, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(repository.Response{
			Status:  "error",
			Message: "invalid amount",
			Data:    nil,
		})
	}
	if amountFloat <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(repository.Response{
			Status:  "error",
			Message: "amount must be positive",
			Data:    nil,
		})
	}

	existing.Amount -= amountFloat
	existing.UpdatedAt = time.Now()
	if err := h.repo.Update(context.Background(), budgetID, existing); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(repository.Response{
			Status:  "error",
			Message: "failed to decrease amount",
			Data:    nil,
		})
	}
	return c.Status(fiber.StatusOK).JSON(repository.Response{
		Status:  "success",
		Message: "amount decreased successfully",
		Data:    existing,
	})
}
