package repository

import "github.com/gofiber/fiber/v3"

// Response is the unified envelope returned by every handler.
// T is the payload type, captured into Swagger via generic annotations
// (e.g. `repository.Response[repository.Budget]`), so each endpoint gets
// a concrete schema in the OpenAPI spec while the runtime stays uniform.
type Response[T any] struct {
	Status  string `json:"status"  example:"success"`
	Message string `json:"message" example:"ok"`
	Data    T      `json:"data,omitempty"`
}

// Empty is the payload type for endpoints with no data (4xx/5xx, deletes,
// link/unlink). Using it keeps `Response[Empty]` swaggerable.
type Empty struct{}

// --- success helpers ----------------------------------------------------

func OK[T any](c fiber.Ctx, msg string, data T) error {
	return c.Status(fiber.StatusOK).JSON(Response[T]{Status: "success", Message: msg, Data: data})
}

func Created[T any](c fiber.Ctx, msg string, data T) error {
	return c.Status(fiber.StatusCreated).JSON(Response[T]{Status: "success", Message: msg, Data: data})
}

// Ack is 200 with no payload — for deletes, link/unlink, etc.
func Ack(c fiber.Ctx, msg string) error {
	return c.Status(fiber.StatusOK).JSON(Response[Empty]{Status: "success", Message: msg, Data: Empty{}})
}

// --- error helpers (always payload-less) --------------------------------

func Fail(c fiber.Ctx, code int, msg string) error {
	return c.Status(code).JSON(Response[Empty]{Status: "error", Message: msg, Data: Empty{}})
}

func BadRequest(c fiber.Ctx, msg string) error {
	return Fail(c, fiber.StatusBadRequest, msg)
}

func Unauthorized(c fiber.Ctx, msg string) error {
	return Fail(c, fiber.StatusUnauthorized, msg)
}

func Forbidden(c fiber.Ctx, msg string) error {
	return Fail(c, fiber.StatusForbidden, msg)
}

func NotFound(c fiber.Ctx, msg string) error {
	return Fail(c, fiber.StatusNotFound, msg)
}

func Conflict(c fiber.Ctx, msg string) error {
	return Fail(c, fiber.StatusConflict, msg)
}

func Internal(c fiber.Ctx, msg string) error {
	return Fail(c, fiber.StatusInternalServerError, msg)
}

func NotImplemented(c fiber.Ctx, msg string) error {
	return Fail(c, fiber.StatusNotImplemented, msg)
}
