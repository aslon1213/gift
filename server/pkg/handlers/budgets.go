package handlers

import (
	"aslon1213/gift/pkg/repository"
	"aslon1213/gift/services"
	"context"
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
// @Failure 401 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/v1/budgets [get]
func (h *BudgetHandler) List(c fiber.Ctx) error {
	userID, err := services.GetUserIDFromContext(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	budgets, err := h.repo.ListByUser(context.Background(), userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to load budgets"})
	}
	return c.Status(fiber.StatusOK).JSON(budgets)
}

// GetByID gets a single budget by ID for the authenticated user.
// @Summary Get budget by ID
// @Tags budgets
// @Produce json
// @Param id path string true "Budget ID"
// @Success 200 {object} repository.Budget
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /api/v1/budgets/{id} [get]
func (h *BudgetHandler) GetByID(c fiber.Ctx) error {
	userID, err := services.GetUserIDFromContext(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	budgetID, err := bson.ObjectIDFromHex(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid budget id"})
	}

	b, err := h.repo.GetByID(context.Background(), budgetID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "budget not found"})
	}
	if b.UserID != userID {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "forbidden"})
	}
	return c.Status(fiber.StatusOK).JSON(b)
}

// Create creates a new budget for the authenticated user.
// @Summary Create budget
// @Tags budgets
// @Accept json
// @Produce json
// @Param budget body repository.Budget true "Budget data"
// @Success 201 {object} repository.Budget
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/v1/budgets [post]
func (h *BudgetHandler) Create(c fiber.Ctx) error {
	userID, err := services.GetUserIDFromContext(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	var req repository.Budget
	if err := c.Bind().Body(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request"})
	}
	if req.Amount <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "amount must be positive"})
	}
	if req.Category == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "category is required"})
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
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to create budget"})
	}
	return c.Status(fiber.StatusCreated).JSON(req)
}

// Update updates a budget by ID for the authenticated user.
// @Summary Update budget
// @Tags budgets
// @Accept json
// @Produce json
// @Param id path string true "Budget ID"
// @Param budget body repository.Budget true "Budget data"
// @Success 200 {object} repository.Budget
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /api/v1/budgets/{id} [put]
func (h *BudgetHandler) Update(c fiber.Ctx) error {
	userID, err := services.GetUserIDFromContext(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	budgetID, err := bson.ObjectIDFromHex(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid budget id"})
	}

	existing, err := h.repo.GetByID(context.Background(), budgetID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "budget not found"})
	}
	if existing.UserID != userID {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "forbidden"})
	}

	var req repository.Budget
	if err := c.Bind().Body(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request"})
	}

	existing.Category = req.Category
	existing.Amount = req.Amount
	existing.Currency = req.Currency
	existing.Period = req.Period
	existing.StartDate = req.StartDate
	existing.EndDate = req.EndDate
	existing.UpdatedAt = time.Now()

	if err := h.repo.Update(context.Background(), budgetID, existing); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to update budget"})
	}
	return c.Status(fiber.StatusOK).JSON(existing)
}

// Delete deletes a budget by ID for the authenticated user.
// @Summary Delete budget
// @Tags budgets
// @Produce json
// @Param id path string true "Budget ID"
// @Success 200 {object} map[string]bool
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /api/v1/budgets/{id} [delete]
func (h *BudgetHandler) Delete(c fiber.Ctx) error {
	userID, err := services.GetUserIDFromContext(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	budgetID, err := bson.ObjectIDFromHex(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid budget id"})
	}

	existing, err := h.repo.GetByID(context.Background(), budgetID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "budget not found"})
	}
	if existing.UserID != userID {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "forbidden"})
	}

	if err := h.repo.Delete(context.Background(), budgetID); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to delete budget"})
	}
	return c.Status(fiber.StatusOK).JSON(fiber.Map{"deleted": true})
}
