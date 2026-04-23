package handlers

import (
	"aslon1213/gift/pkg/repository"
	"aslon1213/gift/services"
	"log"
	"net/http"
	"time"

	jwtware "github.com/gofiber/contrib/v3/jwt"
	"github.com/gofiber/fiber/v3"
	"go.mongodb.org/mongo-driver/v2/bson"
)

type GroupHandler struct {
	repo *repository.GroupRepository
}

func NewGroupHandler(repo *repository.GroupRepository) *GroupHandler {
	return &GroupHandler{repo: repo}
}

// Query godoc
// @Summary List or filter groups for the user
// @Description Query all groups for the user by name. Returns groups where the user is the owner or a member.
// @Tags groups
// @Produce json
// @Param name query string false "Group name"
// @Success 200 {object} repository.Response
// @Failure 400 {object} repository.Response
// @Failure 401 {object} repository.Response
// @Failure 500 {object} repository.Response
// @Security BearerAuth
// @Router /groups [get]
func (h *GroupHandler) Query(c fiber.Ctx) error {
	// Fetch user_id from locals

	token := jwtware.FromContext(c)
	log.Println(token.Claims)
	userIDStr, err := token.Claims.GetSubject()
	if err != nil || userIDStr == "" {
		return c.Status(http.StatusUnauthorized).JSON(repository.NewResponse("error", "user_id missing in context", nil))
	}
	userID, err := bson.ObjectIDFromHex(userIDStr)
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(repository.NewResponse("error", "invalid user_id", nil))
	}

	// Build query to fetch groups where user is owner or a member
	orConditions := []bson.M{
		{"owner_id": userID},
		{"member_ids": userID},
	}

	name := c.Query("name")
	match := bson.M{}
	if name != "" {
		match["name"] = name
	}

	query := bson.M{
		"$and": []bson.M{
			{"$or": orConditions},
		},
	}
	if len(match) > 0 {
		query["$and"] = append(query["$and"].([]bson.M), match)
	}

	// Pagination is not handled in repository, so we ignore limit/offset in query for now

	groups, err := h.repo.Query(c.Context(), query)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(repository.NewResponse("error", "internal server error", nil))
	}

	return c.Status(http.StatusOK).JSON(repository.NewResponse("success", "groups fetched successfully", groups))
}

// GetByID godoc
// @Summary Get group by ID
// @Description Get a specific group by its ID
// @Tags groups
// @Produce json
// @Param id path string true "Group ID"
// @Success 200 {object} repository.Response
// @Failure 400 {object} repository.Response
// @Failure 404 {object} repository.Response
// @Failure 500 {object} repository.Response
// @Router /groups/{id} [get]
// @Security BearerAuth
func (h *GroupHandler) GetByID(c fiber.Ctx) error {
	group_id, err := bson.ObjectIDFromHex(c.Params("id"))
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(repository.NewResponse("error", err.Error(), nil))
	}

	group, err := h.repo.GetByID(c.Context(), group_id)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(repository.NewResponse("error", "internal server error", nil))
	}
	return c.Status(http.StatusOK).JSON(repository.NewResponse("success", "group fetched successfully", group))
}

// CreateGroupRequest represents a request to create a new group
type CreateGroupRequest struct {
	Name      string          `json:"name" example:"Book Club"`
	OwnerID   bson.ObjectID   `json:"owner_id" swaggertype:"string" example:"60d5ec49f99a2d3a829a7d1e"`
	MemberIDs []bson.ObjectID `json:"member_ids" swaggertype:"array,string" example:"[\"60d5ec49f99a2d3a829a7d1f\"]"`
}

// Create godoc
// @Summary Create group
// @Description Create a new group
// @Tags groups
// @Accept json
// @Produce json
// @Param body body CreateGroupRequest true "Group create request"
// @Success 200 {object} repository.Response
// @Failure 400 {object} repository.Response
// @Failure 500 {object} repository.Response
// @Router /groups [post]
// @Security BearerAuth
func (h *GroupHandler) Create(c fiber.Ctx) error {
	var input CreateGroupRequest
	if err := c.Bind().Body(&input); err != nil {
		return c.Status(http.StatusBadRequest).JSON(repository.NewResponse("error", "invalid request body", nil))
	}
	if input.Name == "" {
		return c.Status(http.StatusBadRequest).JSON(repository.NewResponse("error", "name is required", nil))
	}
	if input.OwnerID.IsZero() {
		return c.Status(http.StatusBadRequest).JSON(repository.NewResponse("error", "owner_id is required", nil))
	}
	group := &repository.Group{
		Name:      input.Name,
		OwnerID:   input.OwnerID,
		MemberIDs: input.MemberIDs,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	err := h.repo.Create(c.Context(), group)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(repository.NewResponse("error", "internal server error", nil))
	}
	return c.Status(http.StatusOK).JSON(repository.NewResponse("success", "group created successfully", group))
}

// UpdateGroupRequest represents a request to update a group
type UpdateGroupRequest struct {
	Name      string          `json:"name" example:"Book Club"`
	OwnerID   bson.ObjectID   `json:"owner_id" swaggertype:"string" example:"60d5ec49f99a2d3a829a7d1e"`
	MemberIDs []bson.ObjectID `json:"member_ids" swaggertype:"array,string" example:"[\"60d5ec49f99a2d3a829a7d1f\"]"`
}

// Update godoc
// @Summary Update group
// @Description Update a group's properties by id
// @Tags groups
// @Accept json
// @Produce json
// @Param id path string true "Group ID"
// @Param body body UpdateGroupRequest true "Group update request"
// @Success 200 {object} repository.Response
// @Failure 400 {object} repository.Response
// @Failure 500 {object} repository.Response
// @Router /groups/{id} [put]
// @Security BearerAuth
func (h *GroupHandler) Update(c fiber.Ctx) error {
	var input UpdateGroupRequest
	if err := c.Bind().Body(&input); err != nil {
		return c.Status(http.StatusBadRequest).JSON(repository.NewResponse("error", "invalid request body", nil))
	}

	group_id, err := bson.ObjectIDFromHex(c.Params("id"))
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(repository.NewResponse("error", err.Error(), nil))
	}

	group, err := h.repo.GetByID(c.Context(), group_id)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(repository.NewResponse("error", "internal server error", nil))
	}

	if input.Name != "" {
		group.Name = input.Name
	}
	if !input.OwnerID.IsZero() {
		group.OwnerID = input.OwnerID
	}
	if input.MemberIDs != nil {
		group.MemberIDs = input.MemberIDs
	}
	group.UpdatedAt = time.Now()

	err = h.repo.Update(c.Context(), group_id, group)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(repository.NewResponse("error", "internal server error", nil))
	}
	return c.Status(http.StatusOK).JSON(repository.NewResponse("success", "group updated successfully", group))
}

// Delete godoc
// @Summary Delete group
// @Description Delete a group by id
// @Tags groups
// @Produce json
// @Param id path string true "Group ID"
// @Success 200 {object} repository.Response
// @Failure 400 {object} repository.Response
// @Failure 500 {object} repository.Response
// @Router /groups/{id} [delete]
// @Security BearerAuth
func (h *GroupHandler) Delete(c fiber.Ctx) error {
	group_id, err := bson.ObjectIDFromHex(c.Params("id"))
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(repository.NewResponse("error", err.Error(), nil))
	}

	err = h.repo.Delete(c.Context(), group_id)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(repository.NewResponse("error", "internal server error", nil))
	}
	return c.Status(http.StatusOK).JSON(repository.NewResponse("success", "group deleted successfully", nil))
}

// InviteMemberRequest represents a request to invite a member to a group
type InviteMemberRequest struct {
	MemberID string `json:"member_id" example:"60d5ec49f99a2d3a829a7d1e"`
}

// InviteMember godoc
// @Summary Invite member to group
// @Description Invite a member to a group (only owner can invite)
// @Tags groups
// @Accept json
// @Produce json
// @Param id path string true "Group ID"
// @Param body body InviteMemberRequest true "Invite member request"
// @Success 200 {object} repository.Response
// @Failure 400 {object} repository.Response
// @Failure 401 {object} repository.Response
// @Failure 403 {object} repository.Response
// @Failure 404 {object} repository.Response
// @Failure 500 {object} repository.Response
// @Router /groups/{id}/invite [post]
// @Security BearerAuth
func (h *GroupHandler) InviteMember(c fiber.Ctx) error {
	group_id, err := bson.ObjectIDFromHex(c.Params("id"))
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(repository.NewResponse("error", "invalid group id", nil))
	}

	var req InviteMemberRequest
	if err := c.Bind().Body(&req); err != nil {
		return c.Status(http.StatusBadRequest).JSON(repository.NewResponse("error", "invalid request body", nil))
	}

	memberObjID, err := bson.ObjectIDFromHex(req.MemberID)
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(repository.NewResponse("error", "invalid member id", nil))
	}

	group, err := h.repo.GetByID(c.Context(), group_id)
	if err != nil || group == nil {
		return c.Status(http.StatusNotFound).JSON(repository.NewResponse("error", "group not found", nil))
	}

	// Assume user ID is available in locals (e.g. set by middleware)
	ownerID, err := services.GetUserIDFromContext(c)
	if err != nil {
		return c.Status(http.StatusUnauthorized).JSON(repository.NewResponse("error", err.Error(), nil))
	}

	// Only owner can invite
	if group.OwnerID != ownerID {
		return c.Status(http.StatusForbidden).JSON(repository.NewResponse("error", "only the owner can invite members", nil))
	}

	// Do not add member if already exists
	for _, id := range group.MemberIDs {
		if id == memberObjID {
			return c.Status(http.StatusBadRequest).JSON(repository.NewResponse("error", "member already in group", nil))
		}
	}

	group.MemberIDs = append(group.MemberIDs, memberObjID)
	group.UpdatedAt = time.Now()

	err = h.repo.Update(c.Context(), group_id, group)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(repository.NewResponse("error", "failed to add member", nil))
	}

	return c.Status(http.StatusOK).JSON(repository.NewResponse("success", "member invited successfully", group))
}

// RemoveMemberRequest represents a request to remove a member from a group
type RemoveMemberRequest struct {
	MemberID string `json:"member_id" example:"60d5ec49f99a2d3a829a7d1e"`
}

// RemoveMember godoc
// @Summary Remove member from group
// @Description Remove a member from a group (only owner can remove)
// @Tags groups
// @Accept json
// @Produce json
// @Param id path string true "Group ID"
// @Param body body RemoveMemberRequest true "Remove member request"
// @Success 200 {object} repository.Response
// @Failure 400 {object} repository.Response
// @Failure 401 {object} repository.Response
// @Failure 403 {object} repository.Response
// @Failure 404 {object} repository.Response
// @Failure 500 {object} repository.Response
// @Router /groups/{id}/remove [post]
// @Security BearerAuth
func (h *GroupHandler) RemoveMember(c fiber.Ctx) error {
	group_id, err := bson.ObjectIDFromHex(c.Params("id"))
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(repository.NewResponse("error", "invalid group id", nil))
	}

	var req RemoveMemberRequest
	if err := c.Bind().Body(&req); err != nil {
		return c.Status(http.StatusBadRequest).JSON(repository.NewResponse("error", "invalid request body", nil))
	}

	memberObjID, err := bson.ObjectIDFromHex(req.MemberID)
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(repository.NewResponse("error", "invalid member id", nil))
	}

	group, err := h.repo.GetByID(c.Context(), group_id)
	if err != nil || group == nil {
		return c.Status(http.StatusNotFound).JSON(repository.NewResponse("error", "group not found", nil))
	}

	// Assume user ID is available in locals (e.g. set by middleware)
	ownerID, err := services.GetUserIDFromContext(c)
	if err != nil {
		return c.Status(http.StatusUnauthorized).JSON(repository.NewResponse("error", err.Error(), nil))
	}

	// Only owner can remove
	if group.OwnerID != ownerID {
		return c.Status(http.StatusForbidden).JSON(repository.NewResponse("error", "only the owner can remove members", nil))
	}

	// Remove the member from the list
	found := false
	newMembers := make([]bson.ObjectID, 0, len(group.MemberIDs))
	for _, id := range group.MemberIDs {
		if id == memberObjID {
			found = true
			continue // skip this member
		}
		newMembers = append(newMembers, id)
	}
	if !found {
		return c.Status(http.StatusBadRequest).JSON(repository.NewResponse("error", "member not found in group", nil))
	}

	group.MemberIDs = newMembers
	group.UpdatedAt = time.Now()

	err = h.repo.Update(c.Context(), group_id, group)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(repository.NewResponse("error", "failed to remove member", nil))
	}
	return c.Status(http.StatusOK).JSON(repository.NewResponse("success", "member removed successfully", group))
}
