package jwtToken

import (
	"fmt"
	"github.com/golang-jwt/jwt/v5"
	"time"
)

func New(
	chatId string,
	tokenTTL time.Duration,
	secret []byte,
) (
	string,
	error,
) {
	token := jwt.New(jwt.SigningMethodHS256)

	claims := token.Claims.(jwt.MapClaims)
	claims["chat_id"] = chatId
	claims["exp"] = time.Now().Add(tokenTTL).Unix()

	tokenString, err := token.SignedString(secret)
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

func VerifyToken(tokenString string, secret []byte) (string, error) {
	claims := jwt.MapClaims{}
	token, err := jwt.ParseWithClaims(
		tokenString, claims, func(token *jwt.Token) (interface{}, error) {
			return secret, nil
		},
	)

	// Check for verification errors
	if err != nil {
		fmt.Println("error in jwt.ParseWithClaims(...): ", err.Error())
		return "", err
	}

	// Check if the token is valid
	if !token.Valid {
		fmt.Println("jwt.ParseWithClaims(...) сказал что токен не валидный")
		return "", fmt.Errorf("invalid token")
	}

	// Return the verified token
	return claims["chat_id"].(string), nil
}
