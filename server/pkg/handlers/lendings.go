package handlers

import (
	"errors"
	"slices"
	"time"

	"aslon1213/gift/pkg/repository"
	"aslon1213/gift/services"

	"github.com/gofiber/fiber/v3"
	"go.mongodb.org/mongo-driver/v2/bson"
)

type LendingHandler struct {
	repo *repository.CreditRepository
}

func NewLendingHandler(repo *repository.CreditRepository) *LendingHandler {
	return &LendingHandler{repo: repo}
}

// CreateLendingRequest is the request body for recording a lending. Exactly
// one of ToUserID or ToName must be supplied.
type CreateLendingRequest struct {
	ToUserID       string    `json:"to_user_id"`
	ToName         string    `json:"to_name"`
	Amount         float64   `json:"amount"`
	ResolvedAmount float64   `json:"resolved_amount"`
	Currency       string    `json:"currency"`
	Description    string    `json:"description"`
	Date           time.Time `json:"date"`
}

// UpdateLendingRequest is the request body for updating a lending. Only valid
// for one-OID credits — two-OID credits go through FinanceRequests.
type UpdateLendingRequest struct {
	ToUserID       string    `json:"to_user_id"`
	ToName         string    `json:"to_name"`
	Amount         float64   `json:"amount"`
	ResolvedAmount float64   `json:"resolved_amount"`
	Currency       string    `json:"currency"`
	Description    string    `json:"description"`
	Resolved       bool      `json:"resolved"`
	Date           time.Time `json:"date"`
}

// CreditActionInput is the body shared by repay/take/give/collect helpers.
type CreditActionInput struct {
	Amount      float64 `json:"amount"`
	Description string  `json:"description"`
}

// Create godoc
// @Summary      Create lending
// @Description  Records a new lending for the authenticated user. The borrower
// @Description  is set via to_user_id (registered user) or to_name (free string).
// @Tags         lendings
// @Accept       json
// @Produce      json
// @Param        body body     CreateLendingRequest true "Lending data"
// @Success      201  {object} repository.Response[repository.Credit]
// @Failure      400  {object} repository.Response[repository.Empty]
// @Failure      401  {object} repository.Response[repository.Empty]
// @Failure      500  {object} repository.Response[repository.Empty]
// @Security     ApiKeyAuth
// @Router       /api/v1/lendings [post]
func (h *LendingHandler) Create(c fiber.Ctx) error {
	userID, err := services.GetUserIDFromContext(c)
	if err != nil {
		return repository.Unauthorized(c, "unauthorized")
	}

	var input CreateLendingRequest
	if err := c.Bind().Body(&input); err != nil {
		return repository.BadRequest(c, "invalid request body")
	}

	if input.Amount <= 0 {
		return repository.BadRequest(c, "amount must be greater than 0")
	}
	if input.ResolvedAmount < 0 || input.ResolvedAmount > input.Amount {
		return repository.BadRequest(c, "resolved_amount must be between 0 and amount")
	}

	to, err := resolveCounterparty(input.ToUserID, input.ToName)
	if err != nil {
		return repository.BadRequest(c, err.Error())
	}
	if to.IsOID && to.Oid == userID {
		return repository.BadRequest(c, "cannot lend to yourself")
	}

	if input.Currency == "" {
		input.Currency = "UZS"
	}
	if input.Date.IsZero() {
		input.Date = time.Now()
	}

	credit := &repository.Credit{
		From:           repository.FlexIDFromOID(userID),
		To:             to,
		Amount:         input.Amount,
		ResolvedAmount: input.ResolvedAmount,
		Currency:       input.Currency,
		Description:    input.Description,
		Resolved:       input.ResolvedAmount >= input.Amount,
		Date:           input.Date,
	}
	if err := h.repo.Create(c.Context(), credit); err != nil {
		return repository.Internal(c, "failed to create lending")
	}
	return repository.Created(c, "lending created", credit)
}

// Get godoc
// @Summary      Get lending by ID
// @Description  Returns a lending where the caller is the From party (lender).
// @Tags         lendings
// @Produce      json
// @Param        id path string true "Lending ID (hex)"
// @Success      200 {object} repository.Response[repository.Credit]
// @Failure      400 {object} repository.Response[repository.Empty]
// @Failure      401 {object} repository.Response[repository.Empty]
// @Failure      403 {object} repository.Response[repository.Empty]
// @Failure      404 {object} repository.Response[repository.Empty]
// @Security     ApiKeyAuth
// @Router       /api/v1/lendings/{id} [get]
func (h *LendingHandler) Get(c fiber.Ctx) error {
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
		return repository.NotFound(c, "lending not found")
	}
	if !credit.From.IsOID || credit.From.Oid != userID {
		return repository.Forbidden(c, "forbidden")
	}
	return repository.OK(c, "lending fetched", credit)
}

// List godoc
// @Summary      List lendings
// @Description  Returns lendings where the authenticated user is the From party.
// @Tags         lendings
// @Produce      json
// @Success      200 {object} repository.Response[[]repository.Credit]
// @Failure      401 {object} repository.Response[repository.Empty]
// @Failure      500 {object} repository.Response[repository.Empty]
// @Security     ApiKeyAuth
// @Router       /api/v1/lendings [get]
func (h *LendingHandler) List(c fiber.Ctx) error {
	userID, err := services.GetUserIDFromContext(c)
	if err != nil {
		return repository.Unauthorized(c, "unauthorized")
	}
	credits, err := h.repo.Query(c.Context(), bson.M{"from.oid": userID, "from.is_oid": true})
	if err != nil {
		return repository.Internal(c, "failed to list lendings")
	}
	slices.SortFunc(credits, func(a, b *repository.Credit) int {
		return b.Date.Compare(a.Date)
	})
	return repository.OK(c, "lendings listed", credits)
}

// Update godoc
// @Summary      Update lending (one-OID credits only)
// @Description  Updates a lending where the borrower is a free-form name. For
// @Description  lendings between two registered users, use FinanceRequests.
// @Tags         lendings
// @Accept       json
// @Produce      json
// @Param        id   path     string               true "Lending ID (hex)"
// @Param        body body     UpdateLendingRequest true "Lending fields"
// @Success      200  {object} repository.Response[repository.Credit]
// @Failure      400  {object} repository.Response[repository.Empty]
// @Failure      401  {object} repository.Response[repository.Empty]
// @Failure      403  {object} repository.Response[repository.Empty]
// @Failure      404  {object} repository.Response[repository.Empty]
// @Failure      409  {object} repository.Response[repository.Empty]
// @Failure      500  {object} repository.Response[repository.Empty]
// @Security     ApiKeyAuth
// @Router       /api/v1/lendings/{id} [put]
func (h *LendingHandler) Update(c fiber.Ctx) error {
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
		return repository.NotFound(c, "lending not found")
	}
	if !credit.From.IsOID || credit.From.Oid != userID {
		return repository.Forbidden(c, "forbidden")
	}
	if credit.IsTwoParty() {
		return repository.Conflict(c, "two-party credits must be modified through finance requests")
	}

	var input UpdateLendingRequest
	if err := c.Bind().Body(&input); err != nil {
		return repository.BadRequest(c, "invalid request body")
	}

	if input.ToUserID != "" || input.ToName != "" {
		to, err := resolveCounterparty(input.ToUserID, input.ToName)
		if err != nil {
			return repository.BadRequest(c, err.Error())
		}
		if to.IsOID && to.Oid == userID {
			return repository.BadRequest(c, "cannot lend to yourself")
		}
		credit.To = to
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
		return repository.Internal(c, "failed to update lending")
	}
	return repository.OK(c, "lending updated", credit)
}

// Delete godoc
// @Summary      Delete lending
// @Description  Deletes a one-OID lending, or a two-OID lending once it is
// @Description  fully resolved.
// @Tags         lendings
// @Produce      json
// @Param        id  path     string true "Lending ID (hex)"
// @Success      200 {object} repository.Response[repository.Empty]
// @Failure      400 {object} repository.Response[repository.Empty]
// @Failure      401 {object} repository.Response[repository.Empty]
// @Failure      403 {object} repository.Response[repository.Empty]
// @Failure      404 {object} repository.Response[repository.Empty]
// @Failure      409 {object} repository.Response[repository.Empty]
// @Failure      500 {object} repository.Response[repository.Empty]
// @Security     ApiKeyAuth
// @Router       /api/v1/lendings/{id} [delete]
func (h *LendingHandler) Delete(c fiber.Ctx) error {
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
		return repository.NotFound(c, "lending not found")
	}
	if !credit.From.IsOID || credit.From.Oid != userID {
		return repository.Forbidden(c, "forbidden")
	}
	if credit.IsTwoParty() && !credit.Resolved {
		return repository.Conflict(c, "two-party credits can only be deleted once resolved")
	}
	if err := h.repo.Delete(c.Context(), oid); err != nil {
		return repository.Internal(c, "failed to delete lending")
	}
	return repository.Ack(c, "lending deleted")
}

// Give godoc
// @Summary      Give more on a lending
// @Description  Records that the lender extended more money — increases amount.
// @Description  For two-party credits this opens a FinanceRequest the borrower
// @Description  must approve. For one-OID credits the change is applied immediately.
// @Tags         lendings
// @Accept       json
// @Produce      json
// @Param        id   path     string             true "Lending ID (hex)"
// @Param        body body     CreditActionInput  true "Give data"
// @Success      200  {object} repository.Response[repository.Credit]
// @Failure      400  {object} repository.Response[repository.Empty]
// @Failure      401  {object} repository.Response[repository.Empty]
// @Failure      403  {object} repository.Response[repository.Empty]
// @Failure      404  {object} repository.Response[repository.Empty]
// @Failure      500  {object} repository.Response[repository.Empty]
// @Security     ApiKeyAuth
// @Router       /api/v1/lendings/{id}/give [post]
func (h *LendingHandler) Give(c fiber.Ctx) error {
	return runCreditAction(c, h.repo, lenderSide, repository.FinanceRequestIncreaseAmount, "give recorded")
}

// Collect godoc
// @Summary      Collect on a lending
// @Description  Records that the lender received money back — increases resolved_amount.
// @Description  For two-party credits this opens a FinanceRequest the borrower
// @Description  must approve. For one-OID credits the change is applied immediately.
// @Tags         lendings
// @Accept       json
// @Produce      json
// @Param        id   path     string             true "Lending ID (hex)"
// @Param        body body     CreditActionInput  true "Collect data"
// @Success      200  {object} repository.Response[repository.Credit]
// @Failure      400  {object} repository.Response[repository.Empty]
// @Failure      401  {object} repository.Response[repository.Empty]
// @Failure      403  {object} repository.Response[repository.Empty]
// @Failure      404  {object} repository.Response[repository.Empty]
// @Failure      500  {object} repository.Response[repository.Empty]
// @Security     ApiKeyAuth
// @Router       /api/v1/lendings/{id}/collect [post]
func (h *LendingHandler) Collect(c fiber.Ctx) error {
	return runCreditAction(c, h.repo, lenderSide, repository.FinanceRequestIncreaseResolvedAmount, "collect recorded")
}

// resolveCounterparty turns request fields into a FlexID. The caller picks
// either a registered user (hex OID) or a free-form name string.
func resolveCounterparty(userIDHex, name string) (repository.FlexID, error) {
	if userIDHex != "" {
		oid, err := bson.ObjectIDFromHex(userIDHex)
		if err != nil {
			return repository.FlexID{}, errors.New("invalid counterparty user id")
		}
		return repository.FlexIDFromOID(oid), nil
	}
	if name != "" {
		return repository.FlexIDFromString(name), nil
	}
	return repository.FlexID{}, errors.New("counterparty user id or name is required")
}

// creditSide identifies which leg of a credit a helper acts on. The repay/take
// helpers operate as the borrower (To); give/collect operate as the lender (From).
type creditSide int

const (
	borrowerSide creditSide = iota
	lenderSide
)

// runCreditAction is shared by repay/take/give/collect: it loads the credit,
// authorizes the caller as the appropriate side, then either creates a
// FinanceRequest (two-party) or applies the delta directly (one-OID).
func runCreditAction(
	c fiber.Ctx,
	repo *repository.CreditRepository,
	side creditSide,
	reqType repository.FinanceRequestType,
	successMsg string,
) error {
	userID, err := services.GetUserIDFromContext(c)
	if err != nil {
		return repository.Unauthorized(c, "unauthorized")
	}
	oid, err := bson.ObjectIDFromHex(c.Params("id"))
	if err != nil {
		return repository.BadRequest(c, "invalid id")
	}

	var input CreditActionInput
	if err := c.Bind().Body(&input); err != nil {
		return repository.BadRequest(c, "invalid request body")
	}
	if input.Amount <= 0 {
		return repository.BadRequest(c, "amount must be greater than 0")
	}

	credit, err := repo.GetByID(c.Context(), oid)
	if err != nil || credit == nil {
		return repository.NotFound(c, "credit not found")
	}

	switch side {
	case borrowerSide:
		if !credit.To.IsOID || credit.To.Oid != userID {
			return repository.Forbidden(c, "forbidden")
		}
	case lenderSide:
		if !credit.From.IsOID || credit.From.Oid != userID {
			return repository.Forbidden(c, "forbidden")
		}
	}

	if credit.IsTwoParty() {
		now := time.Now()
		credit.FinanceRequests = append(credit.FinanceRequests, repository.FinanceRequest{
			ID:          bson.NewObjectID(),
			Type:        reqType,
			Amount:      input.Amount,
			Description: input.Description,
			RequestedBy: userID,
			Status:      repository.FinanceRequestPending,
			CreatedAt:   now,
			UpdatedAt:   now,
		})
		if err := repo.Update(c.Context(), oid, credit); err != nil {
			return repository.Internal(c, "failed to open finance request")
		}
		return repository.Created(c, "finance request opened — awaiting counterparty approval", credit)
	}

	if err := applyDelta(credit, reqType, input.Amount); err != nil {
		return repository.BadRequest(c, err.Error())
	}
	if err := repo.Update(c.Context(), oid, credit); err != nil {
		return repository.Internal(c, "failed to update credit")
	}
	return repository.OK(c, successMsg, credit)
}

// applyDelta mutates credit fields per the request type. Returns an error
// if the resulting state would be invalid (negative or out-of-range).
func applyDelta(credit *repository.Credit, t repository.FinanceRequestType, amount float64) error {
	switch t {
	case repository.FinanceRequestIncreaseAmount:
		credit.Amount += amount
	case repository.FinanceRequestDecreaseAmount:
		if amount > credit.Amount {
			return errors.New("decrease exceeds current amount")
		}
		credit.Amount -= amount
		if credit.ResolvedAmount > credit.Amount {
			credit.ResolvedAmount = credit.Amount
		}
	case repository.FinanceRequestIncreaseResolvedAmount:
		next := credit.ResolvedAmount + amount
		if next > credit.Amount {
			return errors.New("repayment exceeds outstanding amount")
		}
		credit.ResolvedAmount = next
	case repository.FinanceRequestDecreaseResolvedAmount:
		if amount > credit.ResolvedAmount {
			return errors.New("decrease exceeds resolved amount")
		}
		credit.ResolvedAmount -= amount
	default:
		return errors.New("unknown finance request type")
	}
	credit.Resolved = credit.Amount > 0 && credit.ResolvedAmount >= credit.Amount
	return nil
}
