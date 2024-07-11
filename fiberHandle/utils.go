package fiberHandle

import (
	"github.com/golang-jwt/jwt/v5"
	"time"
)

func signToken(user string) (string, error) {
	// Create the Claims
	claims := jwt.MapClaims{
		"name":  user,
		"admin": false,
		"exp":   time.Now().Add(time.Hour * 72).Unix(),
	}

	// Create token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// Generate encoded token and send it as response.
	return token.SignedString([]byte("secret"))
}
