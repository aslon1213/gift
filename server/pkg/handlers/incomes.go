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
// @Summary List incomes
// @Description Returns all incomes for the authenticated user
// @Tags incomes
// @Produce json
// @Success 200 {array} repository.Income
// @Failure 401 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/v1/incomes [get]
func (h *IncomeHandler) List(c fiber.Ctx) error {
	userID, err := services.GetUserIDFromContext(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	// Fetch all incomes for this user
	incomes, err := h.repo.List(context.Background())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to load incomes"})
	}

	userIncomes := make([]*repository.Income, 0)
	for _, inc := range incomes {
		if inc.UserID == userID {
			userIncomes = append(userIncomes, inc)
		}
	}
	return c.Status(fiber.StatusOK).JSON(userIncomes)
}

// GetByID gets a single income by ID for the authenticated user.
// @Summary Get income by ID
// @Description Returns the income by its ObjectID, for the authenticated user
// @Tags incomes
// @Produce json
// @Param id path string true "Income ID"
// @Success 200 {object} repository.Income
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /api/v1/incomes/{id} [get]
func (h *IncomeHandler) GetByID(c fiber.Ctx) error {
	userID, err := services.GetUserIDFromContext(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	incomeIDHex := c.Params("id")
	incomeID, err := bson.ObjectIDFromHex(incomeIDHex)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid income id"})
	}

	income, err := h.repo.GetByID(context.Background(), incomeID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "income not found"})
	}
	if income.UserID != userID {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "forbidden"})
	}
	return c.Status(fiber.StatusOK).JSON(income)
}

// Create creates a new income for the authenticated user.
// @Summary Create income
// @Description Creates a new income entry for the authenticated user
// @Tags incomes
// @Accept json
// @Produce json
// @Param income body repository.Income true "Income data"
// @Success 201 {object} repository.Income
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/v1/incomes [post]
func (h *IncomeHandler) Create(c fiber.Ctx) error {
	userID, err := services.GetUserIDFromContext(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	var req repository.Income
	if err := c.Bind().Body(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request"})
	}
	req.ID = bson.NewObjectID() // ensure new object id
	req.UserID = userID
	req.CreatedAt = time.Now()
	req.UpdatedAt = time.Now()

	if req.Date.IsZero() {
		req.Date = time.Now()
	}

	if req.Amount <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "amount must be positive"})
	}

	if err := h.repo.Create(context.Background(), &req); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to create income"})
	}

	// Update the user's balance
	user, err := h.userRepo.GetByID(context.Background(), userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to update balance, user not found"})
	}
	newBalance := user.Balance + req.Amount
	if err := h.userRepo.UpdateBalance(context.Background(), userID, newBalance); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to update balance"})
	}

	return c.Status(fiber.StatusCreated).JSON(req)
}

// Update updates an income by ID for the authenticated user.
// @Summary Update income
// @Description Updates an existing income by its ObjectID for the authenticated user
// @Tags incomes
// @Accept json
// @Produce json
// @Param id path string true "Income ID"
// @Param income body repository.Income true "Income data"
// @Success 200 {object} repository.Income
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/v1/incomes/{id} [put]
func (h *IncomeHandler) Update(c fiber.Ctx) error {
	userID, err := services.GetUserIDFromContext(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	incomeIDHex := c.Params("id")
	incomeID, err := bson.ObjectIDFromHex(incomeIDHex)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid income id"})
	}

	existing, err := h.repo.GetByID(context.Background(), incomeID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "income not found"})
	}
	if existing.UserID != userID {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "forbidden"})
	}
	var req repository.Income
	if err := c.Bind().Body(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request"})
	}

	// Only allow modification of certain fields:
	amountBefore := existing.Amount

	existing.Amount = req.Amount
	existing.Currency = req.Currency
	existing.Source = req.Source
	existing.Description = req.Description
	existing.Date = req.Date
	existing.UpdatedAt = time.Now()

	if err := h.repo.Update(context.Background(), incomeID, existing); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to update income"})
	}

	// If the amount has changed, update user balance
	if req.Amount != amountBefore {
		user, err := h.userRepo.GetByID(context.Background(), userID)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to update balance, user not found"})
		}
		newBalance := user.Balance + (req.Amount - amountBefore)
		if err := h.userRepo.UpdateBalance(context.Background(), userID, newBalance); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to update balance"})
		}
	}

	return c.Status(fiber.StatusOK).JSON(existing)
}

// Delete deletes an income by ID for the authenticated user.
// @Summary Delete income
// @Description Deletes an income entry for the authenticated user by its ObjectID
// @Tags incomes
// @Produce json
// @Param id path string true "Income ID"
// @Success 200 {object} map[string]bool
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/v1/incomes/{id} [delete]
func (h *IncomeHandler) Delete(c fiber.Ctx) error {
	userID, err := services.GetUserIDFromContext(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	incomeIDHex := c.Params("id")
	incomeID, err := bson.ObjectIDFromHex(incomeIDHex)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid income id"})
	}

	income, err := h.repo.GetByID(context.Background(), incomeID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "income not found"})
	}
	if income.UserID != userID {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "forbidden"})
	}

	if err := h.repo.Delete(context.Background(), incomeID); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to delete income"})
	}

	// Reduce the user's balance
	user, err := h.userRepo.GetByID(context.Background(), userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to update balance, user not found"})
	}
	newBalance := user.Balance - income.Amount
	if newBalance < 0 {
		newBalance = 0 // Or allow negative balances if your business logic allows
	}
	if err := h.userRepo.UpdateBalance(context.Background(), userID, newBalance); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to update balance"})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{"deleted": true})
}
