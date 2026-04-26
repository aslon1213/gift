package handlers

import (
	"aslon1213/gift/pkg/repository"

	"github.com/gofiber/fiber/v3"
)

type AlertHandler struct {
	repo *repository.AlertRepository
}

func NewAlertHandler(repo *repository.AlertRepository) *AlertHandler {
	return &AlertHandler{repo: repo}
}

// List lists alerts.
// @Summary      List alerts
// @Tags         alerts
// @Produce      json
// @Failure      501  {object}  repository.Response[repository.Empty]
// @Router       /api/v1/alerts [get]
func (h *AlertHandler) List(c fiber.Ctx) error {
	return repository.NotImplemented(c, "not implemented")
}

// GetByID gets an alert by ID.
// @Summary      Get alert by ID
// @Tags         alerts
// @Produce      json
// @Param        id   path      string true "Alert ID"
// @Failure      501  {object}  repository.Response[repository.Empty]
// @Router       /api/v1/alerts/{id} [get]
func (h *AlertHandler) GetByID(c fiber.Ctx) error {
	return repository.NotImplemented(c, "not implemented")
}

// Create creates an alert.
// @Summary      Create alert
// @Tags         alerts
// @Produce      json
// @Failure      501  {object}  repository.Response[repository.Empty]
// @Router       /api/v1/alerts [post]
func (h *AlertHandler) Create(c fiber.Ctx) error {
	return repository.NotImplemented(c, "not implemented")
}

// Update updates an alert.
// @Summary      Update alert
// @Tags         alerts
// @Produce      json
// @Param        id   path      string true "Alert ID"
// @Failure      501  {object}  repository.Response[repository.Empty]
// @Router       /api/v1/alerts/{id} [put]
func (h *AlertHandler) Update(c fiber.Ctx) error {
	return repository.NotImplemented(c, "not implemented")
}

// Delete deletes an alert.
// @Summary      Delete alert
// @Tags         alerts
// @Produce      json
// @Param        id   path      string true "Alert ID"
// @Failure      501  {object}  repository.Response[repository.Empty]
// @Router       /api/v1/alerts/{id} [delete]
func (h *AlertHandler) Delete(c fiber.Ctx) error {
	return repository.NotImplemented(c, "not implemented")
}
