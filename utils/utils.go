package utils

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	jwt "github.com/golang-jwt/jwt/v5"
	"github.com/gorilla/mux"
	"github.com/mileusna/useragent"
	// "github.com/mvavassori/bare-analytics/models"
)

func ExtractIDFromURL(r *http.Request) (int, error) {
	vars := mux.Vars(r)
	idStr, ok := vars["id"]
	if !ok {
		return 0, errors.New("ID not provided in the URL")
	}

	id, err := strconv.Atoi(idStr)
	if err != nil {
		return 0, errors.New("ID must be a number")
	}

	if id <= 0 {
		return 0, errors.New("ID must be greater than zero")
	}

	return id, nil
}

func ExtractDomainFromURL(r *http.Request) (string, error) {
	vars := mux.Vars(r)

	domain, ok := vars["domain"]
	if !ok {
		return "", errors.New("domain not provided in the URL")
	}

	return domain, nil
}

func GetDeviceType(ua *useragent.UserAgent) string {
	if ua.Mobile {
		return "Mobile"
	} else if ua.Tablet {
		return "Tablet"
	} else if ua.Desktop {
		return "Desktop"
	} else if ua.Bot {
		return "Bot"
	} else {
		return "Unknown"
	}
}

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

func CreateAccessToken(userID int) (string, error) {
	//todo get secret from env
	secret := "my_secret_key"
	// Create the Claims
	claims := &jwt.MapClaims{
		"userId":    userID,
		"expiresAt": time.Now().Add(time.Minute * 15).Unix(), // 15 minutes
		// "expiresAt": time.Now().Add(time.Second * 15).Unix(), // 15 seconds tests
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

func CreateRefreshToken(userID int) (string, error) {
	//todo get secret from env
	secret := "my_secret_key"
	// Create the Claims
	claims := &jwt.MapClaims{
		"userId":    userID,
		"expiresAt": time.Now().Add(time.Hour * 24 * 7).Unix(), // 7 days
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}
