package handlers

import (
	"aslon1213/gift/pkg/repository"
	"aslon1213/gift/services"
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
// @Param        user_id    query string false "User ID (hex)"
// @Param        group_id   query string false "Group ID (hex)"
// @Param        category   query string false "Category"
// @Param        start_date query string false "Start date (RFC3339)"
// @Param        end_date   query string false "End date (RFC3339)"
// @Param        limit      query int    false "Limit"
// @Param        offset     query int    false "Offset"
// @Success      200 {object} repository.Response[[]repository.Spending]
// @Failure      500 {object} repository.Response[repository.Empty]
// @Router       /spendings [get]
func (h *SpendingHandler) Query(c fiber.Ctx) error {
	query := bson.M{}
	if userID, err := bson.ObjectIDFromHex(c.Query("user_id")); err == nil {
		query["user_id"] = userID
	}
	if groupID, err := bson.ObjectIDFromHex(c.Query("group_id")); err == nil {
		query["group_id"] = groupID
	}
	if category := c.Query("category"); category != "" {
		query["category"] = category
	}
	if startDate, err := time.Parse(time.RFC3339, c.Query("start_date")); err == nil {
		query["date"] = bson.M{"$gte": startDate}
	}
	if endDate, err := time.Parse(time.RFC3339, c.Query("end_date")); err == nil {
		query["date"] = bson.M{"$lte": endDate}
	}
	if limit, err := strconv.Atoi(c.Query("limit")); err == nil {
		query["limit"] = limit
	}
	if offset, err := strconv.Atoi(c.Query("offset")); err == nil {
		query["offset"] = offset
	}

	spendings, err := h.repo.Query(c.Context(), query)
	if err != nil {
		return repository.Internal(c, "internal server error")
	}

	// sort based on date
	slices.SortFunc(spendings, func(a, b *repository.Spending) int {
		return b.Date.Compare(a.Date)
	})

	return repository.OK(c, "spendings fetched successfully", spendings)
}

// GetByID godoc
// @Summary      Get spending by ID
// @Description  Get a spending by its ID
// @Tags         spendings
// @Accept       json
// @Produce      json
// @Param        id  path     string true "Spending ID (hex)"
// @Success      200 {object} repository.Response[repository.Spending]
// @Failure      400 {object} repository.Response[repository.Empty]
// @Failure      500 {object} repository.Response[repository.Empty]
// @Router       /spendings/{id} [get]
func (h *SpendingHandler) GetByID(c fiber.Ctx) error {
	spendingID, err := bson.ObjectIDFromHex(c.Params("id"))
	if err != nil {
		return repository.BadRequest(c, err.Error())
	}

	spending, err := h.repo.GetByID(c.Context(), spendingID)
	if err != nil {
		return repository.Internal(c, "internal server error")
	}
	return repository.OK(c, "spending fetched successfully", spending)
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
// @Success      201  {object} repository.Response[repository.Spending]
// @Failure      400  {object} repository.Response[repository.Empty]
// @Failure      403  {object} repository.Response[repository.Empty]
// @Failure      404  {object} repository.Response[repository.Empty]
// @Failure      500  {object} repository.Response[repository.Empty]
// @Security     ApiKeyAuth
// @Router       /spendings [post]
func (h *SpendingHandler) Create(c fiber.Ctx) error {
	userID, err := services.GetUserIDFromContext(c)
	if err != nil {
		return repository.Unauthorized(c, err.Error())
	}

	var input CreateSpendingRequest
	if err := c.Bind().Body(&input); err != nil {
		return repository.BadRequest(c, "invalid request body")
	}
	if input.Amount <= 0 {
		return repository.BadRequest(c, "amount must be greater than 0")
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
		return repository.BadRequest(c, "group_id is required")
	}
	group, err := h.groupRepo.GetByID(c.Context(), input.GroupID)
	if err != nil {
		return repository.Internal(c, "internal server error")
	}
	if group == nil {
		return repository.NotFound(c, "group not found")
	}
	if !slices.Contains(group.MemberIDs, userID) {
		return repository.Forbidden(c, "user is not a member of the group")
	}

	spending := &repository.Spending{
		UserID:      userID,
		GroupID:     input.GroupID,
		Amount:      input.Amount,
		Currency:    input.Currency,
		Category:    input.Category,
		Description: input.Description,
		Date:        input.Date,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	if err := h.repo.Create(c.Context(), spending); err != nil {
		return repository.Internal(c, "internal server error")
	}

	user, err := h.userRepo.GetByID(c.Context(), userID)
	if err != nil || user == nil {
		return repository.Internal(c, "failed to get user for balance update")
	}
	newBalance := user.Balance - input.Amount
	if err := h.userRepo.UpdateBalance(c.Context(), userID, newBalance); err != nil {
		return repository.Internal(c, "failed to update user balance")
	}

	return repository.Created(c, "spending created successfully", spending)
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
// @Param        id   path     string                 true "Spending ID (hex)"
// @Param        body body     UpdateSpendingRequest  true "Spending fields to update"
// @Success      200  {object} repository.Response[repository.Spending]
// @Failure      400  {object} repository.Response[repository.Empty]
// @Failure      401  {object} repository.Response[repository.Empty]
// @Failure      403  {object} repository.Response[repository.Empty]
// @Failure      404  {object} repository.Response[repository.Empty]
// @Failure      500  {object} repository.Response[repository.Empty]
// @Security     ApiKeyAuth
// @Router       /spendings/{id} [put]
func (h *SpendingHandler) Update(c fiber.Ctx) error {
	userID, err := services.GetUserIDFromContext(c)
	if err != nil {
		return repository.Unauthorized(c, err.Error())
	}

	var input UpdateSpendingRequest
	if err := c.Bind().Body(&input); err != nil {
		return repository.BadRequest(c, "invalid request body")
	}

	spendingID, err := bson.ObjectIDFromHex(c.Params("id"))
	if err != nil {
		return repository.BadRequest(c, err.Error())
	}

	spending, err := h.repo.GetByID(c.Context(), spendingID)
	if err != nil || spending == nil {
		return repository.NotFound(c, "spending not found")
	}

	if spending.UserID != userID {
		return repository.Forbidden(c, "only the owner can edit this spending")
	}

	oldAmount := spending.Amount

	if input.Amount > 0 {
		spending.Amount = input.Amount
	} else if input.Amount != 0 {
		return repository.BadRequest(c, "amount must be greater than 0")
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

	if err := h.repo.Update(c.Context(), spendingID, spending); err != nil {
		return repository.Internal(c, "internal server error")
	}

	user, err := h.userRepo.GetByID(c.Context(), userID)
	if err != nil || user == nil {
		return repository.Internal(c, "failed to get user for balance update")
	}
	newBalance := user.Balance + oldAmount - spending.Amount
	if err := h.userRepo.UpdateBalance(c.Context(), userID, newBalance); err != nil {
		return repository.Internal(c, "failed to update user balance")
	}

	return repository.OK(c, "spending updated successfully", spending)
}

// Delete godoc
// @Summary      Delete spending
// @Description  Delete an existing spending (owner only)
// @Tags         spendings
// @Accept       json
// @Produce      json
// @Param        id  path     string true "Spending ID (hex)"
// @Success      200 {object} repository.Response[repository.Empty]
// @Failure      400 {object} repository.Response[repository.Empty]
// @Failure      401 {object} repository.Response[repository.Empty]
// @Failure      403 {object} repository.Response[repository.Empty]
// @Failure      404 {object} repository.Response[repository.Empty]
// @Failure      500 {object} repository.Response[repository.Empty]
// @Security     ApiKeyAuth
// @Router       /spendings/{id} [delete]
func (h *SpendingHandler) Delete(c fiber.Ctx) error {
	userID, err := services.GetUserIDFromContext(c)
	if err != nil {
		return repository.Unauthorized(c, err.Error())
	}

	spendingID, err := bson.ObjectIDFromHex(c.Params("id"))
	if err != nil {
		return repository.BadRequest(c, err.Error())
	}

	spending, err := h.repo.GetByID(c.Context(), spendingID)
	if err != nil || spending == nil {
		return repository.NotFound(c, "spending not found")
	}

	if spending.UserID != userID {
		return repository.Forbidden(c, "only the owner can delete this spending")
	}

	if err := h.repo.Delete(c.Context(), spendingID); err != nil {
		return repository.Internal(c, "internal server error")
	}

	user, err := h.userRepo.GetByID(c.Context(), userID)
	if err != nil || user == nil {
		return repository.Internal(c, "failed to get user for balance update")
	}
	newBalance := user.Balance + spending.Amount
	if err := h.userRepo.UpdateBalance(c.Context(), userID, newBalance); err != nil {
		return repository.Internal(c, "failed to update user balance")
	}

	return repository.Ack(c, "spending deleted successfully")
}

// LinkBudget godoc
// @Summary      Link a budget to a spending
// @Description  Links a budget to a specified spending for the authenticated user
// @Tags         spendings
// @Accept       json
// @Produce      json
// @Param        id        path string true "Spending ID (hex)"
// @Param        budget_id path string true "Budget ID (hex)"
// @Success      200 {object} repository.Response[repository.Empty]
// @Failure      400 {object} repository.Response[repository.Empty]
// @Failure      401 {object} repository.Response[repository.Empty]
// @Failure      403 {object} repository.Response[repository.Empty]
// @Failure      404 {object} repository.Response[repository.Empty]
// @Failure      500 {object} repository.Response[repository.Empty]
// @Security     ApiKeyAuth
// @Router       /api/v1/spendings/{id}/budgets/{budget_id}/link [post]
func (h *SpendingHandler) LinkBudget(c fiber.Ctx) error {
	return h.toggleBudgetLink(c, true)
}

// UnlinkBudget godoc
// @Summary      Unlink a budget from a spending
// @Description  Unlinks a budget from a specified spending for the authenticated user
// @Tags         spendings
// @Accept       json
// @Produce      json
// @Param        id        path string true "Spending ID (hex)"
// @Param        budget_id path string true "Budget ID (hex)"
// @Success      200 {object} repository.Response[repository.Empty]
// @Failure      400 {object} repository.Response[repository.Empty]
// @Failure      401 {object} repository.Response[repository.Empty]
// @Failure      403 {object} repository.Response[repository.Empty]
// @Failure      404 {object} repository.Response[repository.Empty]
// @Failure      500 {object} repository.Response[repository.Empty]
// @Security     ApiKeyAuth
// @Router       /api/v1/spendings/{id}/budgets/{budget_id}/unlink [post]
func (h *SpendingHandler) UnlinkBudget(c fiber.Ctx) error {
	return h.toggleBudgetLink(c, false)
}

func (h *SpendingHandler) toggleBudgetLink(c fiber.Ctx, link bool) error {
	verb := "link"
	if !link {
		verb = "unlink"
	}

	userID, err := services.GetUserIDFromContext(c)
	if err != nil {
		return repository.Unauthorized(c, err.Error())
	}

	spendingID, err := bson.ObjectIDFromHex(c.Params("id"))
	if err != nil {
		return repository.BadRequest(c, err.Error())
	}

	budgetID, err := bson.ObjectIDFromHex(c.Params("budget_id"))
	if err != nil {
		return repository.BadRequest(c, err.Error())
	}

	spending, err := h.repo.GetByID(c.Context(), spendingID)
	if err != nil || spending == nil {
		return repository.Internal(c, "internal server error")
	}
	if spending.UserID != userID {
		return repository.Forbidden(c, "only the owner can "+verb+" this spending")
	}

	budget, err := h.budgetRepo.GetByID(c.Context(), budgetID)
	if err != nil || budget == nil {
		return repository.Internal(c, "internal server error")
	}
	if budget.UserID != userID {
		return repository.Forbidden(c, "only the owner can "+verb+" this budget")
	}

	if link {
		if err := h.repo.LinkBudget(c.Context(), spendingID, budgetID); err != nil {
			return repository.Internal(c, "internal server error")
		}
		budget.Amount += spending.Amount
	} else {
		if err := h.repo.UnlinkBudget(c.Context(), spendingID, budgetID); err != nil {
			return repository.Internal(c, "internal server error")
		}
		budget.Amount -= spending.Amount
	}

	if err := h.budgetRepo.Update(c.Context(), budgetID, budget); err != nil {
		return repository.Internal(c, "internal server error")
	}
	return repository.Ack(c, "budget "+verb+"ed successfully")
}
