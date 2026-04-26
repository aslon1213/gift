package handlers

import (
	"aslon1213/gift/pkg/repository"
	"aslon1213/gift/services"
	"slices"
	"time"

	"github.com/gofiber/fiber/v3"
	"go.mongodb.org/mongo-driver/v2/bson"
)

type BorrowingHandler struct {
	repo *repository.CreditRepository
}

func NewBorrowingHandler(repo *repository.CreditRepository) *BorrowingHandler {
	return &BorrowingHandler{repo: repo}
}

// CreateBorrowingRequest is the request body for recording a borrowing.
// Exactly one of FromUserID (registered user lender) or FromName (free-form
// name for an unregistered counterparty) must be provided.
type CreateBorrowingRequest struct {
	FromUserID     string    `json:"from_user_id"`
	FromName       string    `json:"from_name"`
	Amount         float64   `json:"amount"`
	ResolvedAmount float64   `json:"resolved_amount"`
	Currency       string    `json:"currency"`
	Description    string    `json:"description"`
	Date           time.Time `json:"date"`
}

// UpdateBorrowingRequest is the request body for updating a borrowing.
// Only valid for one-OID credits — two-OID credits go through FinanceRequests.
type UpdateBorrowingRequest struct {
	FromUserID     string    `json:"from_user_id"`
	FromName       string    `json:"from_name"`
	Amount         float64   `json:"amount"`
	ResolvedAmount float64   `json:"resolved_amount"`
	Currency       string    `json:"currency"`
	Description    string    `json:"description"`
	Resolved       bool      `json:"resolved"`
	Date           time.Time `json:"date"`
}

// Create godoc
// @Summary      Create borrowing
// @Description  Records a new borrowing for the authenticated user. The lender
// @Description  is set via from_user_id (registered user) or from_name (free string).
// @Tags         borrowings
// @Accept       json
// @Produce      json
// @Param        body body     CreateBorrowingRequest true "Borrowing data"
// @Success      201  {object} repository.Response[repository.Credit]
// @Failure      400  {object} repository.Response[repository.Empty]
// @Failure      401  {object} repository.Response[repository.Empty]
// @Failure      500  {object} repository.Response[repository.Empty]
// @Security     ApiKeyAuth
// @Router       /api/v1/borrowings [post]
func (h *BorrowingHandler) Create(c fiber.Ctx) error {
	userID, err := services.GetUserIDFromContext(c)
	if err != nil {
		return repository.Unauthorized(c, "unauthorized")
	}

	var input CreateBorrowingRequest
	if err := c.Bind().Body(&input); err != nil {
		return repository.BadRequest(c, "invalid request body")
	}

	if input.Amount <= 0 {
		return repository.BadRequest(c, "amount must be greater than 0")
	}
	if input.ResolvedAmount < 0 || input.ResolvedAmount > input.Amount {
		return repository.BadRequest(c, "resolved_amount must be between 0 and amount")
	}

	from, err := resolveCounterparty(input.FromUserID, input.FromName)
	if err != nil {
		return repository.BadRequest(c, err.Error())
	}
	if from.IsOID && from.Oid == userID {
		return repository.BadRequest(c, "cannot borrow from yourself")
	}

	if input.Currency == "" {
		input.Currency = "UZS"
	}
	if input.Date.IsZero() {
		input.Date = time.Now()
	}

	credit := &repository.Credit{
		From:           from,
		To:             repository.FlexIDFromOID(userID),
		Amount:         input.Amount,
		ResolvedAmount: input.ResolvedAmount,
		Currency:       input.Currency,
		Description:    input.Description,
		Resolved:       input.ResolvedAmount >= input.Amount,
		Date:           input.Date,
	}
	if err := h.repo.Create(c.Context(), credit); err != nil {
		return repository.Internal(c, "failed to create borrowing")
	}
	return repository.Created(c, "borrowing created", credit)
}

// Get godoc
// @Summary      Get borrowing by ID
// @Description  Returns a borrowing where the caller is the To party (borrower).
// @Tags         borrowings
// @Produce      json
// @Param        id path string true "Borrowing ID (hex)"
// @Success      200 {object} repository.Response[repository.Credit]
// @Failure      400 {object} repository.Response[repository.Empty]
// @Failure      401 {object} repository.Response[repository.Empty]
// @Failure      403 {object} repository.Response[repository.Empty]
// @Failure      404 {object} repository.Response[repository.Empty]
// @Security     ApiKeyAuth
// @Router       /api/v1/borrowings/{id} [get]
func (h *BorrowingHandler) Get(c fiber.Ctx) error {
	userID, err := services.GetUserIDFromContext(c)
	if err != nil {
		return repository.Unauthorized(c, "unauthorized")
	}
	oid, err := bson.ObjectIDFromHex(c.Params("id"))
	if err != nil {
		return repository.BadRequest(c, "invalid id")
	}

	credit, err := h.repo.GetByID(c.Context(), oid)
	if err != nil || credit == nil {
		return repository.NotFound(c, "borrowing not found")
	}
	if !credit.To.IsOID || credit.To.Oid != userID {
		return repository.Forbidden(c, "forbidden")
	}
	return repository.OK(c, "borrowing fetched", credit)
}

// List godoc
// @Summary      List borrowings
// @Description  Returns borrowings where the authenticated user is the To party.
// @Tags         borrowings
// @Produce      json
// @Success      200 {object} repository.Response[[]repository.Credit]
// @Failure      401 {object} repository.Response[repository.Empty]
// @Failure      500 {object} repository.Response[repository.Empty]
// @Security     ApiKeyAuth
// @Router       /api/v1/borrowings [get]
func (h *BorrowingHandler) List(c fiber.Ctx) error {
	userID, err := services.GetUserIDFromContext(c)
	if err != nil {
		return repository.Unauthorized(c, "unauthorized")
	}
	credits, err := h.repo.Query(c.Context(), bson.M{"to.oid": userID, "to.is_oid": true})
	if err != nil {
		return repository.Internal(c, "failed to list borrowings")
	}
	slices.SortFunc(credits, func(a, b *repository.Credit) int {
		return b.Date.Compare(a.Date)
	})
	return repository.OK(c, "borrowings listed", credits)
}

// Update godoc
// @Summary      Update borrowing (one-OID credits only)
// @Description  Updates a borrowing where the lender is a free-form name. For
// @Description  borrowings between two registered users, use FinanceRequests.
// @Tags         borrowings
// @Accept       json
// @Produce      json
// @Param        id   path     string                 true "Borrowing ID (hex)"
// @Param        body body     UpdateBorrowingRequest true "Borrowing fields"
// @Success      200  {object} repository.Response[repository.Credit]
// @Failure      400  {object} repository.Response[repository.Empty]
// @Failure      401  {object} repository.Response[repository.Empty]
// @Failure      403  {object} repository.Response[repository.Empty]
// @Failure      404  {object} repository.Response[repository.Empty]
// @Failure      409  {object} repository.Response[repository.Empty]
// @Failure      500  {object} repository.Response[repository.Empty]
// @Security     ApiKeyAuth
// @Router       /api/v1/borrowings/{id} [put]
func (h *BorrowingHandler) Update(c fiber.Ctx) error {
	userID, err := services.GetUserIDFromContext(c)
	if err != nil {
		return repository.Unauthorized(c, "unauthorized")
	}
	oid, err := bson.ObjectIDFromHex(c.Params("id"))
	if err != nil {
		return repository.BadRequest(c, "invalid id")
	}
	credit, err := h.repo.GetByID(c.Context(), oid)
	if err != nil || credit == nil {
		return repository.NotFound(c, "borrowing not found")
	}
	if !credit.To.IsOID || credit.To.Oid != userID {
		return repository.Forbidden(c, "forbidden")
	}
	if credit.IsTwoParty() {
		return repository.Conflict(c, "two-party credits must be modified through finance requests")
	}

	var input UpdateBorrowingRequest
	if err := c.Bind().Body(&input); err != nil {
		return repository.BadRequest(c, "invalid request body")
	}

	if input.FromUserID != "" || input.FromName != "" {
		from, err := resolveCounterparty(input.FromUserID, input.FromName)
		if err != nil {
			return repository.BadRequest(c, err.Error())
		}
		if from.IsOID && from.Oid == userID {
			return repository.BadRequest(c, "cannot borrow from yourself")
		}
		credit.From = from
	}
	if input.Amount > 0 {
		credit.Amount = input.Amount
	}
	if input.ResolvedAmount < 0 || input.ResolvedAmount > credit.Amount {
		return repository.BadRequest(c, "resolved_amount must be between 0 and amount")
	}
	credit.ResolvedAmount = input.ResolvedAmount
	if input.Currency != "" {
		credit.Currency = input.Currency
	}
	if input.Description != "" {
		credit.Description = input.Description
	}
	if !input.Date.IsZero() {
		credit.Date = input.Date
	}
	credit.Resolved = input.Resolved || credit.ResolvedAmount >= credit.Amount

	if err := h.repo.Update(c.Context(), oid, credit); err != nil {
		return repository.Internal(c, "failed to update borrowing")
	}
	return repository.OK(c, "borrowing updated", credit)
}

// Delete godoc
// @Summary      Delete borrowing
// @Description  Deletes a one-OID borrowing, or a two-OID borrowing once it is
// @Description  fully resolved.
// @Tags         borrowings
// @Produce      json
// @Param        id  path     string true "Borrowing ID (hex)"
// @Success      200 {object} repository.Response[repository.Empty]
// @Failure      400 {object} repository.Response[repository.Empty]
// @Failure      401 {object} repository.Response[repository.Empty]
// @Failure      403 {object} repository.Response[repository.Empty]
// @Failure      404 {object} repository.Response[repository.Empty]
// @Failure      409 {object} repository.Response[repository.Empty]
// @Failure      500 {object} repository.Response[repository.Empty]
// @Security     ApiKeyAuth
// @Router       /api/v1/borrowings/{id} [delete]
func (h *BorrowingHandler) Delete(c fiber.Ctx) error {
	userID, err := services.GetUserIDFromContext(c)
	if err != nil {
		return repository.Unauthorized(c, "unauthorized")
	}
	oid, err := bson.ObjectIDFromHex(c.Params("id"))
	if err != nil {
		return repository.BadRequest(c, "invalid id")
	}
	credit, err := h.repo.GetByID(c.Context(), oid)
	if err != nil || credit == nil {
		return repository.NotFound(c, "borrowing not found")
	}
	if !credit.To.IsOID || credit.To.Oid != userID {
		return repository.Forbidden(c, "forbidden")
	}
	if credit.IsTwoParty() && !credit.Resolved {
		return repository.Conflict(c, "two-party credits can only be deleted once resolved")
	}
	if err := h.repo.Delete(c.Context(), oid); err != nil {
		return repository.Internal(c, "failed to delete borrowing")
	}
	return repository.Ack(c, "borrowing deleted")
}

// Repay godoc
// @Summary      Repay borrowing
// @Description  Records a repayment on a borrowing — increases resolved_amount.
// @Description  For two-party credits this opens a FinanceRequest the lender must
// @Description  approve. For one-OID credits the change is applied immediately.
// @Tags         borrowings
// @Accept       json
// @Produce      json
// @Param        id   path     string             true "Borrowing ID (hex)"
// @Param        body body     CreditActionInput  true "Repayment data"
// @Success      200  {object} repository.Response[repository.Credit]
// @Failure      400  {object} repository.Response[repository.Empty]
// @Failure      401  {object} repository.Response[repository.Empty]
// @Failure      403  {object} repository.Response[repository.Empty]
// @Failure      404  {object} repository.Response[repository.Empty]
// @Failure      500  {object} repository.Response[repository.Empty]
// @Security     ApiKeyAuth
// @Router       /api/v1/borrowings/{id}/repay [post]
func (h *BorrowingHandler) Repay(c fiber.Ctx) error {
	return runCreditAction(c, h.repo, borrowerSide, repository.FinanceRequestIncreaseResolvedAmount, "repayment recorded")
}

// Take godoc
// @Summary      Take more on a borrowing
// @Description  Records that the borrower received more money — increases amount.
// @Description  For two-party credits this opens a FinanceRequest the lender must
// @Description  approve. For one-OID credits the change is applied immediately.
// @Tags         borrowings
// @Accept       json
// @Produce      json
// @Param        id   path     string             true "Borrowing ID (hex)"
// @Param        body body     CreditActionInput  true "Take data"
// @Success      200  {object} repository.Response[repository.Credit]
// @Failure      400  {object} repository.Response[repository.Empty]
// @Failure      401  {object} repository.Response[repository.Empty]
// @Failure      403  {object} repository.Response[repository.Empty]
// @Failure      404  {object} repository.Response[repository.Empty]
// @Failure      500  {object} repository.Response[repository.Empty]
// @Security     ApiKeyAuth
// @Router       /api/v1/borrowings/{id}/take [post]
func (h *BorrowingHandler) Take(c fiber.Ctx) error {
	return runCreditAction(c, h.repo, borrowerSide, repository.FinanceRequestIncreaseAmount, "take recorded")
}
