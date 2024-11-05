package utils

import (
	"fmt"
	"os"
	"time"

	jwt "github.com/golang-jwt/jwt/v5"
)

type ExpirationTime struct {
	duration time.Duration
}

func NewExpirationTime(d time.Duration) ExpirationTime {
	return ExpirationTime{duration: d}
}

func (e ExpirationTime) Unix() int64 {
	return time.Now().Add(e.duration).Unix()
}

func (e ExpirationTime) Time() time.Time {
	return time.Now().Add(e.duration)
}

func (e ExpirationTime) Duration() time.Duration {
	return e.duration
}

var (
	// AccessTokenExpiration = NewExpirationTime(10 * time.Second) // for testing
	AccessTokenExpiration = NewExpirationTime(15 * time.Minute)

	// RefreshTokenExpiration = NewExpirationTime(15 * time.Second) // for testing
	RefreshTokenExpiration = NewExpirationTime(14 * 24 * time.Hour)
)

// ValidateTokenAndExtractClaims parses and validates the token, checks expiration, and returns the claims if valid.
func ValidateTokenAndExtractClaims(tokenString string) (jwt.MapClaims, error) {
	// Get the secret from environment variables (recommended for production)
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		// Fallback for development; avoid hardcoding in production.
		secret = "my_secret_key"
	}

	// Parse the token with the secret key
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// Ensure the token method is HMAC (common for JWT)
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(secret), nil
	})

	// Handle parsing errors
	if err != nil {
		return nil, fmt.Errorf("invalid token: %w", err)
	}

	// Extract claims and check if token is valid
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token claims")
	}

	// Check for expiration in claims
	if exp, ok := claims["expiresAt"].(float64); ok {
		if time.Now().Unix() > int64(exp) {
			return nil, fmt.Errorf("token is expired")
		}
	}

	return claims, nil
}

func CreateAccessToken(userID int, role string, name string, email string) (string, error) {
	// Get the secret from environment variables (recommended for production)
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		// Fallback for development; avoid hardcoding in production.
		secret = "my_secret_key"
	}

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
	// Get the secret from environment variables (recommended for production)
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		// Fallback for development; avoid hardcoding in production.
		secret = "my_secret_key"
	}

	// Create the Claims
	claims := &jwt.MapClaims{
		"userId":    userID,
		"expiresAt": RefreshTokenExpiration.Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}
