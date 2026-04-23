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

func (h *AlertHandler) List(c fiber.Ctx) error {
	return c.Status(fiber.StatusNotImplemented).JSON(fiber.Map{"error": "not implemented"})
}

func (h *AlertHandler) GetByID(c fiber.Ctx) error {
	return c.Status(fiber.StatusNotImplemented).JSON(fiber.Map{"error": "not implemented"})
}

func (h *AlertHandler) Create(c fiber.Ctx) error {
	return c.Status(fiber.StatusNotImplemented).JSON(fiber.Map{"error": "not implemented"})
}

func (h *AlertHandler) Update(c fiber.Ctx) error {
	return c.Status(fiber.StatusNotImplemented).JSON(fiber.Map{"error": "not implemented"})
}

func (h *AlertHandler) Delete(c fiber.Ctx) error {
	return c.Status(fiber.StatusNotImplemented).JSON(fiber.Map{"error": "not implemented"})
}
