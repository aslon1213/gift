package handlers

import (
	"aslon1213/gift/pkg/repository"
	"aslon1213/gift/services"
	"net/http"
	"slices"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v3"
	"go.mongodb.org/mongo-driver/v2/bson"
)

type SpendingHandler struct {
	repo       *repository.SpendingRepository
	groupRepo  *repository.GroupRepository
	userRepo   *repository.UserRepository
	budgetRepo *repository.BudgetRepository
}

func NewSpendingHandler(repo *repository.SpendingRepository, groupRepo *repository.GroupRepository, userRepo *repository.UserRepository, budgetRepo *repository.BudgetRepository) *SpendingHandler {
	return &SpendingHandler{repo: repo, groupRepo: groupRepo, userRepo: userRepo, budgetRepo: budgetRepo}
}

// Query godoc
// @Summary      List spendings
// @Description  Returns a list of spendings filtered by user_id, group_id, category, date range; supports limit/offset
// @Tags         spendings
// @Accept       json
// @Produce      json
// @Param        user_id query    string false "User ID (hex)"
// @Param        group_id query   string false "Group ID (hex)"
// @Param        category query   string false "Category"
// @Param        start_date query string false "Start date (RFC3339)"
// @Param        end_date query   string false "End date (RFC3339)"
// @Param        limit query      int    false "Limit"
// @Param        offset query     int    false "Offset"
// @Success      200 {object}     repository.Response
// @Failure      500 {object}     repository.Response
// @Router       /spendings [get]
func (h *SpendingHandler) Query(c fiber.Ctx) error {

	// can be queried by user_id, group_id, category, start_date, end_date
	// can be sorted by amount, date
	query := bson.M{}
	user_id, err := bson.ObjectIDFromHex(c.Query("user_id"))
	if err == nil {
		query["user_id"] = user_id
	}
	group_id, err := bson.ObjectIDFromHex(c.Query("group_id"))
	if err == nil {
		query["group_id"] = group_id
	}
	category := c.Query("category")
	if category != "" {
		query["category"] = category
	}
	start_date, err := time.Parse(time.RFC3339, c.Query("start_date"))
	if err == nil {
		query["date"] = bson.M{"$gte": start_date}
	}
	end_date, err := time.Parse(time.RFC3339, c.Query("end_date"))
	if err == nil {
		query["date"] = bson.M{"$lte": end_date}
	}
	// sort_by := c.Query("sort_by")
	// if sort_by != "" {
	// 	query["sort_by"] = sort_by
	// }
	limit, err := strconv.Atoi(c.Query("limit"))
	if err == nil {
		query["limit"] = limit
	}
	offset, err := strconv.Atoi(c.Query("offset"))
	if err == nil {
		query["offset"] = offset
	}

	spendings, err := h.repo.Query(c.Context(), query)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(repository.NewResponse("error", "internal server error", nil))
	}
	return c.Status(http.StatusOK).JSON(repository.NewResponse("success", "spendings fetched successfully", spendings))
}

// GetByID godoc
// @Summary      Get spending by ID
// @Description  Get a spending by its ID
// @Tags         spendings
// @Accept       json
// @Produce      json
// @Param        id   path      string true  "Spending ID (hex)"
// @Success      200  {object}  repository.Response
// @Failure      400  {object}  repository.Response
// @Failure      500  {object}  repository.Response
// @Router       /spendings/{id} [get]
func (h *SpendingHandler) GetByID(c fiber.Ctx) error {
	spending_id, err := bson.ObjectIDFromHex(c.Params("id"))
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(repository.NewResponse("error", err.Error(), nil))
	}

	spending, err := h.repo.GetByID(c.Context(), spending_id)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(repository.NewResponse("error", "internal server error", nil))
	}
	return c.Status(http.StatusOK).JSON(repository.NewResponse("success", "spending fetched successfully", spending))
}

// CreateSpendingRequest represents the request body for creating a spending
type CreateSpendingRequest struct {
	GroupID     bson.ObjectID `json:"group_id"`
	Amount      float64       `json:"amount"`
	Currency    string        `json:"currency"`
	Category    string        `json:"category"`
	Description string        `json:"description"`
	Date        time.Time     `json:"date"`
}

// Create godoc
// @Summary      Create spending
// @Description  Create a new spending record
// @Tags         spendings
// @Accept       json
// @Produce      json
// @Param        body body     CreateSpendingRequest true "Spending data"
// @Success      200  {object} repository.Response
// @Failure      400  {object} repository.Response
// @Failure      403  {object} repository.Response
// @Failure      404  {object} repository.Response
// @Failure      500  {object} repository.Response
// @Security     ApiKeyAuth
// @Router       /spendings [post]
func (h *SpendingHandler) Create(c fiber.Ctx) error {

	user_id, err := services.GetUserIDFromContext(c)
	if err != nil {
		return c.Status(http.StatusUnauthorized).JSON(repository.NewResponse("error", err.Error(), nil))
	}

	var input CreateSpendingRequest
	if err := c.Bind().Body(&input); err != nil {
		return c.Status(http.StatusBadRequest).JSON(repository.NewResponse("error", "invalid request body", nil))
	}
	if input.Amount <= 0 {
		return c.Status(http.StatusBadRequest).JSON(repository.NewResponse("error", "amount must be greater than 0", nil))
	}
	if input.Currency == "" {
		input.Currency = "UZS"
	}
	if input.Category == "" {
		input.Category = "Undefined"
	}
	if input.Date.IsZero() {
		input.Date = time.Now()
	}

	if input.GroupID.IsZero() {
		return c.Status(http.StatusBadRequest).JSON(repository.NewResponse("error", "group_id is required", nil))
	}
	// check if the group exists and the user is a member of the group
	group, err := h.groupRepo.GetByID(c.Context(), input.GroupID)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(repository.NewResponse("error", "internal server error", nil))
	}
	if group == nil {
		return c.Status(http.StatusNotFound).JSON(repository.NewResponse("error", "group not found", nil))
	}
	// check if the user is a member of the group
	if !slices.Contains(group.MemberIDs, user_id) {
		return c.Status(http.StatusForbidden).JSON(repository.NewResponse("error", "user is not a member of the group", nil))
	}

	spending := &repository.Spending{
		UserID:      user_id,
		GroupID:     input.GroupID,
		Amount:      input.Amount,
		Currency:    input.Currency,
		Category:    input.Category,
		Description: input.Description,
		Date:        input.Date,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	err = h.repo.Create(c.Context(), spending)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(repository.NewResponse("error", "internal server error", nil))
	}

	// --- Update user balance (subtract amount) ---
	// First, retrieve current user for balance
	user, err := h.userRepo.GetByID(c.Context(), user_id)
	if err != nil || user == nil {
		return c.Status(http.StatusInternalServerError).JSON(repository.NewResponse("error", "failed to get user for balance update", nil))
	}
	newBalance := user.Balance - input.Amount
	if err := h.userRepo.UpdateBalance(c.Context(), user_id, newBalance); err != nil {
		return c.Status(http.StatusInternalServerError).JSON(repository.NewResponse("error", "failed to update user balance", nil))
	}
	// --- end user balance update

	return c.Status(http.StatusOK).JSON(repository.NewResponse("success", "spending created successfully", spending))
}

// UpdateSpendingRequest represents the request body for updating a spending
type UpdateSpendingRequest struct {
	Amount      float64   `json:"amount"`
	Currency    string    `json:"currency"`
	Category    string    `json:"category"`
	Description string    `json:"description"`
	Date        time.Time `json:"date"`
}

// Update godoc
// @Summary      Update spending
// @Description  Update an existing spending
// @Tags         spendings
// @Accept       json
// @Produce      json
// @Param        id   path      string true  "Spending ID (hex)"
// @Param        body body      UpdateSpendingRequest true "Spending fields to update"
// @Success      200  {object}  repository.Response
// @Failure      400  {object}  repository.Response
// @Failure      401  {object}  repository.Response
// @Failure      403  {object}  repository.Response
// @Failure      404  {object}  repository.Response
// @Failure      500  {object}  repository.Response
// @Security     ApiKeyAuth
// @Router       /spendings/{id} [put]
func (h *SpendingHandler) Update(c fiber.Ctx) error {
	// Get user_id from locals (set by middleware)
	userID, err := services.GetUserIDFromContext(c)
	if err != nil {
		return c.Status(http.StatusUnauthorized).JSON(repository.NewResponse("error", err.Error(), nil))
	}

	var input UpdateSpendingRequest
	if err := c.Bind().Body(&input); err != nil {
		return c.Status(http.StatusBadRequest).JSON(repository.NewResponse("error", "invalid request body", nil))
	}

	spending_id, err := bson.ObjectIDFromHex(c.Params("id"))
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(repository.NewResponse("error", err.Error(), nil))
	}

	spending, err := h.repo.GetByID(c.Context(), spending_id)
	if err != nil || spending == nil {
		return c.Status(http.StatusNotFound).JSON(repository.NewResponse("error", "spending not found", nil))
	}

	// Only the user who created (owner) can edit
	if spending.UserID != userID {
		return c.Status(http.StatusForbidden).JSON(repository.NewResponse("error", "only the owner can edit this spending", nil))
	}

	// Cache the old amount for balance correction logic
	oldAmount := spending.Amount

	if input.Amount > 0 {
		spending.Amount = input.Amount
	} else if input.Amount != 0 {
		// Only reject if user explicitly tries to set to <= 0
		return c.Status(http.StatusBadRequest).JSON(repository.NewResponse("error", "amount must be greater than 0", nil))
	}
	if input.Currency != "" {
		spending.Currency = input.Currency
	}
	if input.Category != "" {
		spending.Category = input.Category
	}
	if !input.Date.IsZero() {
		spending.Date = input.Date
	}

	err = h.repo.Update(c.Context(), spending_id, spending)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(repository.NewResponse("error", "internal server error", nil))
	}

	// --- Update user balance (correct for new/old amount) ---
	user, err := h.userRepo.GetByID(c.Context(), userID)
	if err != nil || user == nil {
		return c.Status(http.StatusInternalServerError).JSON(repository.NewResponse("error", "failed to get user for balance update", nil))
	}
	// To "reverse out" old amount and subtract new: balance = balance + oldAmount - newAmount
	newBalance := user.Balance + oldAmount - spending.Amount
	if err := h.userRepo.UpdateBalance(c.Context(), userID, newBalance); err != nil {
		return c.Status(http.StatusInternalServerError).JSON(repository.NewResponse("error", "failed to update user balance", nil))
	}
	// --- end user balance update

	return c.Status(http.StatusOK).JSON(repository.NewResponse("success", "spending updated successfully", spending))
}

// Delete godoc
// @Summary      Delete spending
// @Description  Delete an existing spending (owner only)
// @Tags         spendings
// @Accept       json
// @Produce      json
// @Param        id   path      string true  "Spending ID (hex)"
// @Success      200  {object}  repository.Response
// @Failure      400  {object}  repository.Response
// @Failure      401  {object}  repository.Response
// @Failure      403  {object}  repository.Response
// @Failure      404  {object}  repository.Response
// @Failure      500  {object}  repository.Response
// @Security     ApiKeyAuth
// @Router       /spendings/{id} [delete]
func (h *SpendingHandler) Delete(c fiber.Ctx) error {
	// Get user_id from locals (set by middleware)
	userID, err := services.GetUserIDFromContext(c)
	if err != nil {
		return c.Status(http.StatusUnauthorized).JSON(repository.NewResponse("error", err.Error(), nil))
	}

	spending_id, err := bson.ObjectIDFromHex(c.Params("id"))
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(repository.NewResponse("error", err.Error(), nil))
	}

	spending, err := h.repo.GetByID(c.Context(), spending_id)
	if err != nil || spending == nil {
		return c.Status(http.StatusNotFound).JSON(repository.NewResponse("error", "spending not found", nil))
	}

	// Only the user who created (owner) can delete
	if spending.UserID != userID {
		return c.Status(http.StatusForbidden).JSON(repository.NewResponse("error", "only the owner can delete this spending", nil))
	}

	err = h.repo.Delete(c.Context(), spending_id)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(repository.NewResponse("error", "internal server error", nil))
	}

	// --- Update user balance (add back deleted spending amount) ---
	user, err := h.userRepo.GetByID(c.Context(), userID)
	if err != nil || user == nil {
		return c.Status(http.StatusInternalServerError).JSON(repository.NewResponse("error", "failed to get user for balance update", nil))
	}
	// Balance = balance + deletedAmount
	newBalance := user.Balance + spending.Amount
	if err := h.userRepo.UpdateBalance(c.Context(), userID, newBalance); err != nil {
		return c.Status(http.StatusInternalServerError).JSON(repository.NewResponse("error", "failed to update user balance", nil))
	}
	// --- end user balance update

	return c.Status(http.StatusOK).JSON(repository.NewResponse("success", "spending deleted successfully", nil))
}

// LinkBudget godoc
// @Summary      Link a budget to a spending
// @Description  Links a budget to a specified spending for the authenticated user
// @Tags         spendings
// @Accept       json
// @Produce      json
// @Param        id        path      string true  "Spending ID (hex)"
// @Param        budget_id path      string true  "Budget ID (hex)"
// @Success      200  {object}  repository.Response
// @Failure      400  {object}  repository.Response
// @Failure      401  {object}  repository.Response
// @Failure      403  {object}  repository.Response
// @Failure      404  {object}  repository.Response
// @Failure      500  {object}  repository.Response
// @Security     ApiKeyAuth
// @Router       /api/v1/spendings/{id}/budgets/{budget_id}/link [post]
func (h *SpendingHandler) LinkBudget(c fiber.Ctx) error {
	userID, err := services.GetUserIDFromContext(c)
	if err != nil {
		return c.Status(http.StatusUnauthorized).JSON(repository.NewResponse("error", err.Error(), nil))
	}

	spendingID, err := bson.ObjectIDFromHex(c.Params("id"))
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(repository.NewResponse("error", err.Error(), nil))
	}

	budgetID, err := bson.ObjectIDFromHex(c.Params("budget_id"))
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(repository.NewResponse("error", err.Error(), nil))
	}

	// check first if spending and budget are the same person
	spending, err := h.repo.GetByID(c.Context(), spendingID)
	if err != nil || spending == nil {
		return c.Status(http.StatusInternalServerError).JSON(repository.NewResponse("error", "internal server error", nil))
	}
	if spending.UserID != userID {
		return c.Status(http.StatusForbidden).JSON(repository.NewResponse("error", "only the owner can link this spending", nil))
	}

	budget, err := h.budgetRepo.GetByID(c.Context(), budgetID)
	if err != nil || budget == nil {
		return c.Status(http.StatusInternalServerError).JSON(repository.NewResponse("error", "internal server error", nil))
	}
	if budget.UserID != userID {
		return c.Status(http.StatusForbidden).JSON(repository.NewResponse("error", "only the owner can link this budget", nil))
	}

	// link the budget
	err = h.repo.LinkBudget(c.Context(), spendingID, budgetID)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(repository.NewResponse("error", "internal server error", nil))
	}
	// -- update the budget balance as well
	budget.Amount += spending.Amount
	err = h.budgetRepo.Update(c.Context(), budgetID, budget)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(repository.NewResponse("error", "internal server error", nil))
	}
	// -- end budget balance update

	return c.Status(http.StatusOK).JSON(repository.NewResponse("success", "budget linked successfully", nil))
}

// UnlinkBudget godoc
// @Summary      Unlink a budget from a spending
// @Description  Unlinks a budget from a specified spending for the authenticated user
// @Tags         spendings
// @Accept       json
// @Produce      json
// @Param        id        path      string true  "Spending ID (hex)"
// @Param        budget_id path      string true  "Budget ID (hex)"
// @Success      200  {object}  repository.Response
// @Failure      400  {object}  repository.Response
// @Failure      401  {object}  repository.Response
// @Failure      403  {object}  repository.Response
// @Failure      404  {object}  repository.Response
// @Failure      500  {object}  repository.Response
// @Security     ApiKeyAuth
// @Router       /api/v1/spendings/{id}/budgets/{budget_id}/unlink [post]
func (h *SpendingHandler) UnlinkBudget(c fiber.Ctx) error {
	userID, err := services.GetUserIDFromContext(c)
	if err != nil {
		return c.Status(http.StatusUnauthorized).JSON(repository.NewResponse("error", err.Error(), nil))
	}

	spendingID, err := bson.ObjectIDFromHex(c.Params("id"))
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(repository.NewResponse("error", err.Error(), nil))
	}

	budgetID, err := bson.ObjectIDFromHex(c.Params("budget_id"))
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(repository.NewResponse("error", err.Error(), nil))
	}

	// check first if spending and budget are the same person
	spending, err := h.repo.GetByID(c.Context(), spendingID)
	if err != nil || spending == nil {
		return c.Status(http.StatusInternalServerError).JSON(repository.NewResponse("error", "internal server error", nil))
	}
	if spending.UserID != userID {
		return c.Status(http.StatusForbidden).JSON(repository.NewResponse("error", "only the owner can unlink this spending", nil))
	}

	budget, err := h.budgetRepo.GetByID(c.Context(), budgetID)
	if err != nil || budget == nil {
		return c.Status(http.StatusInternalServerError).JSON(repository.NewResponse("error", "internal server error", nil))
	}
	if budget.UserID != userID {
		return c.Status(http.StatusForbidden).JSON(repository.NewResponse("error", "only the owner can unlink this budget", nil))
	}

	// unlink the budget
	err = h.repo.UnlinkBudget(c.Context(), spendingID, budgetID)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(repository.NewResponse("error", "internal server error", nil))
	}

	// -- update the budget balance as well
	budget.Amount -= spending.Amount
	err = h.budgetRepo.Update(c.Context(), budgetID, budget)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(repository.NewResponse("error", "internal server error", nil))
	}
	// -- end budget balance update

	return c.Status(http.StatusOK).JSON(repository.NewResponse("success", "budget unlinked successfully", nil))
}
