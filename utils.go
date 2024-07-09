package main

import (
	"fmt"
	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"strings"
	"time"
)

type RegisterRequest struct {
	User  string `json:"user"`
	Pass  string `json:"pass"`
	Email string `json:"email"`
}

type LoginRequest struct {
	User string `json:"user"`
	Pass string `json:"pass"`
}

func register(handler *DBHandler, c *fiber.Ctx) error {
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
	if handler.nameExists(user) {
		return fiber.NewError(fiber.StatusConflict, "User already exists")
	}
	fmt.Println(user, pass, email)
	_, err := handler.db.Exec("INSERT INTO users (name, password, email) VALUES (?, ?, ?)", user, pass, email)
	if err != nil {
		fmt.Println(err)
		return c.SendStatus(fiber.StatusInternalServerError)
	}
	fmt.Println("11111111111111")
	return c.SendStatus(fiber.StatusCreated)
}

func login(handler *DBHandler, c *fiber.Ctx) error {
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
		if !handler.nameExists(user) {
			return fiber.NewError(fiber.StatusUnauthorized, "User not found or Password does not match")
		}
		if !handler.passwordMatch(user, pass) {
			return fiber.NewError(fiber.StatusUnauthorized, "User not found or Password does not match")
		}
		fmt.Println(user)
	}
	// Create the Claims
	claims := jwt.MapClaims{
		"name":  user,
		"admin": false,
		"exp":   time.Now().Add(time.Hour * 72).Unix(),
	}

	// Create token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// Generate encoded token and send it as response.
	t, err := token.SignedString([]byte("secret"))
	if err != nil {
		return c.SendStatus(fiber.StatusInternalServerError)
	}

	return c.JSON(fiber.Map{"token": t})
}

func keyExists(p *WebSocketPool, key string) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	_, exists := p.connections[key]
	return exists
}
