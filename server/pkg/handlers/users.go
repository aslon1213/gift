package handlers

import (
	"aslon1213/gift/pkg/repository"
	"aslon1213/gift/services"
	"net/http"

	"github.com/gofiber/fiber/v3"
	"go.mongodb.org/mongo-driver/v2/bson"
)

type UserHandler struct {
	repo *repository.UserRepository
}

// NewUserHandler creates a new UserHandler.
// @Summary Create a new UserHandler
// @Description Returns a new handler struct for user operations
// @Tags users
func NewUserHandler(repo *repository.UserRepository) *UserHandler {
	return &UserHandler{repo: repo}
}

// List lists all users.
// @Summary List users
// @Description Returns a list of users
// @Tags users
// @Produce json
// @Param query query string false "Query"
// @Success 200 {object} repository.Response
// @Failure 400 {object} repository.Response
// @Failure 500 {object} repository.Response
// @Router /api/v1/users [get]
func (h *UserHandler) Query(c fiber.Ctx) error {

	// get user_id from context
	user_id, err := services.GetUserIDFromContext(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(repository.NewResponse("error", "unauthorized", nil))
	}

	query_param := c.Query("query")
	if query_param == "" {
		return c.Status(fiber.StatusBadRequest).JSON(repository.NewResponse("error", "query is required", nil))
	}

	query := bson.M{
		"$or": []bson.M{
			{
				"name": bson.M{"$regex": query_param, "$options": "i"},
			},
			{
				"email": bson.M{"$regex": query_param, "$options": "i"},
			},
		},
		"_id": bson.M{"$ne": user_id},
	}
	users, err := h.repo.Query(c.Context(), query)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(repository.NewResponse("error", "internal server error", nil))
	}
	return c.Status(fiber.StatusOK).JSON(repository.NewResponse("success", "users fetched successfully", users))

}

// GetByID gets a user by ID.
// @Summary Get user by ID
// @Description Returns a single user by their ID
// @Tags users
// @Produce json
// @Param id path string true "User ID"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/v1/users/{id} [get]
func (h *UserHandler) GetByID(c fiber.Ctx) error {
	user_id, err := bson.ObjectIDFromHex(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid user id"})
	}

	user, err := h.repo.GetByID(c.Context(), user_id)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "internal server error"})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{"data": user})
}

// UpdateUserRequest represents the request body for updating a user.
// @Description UpdateUserRequest contains user fields required for the update API.
// @Tags users
type UpdateUserRequest struct {
	Name            string `json:"name"`
	Email           string `json:"email"`
	Password        string `json:"password"`
	ConfirmPassword string `json:"confirm_password"`
}

// Update updates a user's information.
// @Summary Update user
// @Description Updates user details by ID
// @Tags users
// @Accept json
// @Produce json
// @Param id path string true "User ID"
// @Param user body UpdateUserRequest true "Update User Request"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/v1/users/{id} [put]
func (h *UserHandler) Update(c fiber.Ctx) error {
	var input UpdateUserRequest
	if err := c.Bind().Body(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(repository.NewResponse("error", "invalid request body", nil))
	}

	user_id, err := bson.ObjectIDFromHex(c.Params("id"))
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(repository.NewResponse("error", err.Error(), nil))
	}

	user, err := h.repo.GetByID(c.Context(), user_id)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(repository.NewResponse("error", "internal server error", nil))
	}

	if input.Name != "" {
		user.Name = input.Name
	} else {
		return c.Status(http.StatusBadRequest).JSON(repository.NewResponse("error", "name is required", nil))
	}
	if input.Email != "" {
		user.Email = input.Email
	} else {
		return c.Status(http.StatusBadRequest).JSON(repository.NewResponse("error", "email is required", nil))
	}
	if input.Password != "" && input.ConfirmPassword != "" && input.Password == input.ConfirmPassword {
		user.Password = input.Password
	} else {
		return c.Status(http.StatusBadRequest).JSON(repository.NewResponse("error", "password and confirm password do not match", nil))
	}
	err = h.repo.Update(c.Context(), user_id, user)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(repository.NewResponse("error", "internal server error", nil))
	}
	return c.Status(http.StatusOK).JSON(repository.NewResponse("success", "user updated successfully", user))
}

// Delete deletes a user by ID.
// @Summary Delete user
// @Description Deletes a user by their ID
// @Tags users
// @Produce json
// @Param id path string true "User ID"
// @Success 501 {object} map[string]string "not implemented"
// @Router /api/v1/users/{id} [delete]
func (h *UserHandler) Delete(c fiber.Ctx) error {
	return c.Status(fiber.StatusNotImplemented).JSON(fiber.Map{"error": "not implemented"})
}
