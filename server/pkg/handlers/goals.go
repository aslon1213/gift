package handlers

import (
	"aslon1213/gift/pkg/repository"
	"aslon1213/gift/services"
	"context"
	"time"

	"github.com/gofiber/fiber/v3"
	"go.mongodb.org/mongo-driver/v2/bson"
)

type GoalHandler struct {
	repo *repository.GoalRepository
}

func NewGoalHandler(repo *repository.GoalRepository) *GoalHandler {
	return &GoalHandler{repo: repo}
}

// List lists all goals for the authenticated user.
// @Summary List goals
// @Description Returns all goals for the authenticated user
// @Tags goals
// @Produce json
// @Success 200 {array} repository.Goal
// @Failure 401 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/v1/goals [get]
func (h *GoalHandler) List(c fiber.Ctx) error {
	userID, err := services.GetUserIDFromContext(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}
	goals, err := h.repo.ListByUser(context.Background(), userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to load goals"})
	}
	return c.Status(fiber.StatusOK).JSON(goals)
}

// GetByID gets a single goal by ID for the authenticated user.
// @Summary Get goal by ID
// @Tags goals
// @Produce json
// @Param id path string true "Goal ID"
// @Success 200 {object} repository.Goal
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /api/v1/goals/{id} [get]
func (h *GoalHandler) GetByID(c fiber.Ctx) error {
	userID, err := services.GetUserIDFromContext(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	goalID, err := bson.ObjectIDFromHex(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid goal id"})
	}

	g, err := h.repo.GetByID(context.Background(), goalID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "goal not found"})
	}
	if g.UserID != userID {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "forbidden"})
	}
	return c.Status(fiber.StatusOK).JSON(g)
}

// Create creates a new goal for the authenticated user.
// @Summary Create goal
// @Tags goals
// @Accept json
// @Produce json
// @Param goal body repository.Goal true "Goal data"
// @Success 201 {object} repository.Goal
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/v1/goals [post]
func (h *GoalHandler) Create(c fiber.Ctx) error {
	userID, err := services.GetUserIDFromContext(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	var req repository.Goal
	if err := c.Bind().Body(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request"})
	}
	if req.Name == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "name is required"})
	}
	if req.TargetAmount <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "target_amount must be positive"})
	}
	if req.CurrentAmount < 0 {
		req.CurrentAmount = 0
	}

	req.ID = bson.NewObjectID()
	req.UserID = userID
	if req.Currency == "" {
		req.Currency = "$"
	}
	req.CreatedAt = time.Now()
	req.UpdatedAt = time.Now()

	if err := h.repo.Create(context.Background(), &req); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to create goal"})
	}
	return c.Status(fiber.StatusCreated).JSON(req)
}

// Update updates a goal by ID for the authenticated user.
// @Summary Update goal
// @Tags goals
// @Accept json
// @Produce json
// @Param id path string true "Goal ID"
// @Param goal body repository.Goal true "Goal data"
// @Success 200 {object} repository.Goal
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /api/v1/goals/{id} [put]
func (h *GoalHandler) Update(c fiber.Ctx) error {
	userID, err := services.GetUserIDFromContext(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	goalID, err := bson.ObjectIDFromHex(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid goal id"})
	}

	existing, err := h.repo.GetByID(context.Background(), goalID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "goal not found"})
	}
	if existing.UserID != userID {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "forbidden"})
	}

	var req repository.Goal
	if err := c.Bind().Body(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request"})
	}

	existing.Name = req.Name
	existing.Description = req.Description
	existing.TargetAmount = req.TargetAmount
	existing.CurrentAmount = req.CurrentAmount
	existing.Currency = req.Currency
	existing.Deadline = req.Deadline
	existing.UpdatedAt = time.Now()

	if err := h.repo.Update(context.Background(), goalID, existing); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to update goal"})
	}
	return c.Status(fiber.StatusOK).JSON(existing)
}

// Delete deletes a goal by ID for the authenticated user.
// @Summary Delete goal
// @Tags goals
// @Produce json
// @Param id path string true "Goal ID"
// @Success 200 {object} map[string]bool
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /api/v1/goals/{id} [delete]
func (h *GoalHandler) Delete(c fiber.Ctx) error {
	userID, err := services.GetUserIDFromContext(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	goalID, err := bson.ObjectIDFromHex(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid goal id"})
	}

	existing, err := h.repo.GetByID(context.Background(), goalID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "goal not found"})
	}
	if existing.UserID != userID {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "forbidden"})
	}

	if err := h.repo.Delete(context.Background(), goalID); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to delete goal"})
	}
	return c.Status(fiber.StatusOK).JSON(fiber.Map{"deleted": true})
}

// Contribute increments the goal's current_amount by the given amount.
// @Summary Contribute to goal
// @Description Adds `amount` to the goal's current_amount
// @Tags goals
// @Accept json
// @Produce json
// @Param id path string true "Goal ID"
// @Param body body handlers.ContributeRequest true "Contribution"
// @Success 200 {object} repository.Goal
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /api/v1/goals/{id}/contribute [post]
func (h *GoalHandler) Contribute(c fiber.Ctx) error {
	userID, err := services.GetUserIDFromContext(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	goalID, err := bson.ObjectIDFromHex(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid goal id"})
	}

	existing, err := h.repo.GetByID(context.Background(), goalID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "goal not found"})
	}
	if existing.UserID != userID {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "forbidden"})
	}

	var req ContributeRequest
	if err := c.Bind().Body(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request"})
	}
	if req.Amount <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "amount is required"})
	}

	if err := h.repo.Contribute(context.Background(), goalID, req.Amount); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to contribute"})
	}

	updated, err := h.repo.GetByID(context.Background(), goalID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to reload goal"})
	}
	return c.Status(fiber.StatusOK).JSON(updated)
}

// ContributeRequest is the body of POST /goals/:id/contribute.
type ContributeRequest struct {
	Amount float64 `json:"amount"`
}
