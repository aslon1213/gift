package handlers

import (
	"aslon1213/gift/configs"
	"aslon1213/gift/pkg/repository"
	"aslon1213/gift/services"
	"context"
	"errors"
	"net/mail"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/golang-jwt/jwt/v5"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// RegisterRequest defines the payload for user registration
// swagger:model
type RegisterRequest struct {
	// User e-mail address
	// required: true
	Email string `json:"email"`
	// Username for the account
	// required: true
	Username string `json:"username"`
	// Password for the account (>=8 chars)
	// required: true
	Password string `json:"password"`
	// Currency for the account
	// required: true
	Currency string `json:"currency"`
}

// RegisterResponse defines the response returned after registration
// swagger:model
type RegisterResponse struct {
	// The user's ID
	Id string `json:"id"`
	// The user's email
	Email string `json:"email"`
	// The user's username
	Username string `json:"username"`
}

// LoginRequest defines the payload for login
// swagger:model
type LoginRequest struct {
	// User's email
	// required: true
	Email string `json:"email"`
	// User's password
	// required: true
	Password string `json:"password"`
}

// LoginResponse is returned upon successful login
// swagger:model
type LoginResponse struct {
	// JWT access token
	AccessToken string `json:"access_token"`
	// JWT refresh token
	RefreshToken string `json:"refresh_token"`
}

// RefreshRequest is the input for refreshing JWT tokens
// swagger:model
type RefreshRequest struct {
	// Refresh Token
	// required: true
	RefreshToken string `json:"refresh_token"`
}

// RefreshResponse defines the output from token refresh
// swagger:model
type RefreshResponse struct {
	// New access token
	Token string `json:"token"`
	// New refresh token
	RefreshToken string `json:"refresh_token"`
}

// AuthHandler contains HTTP handlers for authentication
type AuthHandler struct {
	authService *services.AuthService
	users       *repository.UserRepository
}

// NewAuthHandler creates a new auth handler
func NewAuthHandler(authService *services.AuthService, users *repository.UserRepository) *AuthHandler {
	return &AuthHandler{
		authService: authService,
		users:       users,
	}
}

func getAccessTokenExpiry(token string) time.Time {
	expiration := time.Now().Add(15 * time.Minute)
	parser := new(jwt.Parser)
	claims := jwt.MapClaims{}

	if _, _, err := parser.ParseUnverified(token, claims); err != nil {
		return expiration
	}

	exp, ok := claims["exp"].(float64)
	if !ok || exp <= 0 {
		return expiration
	}
	return time.Unix(int64(exp), 0)
}

func setJWTCookie(c fiber.Ctx, token string) {
	c.Cookie(&fiber.Cookie{
		Name:     "jwt",
		Value:    token,
		Expires:  getAccessTokenExpiry(token),
		HTTPOnly: true,
		Secure:   c.Protocol() == "https",
		SameSite: "Lax",
	})
}

func isValidEmail(email string) bool {
	_, err := mail.ParseAddress(email)
	return err == nil
}

func isValidPassword(password string) bool {
	// password should be at least 8 characters long
	return len(password) >= 8
}

// Login godoc
// @Summary Log in
// @Description Log in with email and password
// @Tags auth
// @Accept  json
// @Produce  json
// @Param  data body LoginRequest true "Login credentials"
// @Success 200 {object} LoginResponse
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /auth/login [post]
func (ah *AuthHandler) Login(c fiber.Ctx) error {
	config := configs.GetConfig()

	var input LoginRequest
	if err := c.Bind().Body(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "Error on login request",
			"data":    nil,
		})
	}

	if strings.TrimSpace(input.Email) == "" || strings.TrimSpace(input.Password) == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "Email and password are required",
			"data":    nil,
		})
	}

	// Authenticate the user
	accessToken, refreshToken, err := ah.authService.LoginWithRefresh(input.Email, input.Password, config.Auth.JwtRefreshExpiresIn)
	if err != nil {
		if errors.Is(err, services.ErrInvalidCredentials) {

			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"status":  "error",
				"message": "Invalid credentials",
				"data":    nil,
			})
		} else {

			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"status":  "error",
				"message": "Internal Server Error",
				"data":    nil,
			})
		}
	}

	setJWTCookie(c, accessToken)

	// Return the token
	response := LoginResponse{AccessToken: accessToken, RefreshToken: refreshToken}

	return c.JSON(fiber.Map{
		"status":  "success",
		"message": "Success login",
		"data":    response,
	})
}

// Logout godoc
// @Summary Log out
// @Description Log out and revoke all refresh tokens for user
// @Tags auth
// @Produce  json
// @Success 200 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /auth/logout [post]
// @Security ApiKeyAuth
func (ah *AuthHandler) Logout(c fiber.Ctx) error {
	tok, ok := c.Locals("user").(*jwt.Token)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"status":  "error",
			"message": "Invalid authentication token",
			"data":    nil,
		})
	}

	claims, ok := tok.Claims.(jwt.MapClaims)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"status":  "error",
			"message": "Invalid authentication token claims",
			"data":    nil,
		})
	}

	sub, ok := claims["sub"].(string)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"status":  "error",
			"message": "Invalid authentication token subject",
			"data":    nil,
		})
	}

	userID, err := bson.ObjectIDFromHex(sub)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"status":  "error",
			"message": "Invalid authentication token subject",
			"data":    nil,
		})
	}

	if err := ah.authService.RevokeAllUserRefreshTokens(userID); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "Failed to revoke refresh tokens on logout",
			"data":    nil,
		})
	}

	// Clear cookie
	c.Cookie(&fiber.Cookie{
		Name:     "jwt",
		Value:    "",
		Expires:  time.Now().Add(-time.Hour),
		HTTPOnly: true,
		Secure:   c.Protocol() == "https",
		SameSite: "Lax",
	})

	return c.JSON(fiber.Map{
		"status":  "success",
		"message": "Success logout",
		"data":    nil,
	})
}

// Register godoc
// @Summary Register user
// @Description Register a new user
// @Tags auth
// @Accept  json
// @Produce  json
// @Param  data body RegisterRequest true "Registration request"
// @Success 200 {object} RegisterResponse
// @Failure 400 {object} map[string]interface{}
// @Failure 409 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /auth/register [post]
func (ah *AuthHandler) Register(c fiber.Ctx) error {
	var input RegisterRequest
	if err := c.Bind().Body(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "Error on register request",
			"data":    nil,
		})
	}
	if strings.TrimSpace(input.Email) == "" || strings.TrimSpace(input.Username) == "" || strings.TrimSpace(input.Password) == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "Email, username, and password are required",
			"data":    nil,
		})
	}

	// validate password
	if !isValidPassword(input.Password) {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "Password should be at least 8 characters long",
			"data":    nil,
		})
	}

	if !isValidEmail(input.Email) {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "Invalid email",
			"data":    nil,
		})
	}
	if input.Currency == "" {
		input.Currency = "UZS"
	}

	user, err := ah.authService.Register(input.Email, input.Username, input.Password, input.Currency)
	if err != nil {
		if errors.Is(err, services.ErrEmailInUse) {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{
				"status":  "error",
				"message": "Email already in use",
				"data":    nil,
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "Error on registering user",
			"data":    nil,
		})
	}

	newUser := RegisterResponse{
		Id:       user.ID.Hex(),
		Email:    user.Email,
		Username: user.Name,
	}

	return c.JSON(fiber.Map{
		"status":  "success",
		"message": "Success register",
		"data":    newUser,
	})
}

// RefreshToken godoc
// @Summary Refresh JWT tokens
// @Description Refresh access and refresh tokens using a refresh token
// @Tags auth
// @Accept  json
// @Produce  json
// @Param  data body RefreshRequest true "Refresh request"
// @Success 200 {object} RefreshResponse
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /auth/refresh [post]
func (ah *AuthHandler) RefreshToken(c fiber.Ctx) error {
	var input RefreshRequest
	if err := c.Bind().Body(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "Invalid request payload",
			"data":    nil,
		})
	}
	token, newRefreshToken, err := ah.authService.RefreshAccessToken(input.RefreshToken)
	if err != nil {
		if errors.Is(err, services.ErrInvalidToken) || errors.Is(err, services.ErrExpiredToken) {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"status":  "error",
				"message": "Invalid or expired refresh token",
				"data":    nil,
			})
		} else {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"status":  "error",
				"message": "Internal server error",
				"data":    nil,
			})
		}
	}
	// Clear cookie
	setJWTCookie(c, token)

	response := RefreshResponse{Token: token, RefreshToken: newRefreshToken}

	return c.JSON(fiber.Map{
		"status":  "success",
		"message": "Success refresh token",
		"data":    response,
	})
}

// GetUserInfo godoc
// @Summary      Get user info
// @Description  Retrieves the authenticated user's information
// @Tags         auth
// @Accept       json
// @Produce      json
// @Success      200  {object}  map[string]interface{}  "Success"
// @Failure      401  {object}  map[string]interface{}  "Unauthorized"
// @Failure      500  {object}  map[string]interface{}  "Internal server error"
// @Router       /api/v1/auth/me [get]
func (ah *AuthHandler) GetUserInfo(c fiber.Ctx) error {
	userID, err := services.GetUserIDFromContext(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"status":  "error",
			"message": "Invalid authentication token",
			"data":    nil,
		})
	}
	user, err := ah.users.GetByID(context.Background(), userID)
	if err != nil || user == nil {
		return c.Status(fiber.StatusInternalServerError).JSON(repository.NewResponse("error", "internal server error", nil))
	}
	user.Password = ""
	if user.Currency == "" {
		user.Currency = "UZS"
	}
	return c.Status(fiber.StatusOK).JSON(repository.NewResponse("success", "user info fetched successfully", user))
}
