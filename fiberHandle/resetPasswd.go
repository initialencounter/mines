package fiberHandle

import (
	"github.com/gofiber/fiber/v2"
	"main/database"
	"main/utils"
)

type ResetPasswordRequest struct {
	User string `json:"user"`
	Pass string `json:"pass"`
	Code string `json:"code"`
}

func ResetPassword(handler *database.DBHandler, c *fiber.Ctx, codeCache *utils.CodeCache) error {
	var req ResetPasswordRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).SendString("Invalid request body")
	}

	userName := req.User
	code := req.Code
	pass := req.Pass
	if res, _ := handler.NameExists(userName); !res {
		return fiber.NewError(fiber.StatusUnauthorized, "User not found")
	}
	entry, found := codeCache.Get(userName)
	if !found || entry.Code != code || utils.IsCodeExpired(entry.CreationTime) {
		return c.Status(fiber.StatusBadRequest).SendString("Invalid code")
	}
	err := handler.ChangePasswordByName(userName, pass)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("Failed to change password")
	}
	codeCache.Delete(userName)
	return c.Status(fiber.StatusOK).SendString("Code valid")
}
