package middleware

import (
	"aslon1213/gift/configs"
	"strings"

	jwtware "github.com/gofiber/contrib/v3/jwt"
	"github.com/gofiber/fiber/v3"
)

var config *configs.Config // nolint:unused

// Protected protect routes
func Protected() fiber.Handler {

	config := configs.GetConfig()

	return jwtware.New(jwtware.Config{
		SigningKey: jwtware.SigningKey{Key: []byte(config.Auth.JwtSecret)},
		ErrorHandler: func(c fiber.Ctx, err error) error {
			status := fiber.StatusUnauthorized
			message := "Invalid or expired JWT"

			if strings.Contains(strings.ToLower(err.Error()), "missing or malformed jwt") {
				status = fiber.StatusBadRequest
				message = "Missing or malformed JWT"
			}

			return c.Status(status).JSON(fiber.Map{
				"status":  "error",
				"message": message,
				"data":    nil,
			})
		},
	})
}
