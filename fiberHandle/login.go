package fiberHandle

import (
	"fmt"
	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"main/database"
	"strings"
)

type LoginRequest struct {
	User string `json:"user"`
	Pass string `json:"pass"`
}

func Login(handler *database.DBHandler, c *fiber.Ctx) error {
	var req LoginRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).SendString("Invalid request body")
	}

	user := req.User
	pass := req.Pass

	if user == "" || pass == "" {
		tokenString := c.Query("token")
		if tokenString == "" {
			tokenString = c.Get("Authorization")
			if strings.HasPrefix(tokenString, "Bearer ") {
				tokenString = strings.TrimPrefix(tokenString, "Bearer ")
			}
		}
		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			// Validate the signing method
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fiber.NewError(fiber.StatusUnauthorized, "invalid signing method")
			}
			return []byte("secret"), nil
		})
		if err != nil || !token.Valid {
			return c.SendStatus(fiber.StatusUnauthorized)
		}
	} else {
		if res, _ := handler.NameExists(user); !res {
			return fiber.NewError(fiber.StatusUnauthorized, "User not found or Password does not match")
		}
		if res, _ := handler.PasswordMatch(user, pass); !res {
			return fiber.NewError(fiber.StatusUnauthorized, "User not found or Password does not match")
		}
		fmt.Println(user)
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
