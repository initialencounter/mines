package main

import (
	"fmt"
	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"strings"
	"time"
)

func login(c *fiber.Ctx) error {
	user := c.FormValue("user")
	pass := c.FormValue("pass")

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
		fmt.Println(user, pass)
		// Throws Unauthorized error
		exists := keyExists(&pool, user)
		if exists {
			return fiber.NewError(fiber.StatusForbidden, "User already exists:"+user)
		}
		if pass != "doe" {
			return c.SendStatus(fiber.StatusUnauthorized)
		}
	}
	// Create the Claims
	claims := jwt.MapClaims{
		"name":  "John Doe",
		"admin": true,
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
