package handlers

import (
	"time"

	"aslon1213/gift/pkg/repository"
	"aslon1213/gift/services"

	"github.com/gofiber/fiber/v3"
	"go.mongodb.org/mongo-driver/v2/bson"
)

type FinanceRequestHandler struct {
	repo *repository.CreditRepository
}

func NewFinanceRequestHandler(repo *repository.CreditRepository) *FinanceRequestHandler {
	return &FinanceRequestHandler{repo: repo}
}

// Approve godoc
// @Summary      Approve a finance request
// @Description  Counterparty approves a pending FinanceRequest. The requester
// @Description  cannot approve their own request — only the other party can.
// @Description  On approval, the requested delta is applied to the credit.
// @Tags         finance-requests
// @Produce      json
// @Param        id     path     string true "Credit ID (hex)"
// @Param        req_id path     string true "FinanceRequest ID (hex)"
// @Success      200    {object} repository.Response[repository.Credit]
// @Failure      400    {object} repository.Response[repository.Empty]
// @Failure      401    {object} repository.Response[repository.Empty]
// @Failure      403    {object} repository.Response[repository.Empty]
// @Failure      404    {object} repository.Response[repository.Empty]
// @Failure      409    {object} repository.Response[repository.Empty]
// @Failure      500    {object} repository.Response[repository.Empty]
// @Security     ApiKeyAuth
// @Router       /api/v1/borrowings/{id}/requests/{req_id}/approve [post]
// @Router       /api/v1/lendings/{id}/requests/{req_id}/approve [post]
func (h *FinanceRequestHandler) Approve(c fiber.Ctx) error {
	return h.decide(c, true)
}

// Reject godoc
// @Summary      Reject a finance request
// @Description  Counterparty rejects a pending FinanceRequest — no change is
// @Description  applied to the credit; the request is marked rejected.
// @Tags         finance-requests
// @Produce      json
// @Param        id     path     string true "Credit ID (hex)"
// @Param        req_id path     string true "FinanceRequest ID (hex)"
// @Success      200    {object} repository.Response[repository.Credit]
// @Failure      400    {object} repository.Response[repository.Empty]
// @Failure      401    {object} repository.Response[repository.Empty]
// @Failure      403    {object} repository.Response[repository.Empty]
// @Failure      404    {object} repository.Response[repository.Empty]
// @Failure      409    {object} repository.Response[repository.Empty]
// @Failure      500    {object} repository.Response[repository.Empty]
// @Security     ApiKeyAuth
// @Router       /api/v1/borrowings/{id}/requests/{req_id}/reject [post]
// @Router       /api/v1/lendings/{id}/requests/{req_id}/reject [post]
func (h *FinanceRequestHandler) Reject(c fiber.Ctx) error {
	return h.decide(c, false)
}

func (h *FinanceRequestHandler) decide(c fiber.Ctx, approve bool) error {
	userID, err := services.GetUserIDFromContext(c)
	if err != nil {
		return repository.Unauthorized(c, "unauthorized")
	}
	creditID, err := bson.ObjectIDFromHex(c.Params("id"))
	if err != nil {
		return repository.BadRequest(c, "invalid credit id")
	}
	requestID, err := bson.ObjectIDFromHex(c.Params("req_id"))
	if err != nil {
		return repository.BadRequest(c, "invalid request id")
	}

	credit, err := h.repo.GetByID(c.Context(), creditID)
	if err != nil || credit == nil {
		return repository.NotFound(c, "credit not found")
	}
	if !credit.IsTwoParty() {
		return repository.BadRequest(c, "finance requests apply only to two-party credits")
	}
	if !credit.HasParty(userID) {
		return repository.Forbidden(c, "forbidden")
	}

	idx := -1
	for i := range credit.FinanceRequests {
		if credit.FinanceRequests[i].ID == requestID {
			idx = i
			break
		}
	}
	if idx == -1 {
		return repository.NotFound(c, "finance request not found")
	}
	req := &credit.FinanceRequests[idx]
	if req.Status != repository.FinanceRequestPending {
		return repository.Conflict(c, "finance request already decided")
	}
	if req.RequestedBy == userID {
		return repository.Forbidden(c, "requester cannot decide their own request")
	}

	now := time.Now()
	if approve {
		if err := applyDelta(credit, req.Type, req.Amount); err != nil {
			return repository.BadRequest(c, err.Error())
		}
		req.Status = repository.FinanceRequestApproved
	} else {
		req.Status = repository.FinanceRequestRejected
	}
	req.DecidedBy = userID
	req.DecidedAt = now
	req.UpdatedAt = now

	if err := h.repo.Update(c.Context(), creditID, credit); err != nil {
		return repository.Internal(c, "failed to update credit")
	}
	msg := "finance request approved"
	if !approve {
		msg = "finance request rejected"
	}
	return repository.OK(c, msg, credit)
}
