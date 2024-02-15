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
	// The jwt.Parse method returns a *jwt.Token and an error. The *jwt.Token is a struct that contains the claims in the JWT. The error will be non-nil if there was an error while parsing or validating the JWT.
	return jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// Don't forget to validate the alg is what you expect:
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}

		return []byte(secret), nil
	})
}

func CreateToken(userID int) (string, error) {
	//todo get secret from env
	secret := "my_secret_key"
	// Create the Claims
	claims := &jwt.MapClaims{
		"userId": userID,
		// "expiresAt": time.Now().Add(time.Hour * 24 * 7).Unix(), // test
		"expiresAt": time.Now().Add(time.Second * 15).Unix(), // 1 week
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}
