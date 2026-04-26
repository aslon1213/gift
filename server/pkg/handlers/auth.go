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
	Email    string `json:"email"`
	Username string `json:"username"`
	Password string `json:"password"`
	Currency string `json:"currency"`
}

// RegisterResponse defines the response returned after registration
// swagger:model
type RegisterResponse struct {
	Id       string `json:"id"`
	Email    string `json:"email"`
	Username string `json:"username"`
}

// LoginRequest defines the payload for login
// swagger:model
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// LoginResponse is returned upon successful login
// swagger:model
type LoginResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

// RefreshRequest is the input for refreshing JWT tokens
// swagger:model
type RefreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

// RefreshResponse defines the output from token refresh
// swagger:model
type RefreshResponse struct {
	Token        string `json:"token"`
	RefreshToken string `json:"refresh_token"`
}

// AuthHandler contains HTTP handlers for authentication
type AuthHandler struct {
	authService *services.AuthService
	users       *repository.UserRepository
	credits     *repository.CreditRepository
}

// NewAuthHandler creates a new auth handler
func NewAuthHandler(authService *services.AuthService, users *repository.UserRepository, credits *repository.CreditRepository) *AuthHandler {
	return &AuthHandler{
		authService: authService,
		users:       users,
		credits:     credits,
	}
}

// UserInfoResponse is the GetUserInfo payload: the user record plus a
// computed credit summary the client renders directly.
type UserInfoResponse struct {
	*repository.User
	Credits *repository.CreditSummary `json:"credits"`
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
	return len(password) >= 8
}

// Login godoc
// @Summary      Log in
// @Description  Log in with email and password
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        data body     LoginRequest true "Login credentials"
// @Success      200  {object} repository.Response[LoginResponse]
// @Failure      400  {object} repository.Response[repository.Empty]
// @Failure      401  {object} repository.Response[repository.Empty]
// @Failure      500  {object} repository.Response[repository.Empty]
// @Router       /auth/login [post]
func (ah *AuthHandler) Login(c fiber.Ctx) error {
	config := configs.GetConfig()

	var input LoginRequest
	if err := c.Bind().Body(&input); err != nil {
		return repository.BadRequest(c, "Error on login request")
	}

	if strings.TrimSpace(input.Email) == "" || strings.TrimSpace(input.Password) == "" {
		return repository.BadRequest(c, "Email and password are required")
	}

	accessToken, refreshToken, err := ah.authService.LoginWithRefresh(input.Email, input.Password, config.Auth.JwtRefreshExpiresIn)
	if err != nil {
		if errors.Is(err, services.ErrInvalidCredentials) {
			return repository.Unauthorized(c, "Invalid credentials")
		}
		return repository.Internal(c, "Internal Server Error")
	}

	setJWTCookie(c, accessToken)

	return repository.OK(c, "Success login", LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	})
}

// Logout godoc
// @Summary      Log out
// @Description  Log out and revoke all refresh tokens for user
// @Tags         auth
// @Produce      json
// @Success      200 {object} repository.Response[repository.Empty]
// @Failure      401 {object} repository.Response[repository.Empty]
// @Failure      500 {object} repository.Response[repository.Empty]
// @Router       /auth/logout [post]
// @Security     ApiKeyAuth
func (ah *AuthHandler) Logout(c fiber.Ctx) error {
	tok, ok := c.Locals("user").(*jwt.Token)
	if !ok {
		return repository.Unauthorized(c, "Invalid authentication token")
	}

	claims, ok := tok.Claims.(jwt.MapClaims)
	if !ok {
		return repository.Unauthorized(c, "Invalid authentication token claims")
	}

	sub, ok := claims["sub"].(string)
	if !ok {
		return repository.Unauthorized(c, "Invalid authentication token subject")
	}

	userID, err := bson.ObjectIDFromHex(sub)
	if err != nil {
		return repository.Unauthorized(c, "Invalid authentication token subject")
	}

	if err := ah.authService.RevokeAllUserRefreshTokens(userID); err != nil {
		return repository.Internal(c, "Failed to revoke refresh tokens on logout")
	}

	c.Cookie(&fiber.Cookie{
		Name:     "jwt",
		Value:    "",
		Expires:  time.Now().Add(-time.Hour),
		HTTPOnly: true,
		Secure:   c.Protocol() == "https",
		SameSite: "Lax",
	})

	return repository.Ack(c, "Success logout")
}

// Register godoc
// @Summary      Register user
// @Description  Register a new user
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        data body     RegisterRequest true "Registration request"
// @Success      200  {object} repository.Response[RegisterResponse]
// @Failure      400  {object} repository.Response[repository.Empty]
// @Failure      409  {object} repository.Response[repository.Empty]
// @Failure      500  {object} repository.Response[repository.Empty]
// @Router       /auth/register [post]
func (ah *AuthHandler) Register(c fiber.Ctx) error {
	var input RegisterRequest
	if err := c.Bind().Body(&input); err != nil {
		return repository.BadRequest(c, "Error on register request")
	}
	if strings.TrimSpace(input.Email) == "" || strings.TrimSpace(input.Username) == "" || strings.TrimSpace(input.Password) == "" {
		return repository.BadRequest(c, "Email, username, and password are required")
	}

	if !isValidPassword(input.Password) {
		return repository.BadRequest(c, "Password should be at least 8 characters long")
	}

	if !isValidEmail(input.Email) {
		return repository.BadRequest(c, "Invalid email")
	}
	if input.Currency == "" {
		input.Currency = "UZS"
	}

	user, err := ah.authService.Register(input.Email, input.Username, input.Password, input.Currency)
	if err != nil {
		if errors.Is(err, services.ErrEmailInUse) {
			return repository.Conflict(c, "Email already in use")
		}
		return repository.Internal(c, "Error on registering user")
	}

	return repository.OK(c, "Success register", RegisterResponse{
		Id:       user.ID.Hex(),
		Email:    user.Email,
		Username: user.Name,
	})
}

// RefreshToken godoc
// @Summary      Refresh JWT tokens
// @Description  Refresh access and refresh tokens using a refresh token
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        data body     RefreshRequest true "Refresh request"
// @Success      200  {object} repository.Response[RefreshResponse]
// @Failure      400  {object} repository.Response[repository.Empty]
// @Failure      401  {object} repository.Response[repository.Empty]
// @Failure      500  {object} repository.Response[repository.Empty]
// @Router       /auth/refresh [post]
func (ah *AuthHandler) RefreshToken(c fiber.Ctx) error {
	var input RefreshRequest
	if err := c.Bind().Body(&input); err != nil {
		return repository.BadRequest(c, "Invalid request payload")
	}
	token, newRefreshToken, err := ah.authService.RefreshAccessToken(input.RefreshToken)
	if err != nil {
		if errors.Is(err, services.ErrInvalidToken) || errors.Is(err, services.ErrExpiredToken) {
			return repository.Unauthorized(c, "Invalid or expired refresh token")
		}
		return repository.Internal(c, "Internal server error")
	}
	setJWTCookie(c, token)

	return repository.OK(c, "Success refresh token", RefreshResponse{
		Token:        token,
		RefreshToken: newRefreshToken,
	})
}

// GetUserInfo godoc
// @Summary      Get user info
// @Description  Retrieves the authenticated user's information along with a credit summary.
// @Tags         auth
// @Accept       json
// @Produce      json
// @Success      200  {object} repository.Response[UserInfoResponse]
// @Failure      401  {object} repository.Response[repository.Empty]
// @Failure      500  {object} repository.Response[repository.Empty]
// @Router       /api/v1/auth/me [get]
func (ah *AuthHandler) GetUserInfo(c fiber.Ctx) error {
	userID, err := services.GetUserIDFromContext(c)
	if err != nil {
		return repository.Unauthorized(c, "Invalid authentication token")
	}
	user, err := ah.users.GetByID(context.Background(), userID)
	if err != nil || user == nil {
		return repository.Internal(c, "internal server error")
	}
	user.Password = ""
	if user.Currency == "" {
		user.Currency = "UZS"
	}
	credits, err := ah.credits.Summary(context.Background(), userID)
	if err != nil {
		return repository.Internal(c, "internal server error")
	}
	return repository.OK(c, "user info fetched successfully", UserInfoResponse{
		User:    user,
		Credits: credits,
	})
}
