package handlers

import (
	"aslon1213/gift/pkg/repository"
	"aslon1213/gift/services"

	"github.com/gofiber/fiber/v3"
	"go.mongodb.org/mongo-driver/v2/bson"
)

type UserHandler struct {
	repo *repository.UserRepository
}

func NewUserHandler(repo *repository.UserRepository) *UserHandler {
	return &UserHandler{repo: repo}
}

// Query lists users matching a query string (excluding the caller).
// @Summary      List users
// @Description  Returns a list of users
// @Tags         users
// @Produce      json
// @Param        query query     string false "Query"
// @Success      200   {object}  repository.Response[[]repository.User]
// @Failure      400   {object}  repository.Response[repository.Empty]
// @Failure      401   {object}  repository.Response[repository.Empty]
// @Failure      500   {object}  repository.Response[repository.Empty]
// @Router       /api/v1/users [get]
func (h *UserHandler) Query(c fiber.Ctx) error {
	userID, err := services.GetUserIDFromContext(c)
	if err != nil {
		return repository.Unauthorized(c, "unauthorized")
	}

	queryParam := c.Query("query")
	if queryParam == "" {
		return repository.BadRequest(c, "query is required")
	}

	query := bson.M{
		"$or": []bson.M{
			{"name": bson.M{"$regex": queryParam, "$options": "i"}},
			{"email": bson.M{"$regex": queryParam, "$options": "i"}},
		},
		"_id": bson.M{"$ne": userID},
	}
	users, err := h.repo.Query(c.Context(), query)
	if err != nil {
		return repository.Internal(c, "internal server error")
	}
	return repository.OK(c, "users fetched successfully", users)
}

// GetByID gets a user by ID.
// @Summary      Get user by ID
// @Description  Returns a single user by their ID
// @Tags         users
// @Produce      json
// @Param        id  path     string true "User ID"
// @Success      200 {object} repository.Response[repository.User]
// @Failure      400 {object} repository.Response[repository.Empty]
// @Failure      500 {object} repository.Response[repository.Empty]
// @Router       /api/v1/users/{id} [get]
func (h *UserHandler) GetByID(c fiber.Ctx) error {
	userID, err := bson.ObjectIDFromHex(c.Params("id"))
	if err != nil {
		return repository.BadRequest(c, "invalid user id")
	}

	user, err := h.repo.GetByID(c.Context(), userID)
	if err != nil {
		return repository.Internal(c, "internal server error")
	}
	return repository.OK(c, "user fetched successfully", user)
}

// UpdateUserRequest represents the request body for updating a user.
type UpdateUserRequest struct {
	Name            string `json:"name"`
	Email           string `json:"email"`
	Password        string `json:"password"`
	ConfirmPassword string `json:"confirm_password"`
}

// Update updates a user's information.
// @Summary      Update user
// @Description  Updates user details by ID
// @Tags         users
// @Accept       json
// @Produce      json
// @Param        id   path     string             true "User ID"
// @Param        user body     UpdateUserRequest  true "Update User Request"
// @Success      200  {object} repository.Response[repository.User]
// @Failure      400  {object} repository.Response[repository.Empty]
// @Failure      500  {object} repository.Response[repository.Empty]
// @Router       /api/v1/users/{id} [put]
func (h *UserHandler) Update(c fiber.Ctx) error {
	var input UpdateUserRequest
	if err := c.Bind().Body(&input); err != nil {
		return repository.BadRequest(c, "invalid request body")
	}

	userID, err := bson.ObjectIDFromHex(c.Params("id"))
	if err != nil {
		return repository.BadRequest(c, err.Error())
	}

	user, err := h.repo.GetByID(c.Context(), userID)
	if err != nil {
		return repository.Internal(c, "internal server error")
	}

	if input.Name == "" {
		return repository.BadRequest(c, "name is required")
	}
	user.Name = input.Name

	if input.Email == "" {
		return repository.BadRequest(c, "email is required")
	}
	user.Email = input.Email

	if input.Password == "" || input.ConfirmPassword == "" || input.Password != input.ConfirmPassword {
		return repository.BadRequest(c, "password and confirm password do not match")
	}
	user.Password = input.Password

	if err := h.repo.Update(c.Context(), userID, user); err != nil {
		return repository.Internal(c, "internal server error")
	}
	return repository.OK(c, "user updated successfully", user)
}

// Delete deletes a user by ID.
// @Summary      Delete user
// @Description  Deletes a user by their ID
// @Tags         users
// @Produce      json
// @Param        id  path     string true "User ID"
// @Failure      501 {object} repository.Response[repository.Empty]
// @Router       /api/v1/users/{id} [delete]
func (h *UserHandler) Delete(c fiber.Ctx) error {
	return repository.NotImplemented(c, "not implemented")
}
