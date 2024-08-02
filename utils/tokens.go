package utils

import (
	"fmt"
	"time"

	jwt "github.com/golang-jwt/jwt/v5"
)

var (
	AccessTokenExpiration = NewExpirationTime(10 * time.Second) // for testing
	// AccessTokenExpiration  = NewExpirationTime(15 * time.Minute)

	// RefreshTokenExpiration = NewExpirationTime(15 * time.Second) // for testing
	RefreshTokenExpiration = NewExpirationTime(14 * 24 * time.Hour)
)

func ValidateToken(tokenString string) (*jwt.Token, error) {
	//todo get secret from env
	secret := "my_secret_key"

	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(secret), nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		// Check if the token is expired
		if exp, ok := claims["expiresAt"].(float64); ok {
			if time.Now().Unix() > int64(exp) {
				return nil, fmt.Errorf("token is expired")
			}
		}
		return token, nil
	}

	return nil, fmt.Errorf("invalid token")
}

func CreateAccessToken(userID int, role string, name string, email string) (string, error) {
	//todo get secret from env
	secret := "my_secret_key"

	//expirationTime := time.Now().Add(time.Minute * 15).Unix() // 15 minutes
	// expirationTime := time.Now().Add(time.Second * 10).Unix() // for debugging

	// Create the Claims
	claims := &jwt.MapClaims{
		"userId":    userID,
		"role":      role,
		"name":      name,
		"email":     email,
		"expiresAt": AccessTokenExpiration.Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

func CreateRefreshToken(userID int) (string, error) {
	//todo get secret from env
	secret := "my_secret_key"

	// expirationTime := time.Now().Add(time.Hour * 24 * 14).Unix() // 14 days
	// expirationTime := time.Now().Add(time.Second * 15).Unix() // for debugging

	// Create the Claims
	claims := &jwt.MapClaims{
		"userId":    userID,
		"expiresAt": RefreshTokenExpiration.Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}
