package fiberHandle

import (
	"github.com/gofiber/fiber/v2"
	"main/database"
	"main/utils"
)

type ResetPasswordRequest struct {
	User string `json:"user"`
	Code string `json:"code"`
}

func ResetPassword(handler *database.DBHandler, c *fiber.Ctx, codeCache *utils.CodeCache) error {
	var req ResetPasswordRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).SendString("Invalid request body")
	}

	userName := req.User
	code := req.Code
	entry, found := codeCache.Get(userName)
	if !found {
		return c.Status(fiber.StatusBadRequest).SendString("Invalid user")
	}
	if entry.Code != code {
		return c.Status(fiber.StatusBadRequest).SendString("Invalid code")
	}
	if utils.IsCodeExpired(entry.CreationTime) {
		return c.Status(fiber.StatusBadRequest).SendString("Code expired")
	}
	err := handler.ChangePasswordByName(userName, code)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("Failed to change password")
	}
	return c.Status(fiber.StatusOK).SendString("Code valid")
}
