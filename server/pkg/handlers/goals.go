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
// @Summary      List goals
// @Tags         goals
// @Produce      json
// @Success      200 {object} repository.Response[[]repository.Goal]
// @Failure      401 {object} repository.Response[repository.Empty]
// @Failure      500 {object} repository.Response[repository.Empty]
// @Router       /api/v1/goals [get]
func (h *GoalHandler) List(c fiber.Ctx) error {
	userID, err := services.GetUserIDFromContext(c)
	if err != nil {
		return repository.Unauthorized(c, "unauthorized")
	}
	goals, err := h.repo.ListByUser(context.Background(), userID)
	if err != nil {
		return repository.Internal(c, "failed to load goals")
	}
	return repository.OK(c, "goals loaded", goals)
}

// GetByID gets a single goal by ID for the authenticated user.
// @Summary      Get goal by ID
// @Tags         goals
// @Produce      json
// @Param        id  path     string true "Goal ID"
// @Success      200 {object} repository.Response[repository.Goal]
// @Failure      400 {object} repository.Response[repository.Empty]
// @Failure      401 {object} repository.Response[repository.Empty]
// @Failure      403 {object} repository.Response[repository.Empty]
// @Failure      404 {object} repository.Response[repository.Empty]
// @Router       /api/v1/goals/{id} [get]
func (h *GoalHandler) GetByID(c fiber.Ctx) error {
	userID, err := services.GetUserIDFromContext(c)
	if err != nil {
		return repository.Unauthorized(c, "unauthorized")
	}

	goalID, err := bson.ObjectIDFromHex(c.Params("id"))
	if err != nil {
		return repository.BadRequest(c, "invalid goal id")
	}

	g, err := h.repo.GetByID(context.Background(), goalID)
	if err != nil || g == nil {
		return repository.NotFound(c, "goal not found")
	}
	if g.UserID != userID {
		return repository.Forbidden(c, "forbidden")
	}
	return repository.OK(c, "goal found", g)
}

// Create creates a new goal for the authenticated user.
// @Summary      Create goal
// @Tags         goals
// @Accept       json
// @Produce      json
// @Param        goal body     repository.Goal true "Goal data"
// @Success      201  {object} repository.Response[repository.Goal]
// @Failure      400  {object} repository.Response[repository.Empty]
// @Failure      401  {object} repository.Response[repository.Empty]
// @Failure      500  {object} repository.Response[repository.Empty]
// @Router       /api/v1/goals [post]
func (h *GoalHandler) Create(c fiber.Ctx) error {
	userID, err := services.GetUserIDFromContext(c)
	if err != nil {
		return repository.Unauthorized(c, "unauthorized")
	}

	var req repository.Goal
	if err := c.Bind().Body(&req); err != nil {
		return repository.BadRequest(c, "invalid request")
	}
	if req.Name == "" {
		return repository.BadRequest(c, "name is required")
	}
	if req.TargetAmount <= 0 {
		return repository.BadRequest(c, "target_amount must be positive")
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
		return repository.Internal(c, "failed to create goal")
	}
	return repository.Created(c, "goal created", req)
}

// Update updates a goal by ID for the authenticated user.
// @Summary      Update goal
// @Tags         goals
// @Accept       json
// @Produce      json
// @Param        id   path     string          true "Goal ID"
// @Param        goal body     repository.Goal true "Goal data"
// @Success      200  {object} repository.Response[repository.Goal]
// @Failure      400  {object} repository.Response[repository.Empty]
// @Failure      401  {object} repository.Response[repository.Empty]
// @Failure      403  {object} repository.Response[repository.Empty]
// @Failure      404  {object} repository.Response[repository.Empty]
// @Failure      500  {object} repository.Response[repository.Empty]
// @Router       /api/v1/goals/{id} [put]
func (h *GoalHandler) Update(c fiber.Ctx) error {
	userID, err := services.GetUserIDFromContext(c)
	if err != nil {
		return repository.Unauthorized(c, "unauthorized")
	}

	goalID, err := bson.ObjectIDFromHex(c.Params("id"))
	if err != nil {
		return repository.BadRequest(c, "invalid goal id")
	}

	existing, err := h.repo.GetByID(context.Background(), goalID)
	if err != nil || existing == nil {
		return repository.NotFound(c, "goal not found")
	}
	if existing.UserID != userID {
		return repository.Forbidden(c, "forbidden")
	}

	var req repository.Goal
	if err := c.Bind().Body(&req); err != nil {
		return repository.BadRequest(c, "invalid request")
	}

	existing.Name = req.Name
	existing.Description = req.Description
	existing.TargetAmount = req.TargetAmount
	existing.CurrentAmount = req.CurrentAmount
	existing.Currency = req.Currency
	existing.Deadline = req.Deadline
	existing.UpdatedAt = time.Now()

	if err := h.repo.Update(context.Background(), goalID, existing); err != nil {
		return repository.Internal(c, "failed to update goal")
	}
	return repository.OK(c, "goal updated", existing)
}

// Delete deletes a goal by ID for the authenticated user.
// @Summary      Delete goal
// @Tags         goals
// @Produce      json
// @Param        id  path     string true "Goal ID"
// @Success      200 {object} repository.Response[repository.Empty]
// @Failure      400 {object} repository.Response[repository.Empty]
// @Failure      401 {object} repository.Response[repository.Empty]
// @Failure      403 {object} repository.Response[repository.Empty]
// @Failure      404 {object} repository.Response[repository.Empty]
// @Failure      500 {object} repository.Response[repository.Empty]
// @Router       /api/v1/goals/{id} [delete]
func (h *GoalHandler) Delete(c fiber.Ctx) error {
	userID, err := services.GetUserIDFromContext(c)
	if err != nil {
		return repository.Unauthorized(c, "unauthorized")
	}

	goalID, err := bson.ObjectIDFromHex(c.Params("id"))
	if err != nil {
		return repository.BadRequest(c, "invalid goal id")
	}

	existing, err := h.repo.GetByID(context.Background(), goalID)
	if err != nil || existing == nil {
		return repository.NotFound(c, "goal not found")
	}
	if existing.UserID != userID {
		return repository.Forbidden(c, "forbidden")
	}

	if err := h.repo.Delete(context.Background(), goalID); err != nil {
		return repository.Internal(c, "failed to delete goal")
	}
	return repository.Ack(c, "goal deleted")
}

// Contribute increments the goal's current_amount by the given amount.
// @Summary      Contribute to a goal
// @Tags         goals
// @Accept       json
// @Produce      json
// @Param        id   path     string             true "Goal ID"
// @Param        body body     ContributeRequest  true "Contribution body"
// @Success      200  {object} repository.Response[repository.Goal]
// @Failure      400  {object} repository.Response[repository.Empty]
// @Failure      401  {object} repository.Response[repository.Empty]
// @Failure      403  {object} repository.Response[repository.Empty]
// @Failure      404  {object} repository.Response[repository.Empty]
// @Failure      500  {object} repository.Response[repository.Empty]
// @Router       /api/v1/goals/{id}/contribute [post]
func (h *GoalHandler) Contribute(c fiber.Ctx) error {
	userID, err := services.GetUserIDFromContext(c)
	if err != nil {
		return repository.Unauthorized(c, "unauthorized")
	}

	goalID, err := bson.ObjectIDFromHex(c.Params("id"))
	if err != nil {
		return repository.BadRequest(c, "invalid goal id")
	}

	existing, err := h.repo.GetByID(context.Background(), goalID)
	if err != nil || existing == nil {
		return repository.NotFound(c, "goal not found")
	}
	if existing.UserID != userID {
		return repository.Forbidden(c, "forbidden")
	}

	var req ContributeRequest
	if err := c.Bind().Body(&req); err != nil {
		return repository.BadRequest(c, "invalid request")
	}
	if req.Amount <= 0 {
		return repository.BadRequest(c, "amount is required")
	}

	if err := h.repo.Contribute(context.Background(), goalID, req.Amount); err != nil {
		return repository.Internal(c, "failed to contribute")
	}

	updated, err := h.repo.GetByID(context.Background(), goalID)
	if err != nil || updated == nil {
		return repository.Internal(c, "failed to reload goal")
	}
	return repository.OK(c, "contribution added", updated)
}

// ContributeRequest is the body of POST /goals/:id/contribute.
type ContributeRequest struct {
	Amount float64 `json:"amount"`
}
