package services

import (
	"aslon1213/gift/configs"
	"aslon1213/gift/pkg/repository"
	"context"
	"errors"
	"fmt"
	"time"

	jwtware "github.com/gofiber/contrib/v3/jwt"
	"github.com/gofiber/fiber/v3"
	"github.com/golang-jwt/jwt/v5"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"golang.org/x/crypto/bcrypt"
)

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrInvalidToken       = errors.New("invalid token")
	ErrExpiredToken       = errors.New("token has expired")
	ErrEmailInUse         = errors.New("email already in use")
)

// AuthService provides authentication functionality
type AuthService struct {
	userRepo         *repository.UserRepository
	refreshTokenRepo *repository.RefreshTokenRepository
	jwtSecret        []byte
	accessTokenTTL   time.Duration
}

// HashPassword creates a bcrypt hash from a plain-text password
func HashPassword(password string) (string, error) {
	// The cost determines how computationally expensive the hash is
	// Higher is more secure but slower (default is 10)
	hashedBytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hashedBytes), nil
}

// VerifyPassword checks if the provided password matches the stored hash
func VerifyPassword(hashedPassword, providedPassword string) error {
	return bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(providedPassword))
}

// NewAuthService creates a new authentication service
func NewAuthService(userRepo *repository.UserRepository, refreshTokenRepo *repository.RefreshTokenRepository, jwtSecret string, accessTokenTTL time.Duration) *AuthService {
	if jwtSecret == "" {
		panic("jwt secret must not be empty")
	}

	return &AuthService{
		userRepo:         userRepo,
		refreshTokenRepo: refreshTokenRepo,
		jwtSecret:        []byte(jwtSecret),
		accessTokenTTL:   accessTokenTTL,
	}
}

func (s *AuthService) GetUserByEmail(email string) (*repository.User, error) {
	user, err := s.userRepo.GetUserByEmail(email)
	if err != nil {
		return nil, err
	}

	return user, nil
}

// Register creates a new user with the provided credentials
func (s *AuthService) Register(email, username, password string) (*repository.User, error) {
	// Check if user already exists
	_, err := s.userRepo.GetUserByEmail(email)
	if err == nil {
		return nil, ErrEmailInUse
	}
	// Return database errors; only proceed if user not found
	if !errors.Is(err, mongo.ErrNoDocuments) {
		return nil, err
	}

	// Hash the password
	hashedPassword, err := HashPassword(password)
	if err != nil {
		return nil, err
	}

	user := &repository.User{
		Email:    email,
		Name:     username,
		Password: hashedPassword,
	}

	err = s.userRepo.Create(
		context.Background(),
		user,
	)
	if err != nil {
		return nil, err
	}
	return user, nil
}

// generateAccessToken creates a new JWT access token
func (s *AuthService) generateAccessToken(user *repository.User) (string, error) {
	// Set the expiration time
	expirationTime := time.Now().Add(s.accessTokenTTL)

	// Create the JWT claims
	claims := jwt.MapClaims{
		"sub":      user.ID.Hex(),         // subject (user ID)
		"username": user.Name,             // custom claim
		"email":    user.Email,            // custom claim
		"exp":      expirationTime.Unix(), // expiration time
		"iat":      time.Now().Unix(),     // issued at time
	}

	// Create the token with claims
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// Sign the token with our secret key
	tokenString, err := token.SignedString(s.jwtSecret)
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

// ValidateToken verifies a JWT token and returns the claims
func (s *AuthService) ValidateToken(tokenString string) (jwt.MapClaims, error) {
	// Parse the token
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// Validate the signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrInvalidToken
		}
		return s.jwtSecret, nil
	})

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrExpiredToken
		}
		return nil, ErrInvalidToken
	}

	// Extract and validate claims
	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, ErrInvalidToken
}

// LoginWithRefresh authenticates a user and returns both access and refresh tokens
func (s *AuthService) LoginWithRefresh(email, password string, refreshTokenTTL time.Duration) (accessToken string, refreshToken string, err error) {
	// Get the user from the database
	user, err := s.userRepo.GetUserByEmail(email)
	if err != nil {
		return "", "", ErrInvalidCredentials
	}

	// Verify the password
	if err := VerifyPassword(user.Password, password); err != nil {
		return "", "", ErrInvalidCredentials
	}

	// Generate an access token
	accessToken, err = s.generateAccessToken(user)
	if err != nil {
		return "", "", err
	}

	// Create a refresh token
	token, err := s.refreshTokenRepo.CreateRefreshToken(user.ID, refreshTokenTTL)
	if err != nil {
		return "", "", err
	}

	return accessToken, token.Token, nil
}

// RefreshAccessToken creates a new access token using a refresh token and rotates the refresh token.
func (s *AuthService) RefreshAccessToken(refreshTokenString string) (accessToken string, refreshToken string, err error) {
	// Retrieve the refresh token
	token, err := s.refreshTokenRepo.GetRefreshToken(refreshTokenString)
	if err != nil {
		return "", "", ErrInvalidToken
	}

	// Check if the token is valid
	if token.Revoked {
		return "", "", ErrInvalidToken
	}

	// Check if the token has expired
	if time.Now().After(token.ExpiresAt) {
		return "", "", ErrExpiredToken
	}

	// Get the user
	user, err := s.userRepo.GetByID(context.Background(), token.UserId)
	if err != nil {
		return "", "", err
	}

	// Generate a new access token
	accessToken, err = s.generateAccessToken(user)
	if err != nil {
		return "", "", err
	}

	config := configs.GetConfig()

	if err := s.refreshTokenRepo.RevokeRefreshToken(refreshTokenString); err != nil {
		return "", "", err
	}

	newToken, err := s.refreshTokenRepo.CreateRefreshToken(user.ID, config.Auth.JwtRefreshExpiresIn)
	if err != nil {
		return "", "", err
	}

	return accessToken, newToken.Token, nil
}

// RevokeAllUserRefreshTokens revokes all refresh tokens for a user.
func (s *AuthService) RevokeAllUserRefreshTokens(userID bson.ObjectID) error {
	if err := s.refreshTokenRepo.RevokeAllUserTokens(userID); err != nil {
		return fmt.Errorf("failed to revoke refresh tokens: %w", err)
	}
	return nil
}

func GetUserIDFromContext(c fiber.Ctx) (bson.ObjectID, error) {
	token := jwtware.FromContext(c)
	userIDStr, err := token.Claims.GetSubject()
	if err != nil || userIDStr == "" {
		return bson.ObjectID{}, errors.New("user_id missing in context")
	}
	userID, err := bson.ObjectIDFromHex(userIDStr)
	if err != nil {
		return bson.ObjectID{}, errors.New("invalid user_id")
	}
	return userID, nil
}
