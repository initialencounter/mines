package fiberHandle

import (
	"fmt"
	"github.com/gofiber/fiber/v2"
	"main/database"
)

type RegisterRequest struct {
	User  string `json:"user"`
	Pass  string `json:"pass"`
	Email string `json:"email"`
}

func Register(handler *database.DBHandler, c *fiber.Ctx) error {
	var req RegisterRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).SendString("Invalid request body")
	}

	user := req.User
	pass := req.Pass
	email := req.Email

	if user == "" || pass == "" || email == "" {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid request body not enough data")
	}
	if res, _ := handler.NameExists(user); res {
		return fiber.NewError(fiber.StatusConflict, "User already exists")
	}

	if VerifyUserName(user) {
		return fiber.NewError(fiber.StatusBadRequest, "The user name cannot contain special characters")
	}

	if valid, reason := VerifyEmail(email); !valid {
		return fiber.NewError(fiber.StatusBadRequest, reason)
	}

	err := handler.InsertRecord(user, pass, email, 0)
	if err != nil {
		fmt.Println(err)
		return c.SendStatus(fiber.StatusInternalServerError)
	}

	t, err := signToken(user)
	if err != nil {
		return c.SendStatus(fiber.StatusInternalServerError)
	}
	id, err := handler.GetId(user)
	if err != nil {
		return err
	}
	return c.JSON(fiber.Map{"token": t, "id": id})
}
