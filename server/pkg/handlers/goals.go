package handlers

import (
	"aslon1213/gift/pkg/repository"

	"github.com/gofiber/fiber/v3"
)

type GoalHandler struct {
	repo *repository.GoalRepository
}

func NewGoalHandler(repo *repository.GoalRepository) *GoalHandler {
	return &GoalHandler{repo: repo}
}

func (h *GoalHandler) List(c fiber.Ctx) error {
	return c.Status(fiber.StatusNotImplemented).JSON(fiber.Map{"error": "not implemented"})
}

func (h *GoalHandler) GetByID(c fiber.Ctx) error {
	return c.Status(fiber.StatusNotImplemented).JSON(fiber.Map{"error": "not implemented"})
}

func (h *GoalHandler) Create(c fiber.Ctx) error {
	return c.Status(fiber.StatusNotImplemented).JSON(fiber.Map{"error": "not implemented"})
}

func (h *GoalHandler) Update(c fiber.Ctx) error {
	return c.Status(fiber.StatusNotImplemented).JSON(fiber.Map{"error": "not implemented"})
}

func (h *GoalHandler) Delete(c fiber.Ctx) error {
	return c.Status(fiber.StatusNotImplemented).JSON(fiber.Map{"error": "not implemented"})
}
