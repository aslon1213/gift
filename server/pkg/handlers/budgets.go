package handlers

import (
	"aslon1213/gift/pkg/repository"

	"github.com/gofiber/fiber/v3"
)

type BudgetHandler struct {
	repo *repository.BudgetRepository
}

func NewBudgetHandler(repo *repository.BudgetRepository) *BudgetHandler {
	return &BudgetHandler{repo: repo}
}

func (h *BudgetHandler) List(c fiber.Ctx) error {
	return c.Status(fiber.StatusNotImplemented).JSON(fiber.Map{"error": "not implemented"})
}

func (h *BudgetHandler) GetByID(c fiber.Ctx) error {
	return c.Status(fiber.StatusNotImplemented).JSON(fiber.Map{"error": "not implemented"})
}

func (h *BudgetHandler) Create(c fiber.Ctx) error {
	return c.Status(fiber.StatusNotImplemented).JSON(fiber.Map{"error": "not implemented"})
}

func (h *BudgetHandler) Update(c fiber.Ctx) error {
	return c.Status(fiber.StatusNotImplemented).JSON(fiber.Map{"error": "not implemented"})
}

func (h *BudgetHandler) Delete(c fiber.Ctx) error {
	return c.Status(fiber.StatusNotImplemented).JSON(fiber.Map{"error": "not implemented"})
}
