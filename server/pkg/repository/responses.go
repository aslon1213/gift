package repository

import "github.com/gofiber/fiber/v3"

type Response struct {
	Status  string      `json:"status"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}

func NewResponse(status string, message string, data interface{}) *Response {
	return &Response{Status: status, Message: message, Data: data}
}

func (r *Response) Success(c fiber.Ctx, data interface{}) error {
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"status":  "success",
		"message": "success",
		"data":    data,
	})
}

func (r *Response) Error(c fiber.Ctx, message string) error {
	return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
		"status":  "error",
		"message": message,
		"data":    nil,
	})
}

func (r *Response) NotFound(c fiber.Ctx, message string) error {
	return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
		"status":  "not found",
		"message": message,
		"data":    nil,
	})
}

func (r *Response) BadRequest(c fiber.Ctx, message string) error {
	return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
		"status":  "bad request",
		"message": message,
		"data":    nil,
	})
}

func (r *Response) InternalServerError(c fiber.Ctx, message string) error {
	return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
		"status":  "internal server error",
		"message": message,
		"data":    nil,
	})
}

func (r *Response) Unauthorized(c fiber.Ctx, message string) error {
	return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
		"status":  "unauthorized",
		"message": message,
		"data":    nil,
	})
}

func (r *Response) Forbidden(c fiber.Ctx, message string) error {
	return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
		"status":  "forbidden",
		"message": message,
		"data":    nil,
	})
}

func (r *Response) Conflict(c fiber.Ctx, message string) error {
	return c.Status(fiber.StatusConflict).JSON(fiber.Map{
		"status":  "conflict",
		"message": message,
		"data":    nil,
	})
}
