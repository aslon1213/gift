package handlers

import (
	"aslon1213/gift/pkg/repository"
	"aslon1213/gift/services"
	"log"
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
// @Summary      List or filter groups for the user
// @Description  Query all groups for the user by name. Returns groups where the user is the owner or a member.
// @Tags         groups
// @Produce      json
// @Param        name query    string false "Group name"
// @Success      200  {object} repository.Response[[]repository.Group]
// @Failure      400  {object} repository.Response[repository.Empty]
// @Failure      401  {object} repository.Response[repository.Empty]
// @Failure      500  {object} repository.Response[repository.Empty]
// @Security     BearerAuth
// @Router       /groups [get]
func (h *GroupHandler) Query(c fiber.Ctx) error {
	token := jwtware.FromContext(c)
	log.Println(token.Claims)
	userIDStr, err := token.Claims.GetSubject()
	if err != nil || userIDStr == "" {
		return repository.Unauthorized(c, "user_id missing in context")
	}
	userID, err := bson.ObjectIDFromHex(userIDStr)
	if err != nil {
		return repository.BadRequest(c, "invalid user_id")
	}

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

	groups, err := h.repo.Query(c.Context(), query)
	if err != nil {
		return repository.Internal(c, "internal server error")
	}

	return repository.OK(c, "groups fetched successfully", groups)
}

// GetByID godoc
// @Summary      Get group by ID
// @Description  Get a specific group by its ID
// @Tags         groups
// @Produce      json
// @Param        id  path     string true "Group ID"
// @Success      200 {object} repository.Response[repository.Group]
// @Failure      400 {object} repository.Response[repository.Empty]
// @Failure      404 {object} repository.Response[repository.Empty]
// @Failure      500 {object} repository.Response[repository.Empty]
// @Security     BearerAuth
// @Router       /groups/{id} [get]
func (h *GroupHandler) GetByID(c fiber.Ctx) error {
	groupID, err := bson.ObjectIDFromHex(c.Params("id"))
	if err != nil {
		return repository.BadRequest(c, err.Error())
	}

	group, err := h.repo.GetByID(c.Context(), groupID)
	if err != nil {
		return repository.Internal(c, "internal server error")
	}
	return repository.OK(c, "group fetched successfully", group)
}

// CreateGroupRequest represents a request to create a new group
type CreateGroupRequest struct {
	Name      string          `json:"name" example:"Book Club"`
	OwnerID   bson.ObjectID   `json:"owner_id" swaggertype:"string" example:"60d5ec49f99a2d3a829a7d1e"`
	MemberIDs []bson.ObjectID `json:"member_ids" swaggertype:"array,string" example:"[\"60d5ec49f99a2d3a829a7d1f\"]"`
}

// Create godoc
// @Summary      Create group
// @Description  Create a new group
// @Tags         groups
// @Accept       json
// @Produce      json
// @Param        body body     CreateGroupRequest true "Group create request"
// @Success      201  {object} repository.Response[repository.Group]
// @Failure      400  {object} repository.Response[repository.Empty]
// @Failure      500  {object} repository.Response[repository.Empty]
// @Security     BearerAuth
// @Router       /groups [post]
func (h *GroupHandler) Create(c fiber.Ctx) error {
	var input CreateGroupRequest
	if err := c.Bind().Body(&input); err != nil {
		return repository.BadRequest(c, "invalid request body")
	}
	if input.Name == "" {
		return repository.BadRequest(c, "name is required")
	}
	if input.OwnerID.IsZero() {
		return repository.BadRequest(c, "owner_id is required")
	}
	group := &repository.Group{
		Name:      input.Name,
		OwnerID:   input.OwnerID,
		MemberIDs: input.MemberIDs,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	if err := h.repo.Create(c.Context(), group); err != nil {
		return repository.Internal(c, "internal server error")
	}
	return repository.Created(c, "group created successfully", group)
}

// UpdateGroupRequest represents a request to update a group
type UpdateGroupRequest struct {
	Name      string          `json:"name" example:"Book Club"`
	OwnerID   bson.ObjectID   `json:"owner_id" swaggertype:"string" example:"60d5ec49f99a2d3a829a7d1e"`
	MemberIDs []bson.ObjectID `json:"member_ids" swaggertype:"array,string" example:"[\"60d5ec49f99a2d3a829a7d1f\"]"`
}

// Update godoc
// @Summary      Update group
// @Description  Update a group's properties by id
// @Tags         groups
// @Accept       json
// @Produce      json
// @Param        id   path     string             true "Group ID"
// @Param        body body     UpdateGroupRequest true "Group update request"
// @Success      200  {object} repository.Response[repository.Group]
// @Failure      400  {object} repository.Response[repository.Empty]
// @Failure      500  {object} repository.Response[repository.Empty]
// @Security     BearerAuth
// @Router       /groups/{id} [put]
func (h *GroupHandler) Update(c fiber.Ctx) error {
	var input UpdateGroupRequest
	if err := c.Bind().Body(&input); err != nil {
		return repository.BadRequest(c, "invalid request body")
	}

	groupID, err := bson.ObjectIDFromHex(c.Params("id"))
	if err != nil {
		return repository.BadRequest(c, err.Error())
	}

	group, err := h.repo.GetByID(c.Context(), groupID)
	if err != nil {
		return repository.Internal(c, "internal server error")
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

	if err := h.repo.Update(c.Context(), groupID, group); err != nil {
		return repository.Internal(c, "internal server error")
	}
	return repository.OK(c, "group updated successfully", group)
}

// Delete godoc
// @Summary      Delete group
// @Description  Delete a group by id
// @Tags         groups
// @Produce      json
// @Param        id  path     string true "Group ID"
// @Success      200 {object} repository.Response[repository.Empty]
// @Failure      400 {object} repository.Response[repository.Empty]
// @Failure      500 {object} repository.Response[repository.Empty]
// @Security     BearerAuth
// @Router       /groups/{id} [delete]
func (h *GroupHandler) Delete(c fiber.Ctx) error {
	groupID, err := bson.ObjectIDFromHex(c.Params("id"))
	if err != nil {
		return repository.BadRequest(c, err.Error())
	}

	if err := h.repo.Delete(c.Context(), groupID); err != nil {
		return repository.Internal(c, "internal server error")
	}
	return repository.Ack(c, "group deleted successfully")
}

// InviteMemberRequest represents a request to invite a member to a group
type InviteMemberRequest struct {
	MemberID string `json:"member_id" example:"60d5ec49f99a2d3a829a7d1e"`
}

// InviteMember godoc
// @Summary      Invite member to group
// @Description  Invite a member to a group (only owner can invite)
// @Tags         groups
// @Accept       json
// @Produce      json
// @Param        id   path     string              true "Group ID"
// @Param        body body     InviteMemberRequest true "Invite member request"
// @Success      200  {object} repository.Response[repository.Group]
// @Failure      400  {object} repository.Response[repository.Empty]
// @Failure      401  {object} repository.Response[repository.Empty]
// @Failure      403  {object} repository.Response[repository.Empty]
// @Failure      404  {object} repository.Response[repository.Empty]
// @Failure      500  {object} repository.Response[repository.Empty]
// @Security     BearerAuth
// @Router       /groups/{id}/invite [post]
func (h *GroupHandler) InviteMember(c fiber.Ctx) error {
	groupID, err := bson.ObjectIDFromHex(c.Params("id"))
	if err != nil {
		return repository.BadRequest(c, "invalid group id")
	}

	var req InviteMemberRequest
	if err := c.Bind().Body(&req); err != nil {
		return repository.BadRequest(c, "invalid request body")
	}

	memberObjID, err := bson.ObjectIDFromHex(req.MemberID)
	if err != nil {
		return repository.BadRequest(c, "invalid member id")
	}

	group, err := h.repo.GetByID(c.Context(), groupID)
	if err != nil || group == nil {
		return repository.NotFound(c, "group not found")
	}

	ownerID, err := services.GetUserIDFromContext(c)
	if err != nil {
		return repository.Unauthorized(c, err.Error())
	}

	if group.OwnerID != ownerID {
		return repository.Forbidden(c, "only the owner can invite members")
	}

	for _, id := range group.MemberIDs {
		if id == memberObjID {
			return repository.BadRequest(c, "member already in group")
		}
	}

	group.MemberIDs = append(group.MemberIDs, memberObjID)
	group.UpdatedAt = time.Now()

	if err := h.repo.Update(c.Context(), groupID, group); err != nil {
		return repository.Internal(c, "failed to add member")
	}

	return repository.OK(c, "member invited successfully", group)
}

// RemoveMemberRequest represents a request to remove a member from a group
type RemoveMemberRequest struct {
	MemberID string `json:"member_id" example:"60d5ec49f99a2d3a829a7d1e"`
}

// RemoveMember godoc
// @Summary      Remove member from group
// @Description  Remove a member from a group (only owner can remove)
// @Tags         groups
// @Accept       json
// @Produce      json
// @Param        id   path     string              true "Group ID"
// @Param        body body     RemoveMemberRequest true "Remove member request"
// @Success      200  {object} repository.Response[repository.Group]
// @Failure      400  {object} repository.Response[repository.Empty]
// @Failure      401  {object} repository.Response[repository.Empty]
// @Failure      403  {object} repository.Response[repository.Empty]
// @Failure      404  {object} repository.Response[repository.Empty]
// @Failure      500  {object} repository.Response[repository.Empty]
// @Security     BearerAuth
// @Router       /groups/{id}/remove [post]
func (h *GroupHandler) RemoveMember(c fiber.Ctx) error {
	groupID, err := bson.ObjectIDFromHex(c.Params("id"))
	if err != nil {
		return repository.BadRequest(c, "invalid group id")
	}

	var req RemoveMemberRequest
	if err := c.Bind().Body(&req); err != nil {
		return repository.BadRequest(c, "invalid request body")
	}

	memberObjID, err := bson.ObjectIDFromHex(req.MemberID)
	if err != nil {
		return repository.BadRequest(c, "invalid member id")
	}

	group, err := h.repo.GetByID(c.Context(), groupID)
	if err != nil || group == nil {
		return repository.NotFound(c, "group not found")
	}

	ownerID, err := services.GetUserIDFromContext(c)
	if err != nil {
		return repository.Unauthorized(c, err.Error())
	}

	if group.OwnerID != ownerID {
		return repository.Forbidden(c, "only the owner can remove members")
	}

	found := false
	newMembers := make([]bson.ObjectID, 0, len(group.MemberIDs))
	for _, id := range group.MemberIDs {
		if id == memberObjID {
			found = true
			continue
		}
		newMembers = append(newMembers, id)
	}
	if !found {
		return repository.BadRequest(c, "member not found in group")
	}

	group.MemberIDs = newMembers
	group.UpdatedAt = time.Now()

	if err := h.repo.Update(c.Context(), groupID, group); err != nil {
		return repository.Internal(c, "failed to remove member")
	}
	return repository.OK(c, "member removed successfully", group)
}
