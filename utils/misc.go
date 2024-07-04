package utils

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"sort"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"github.com/mileusna/useragent"
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

type ErrorResponse struct {
	Message string `json:"message"`
}

func WriteErrorResponse(w http.ResponseWriter, statusCode int, err error) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(ErrorResponse{Message: err.Error()})
}

type dailySalt struct {
	salt []byte
	date time.Time
}

// Stored in memory
var dailySaltCache = make(map[string]dailySalt)

// getDailySalt generates a random 16-byte salt
func getDailySalt() ([]byte, error) {
	// Generate a random 16-byte salt
	salt := make([]byte, 16)
	_, err := rand.Read(salt)
	if err != nil {
		return nil, err
	}
	return salt, nil
}

// GenerateDailySalt generates a unique salt for the current day if it hasn't been generated yet.
func GenerateDailySalt() ([]byte, error) {
	now := time.Now()
	dateString := now.Format("2006-01-02")

	if salt, ok := dailySaltCache[dateString]; ok {
		return salt.salt, nil
	}

	salt, err := getDailySalt()
	if err != nil {
		return nil, err
	}

	dailySaltCache[dateString] = dailySalt{salt: salt, date: now}
	return salt, nil
}

func GenerateUniqueIdentifier(dailySalt []byte, websiteDomain, ipAddress, userAgent string) (string, error) {
	// Combine daily salt, website domain, IP address, and user agent
	combinedString := string(dailySalt) + websiteDomain + ipAddress + userAgent

	// Hash the combined string using SHA-256
	hasher := sha256.New()
	hasher.Write([]byte(combinedString))
	hashedBytes := hasher.Sum(nil)

	// Convert hashed bytes to a hexadecimal string
	hashedString := hex.EncodeToString(hashedBytes)

	return hashedString, nil
}

func SortByDate(slice []map[string]interface{}) {
	sort.Slice(slice, func(i, j int) bool {
		t1, _ := time.Parse("2006-01-02", slice[i]["date"].(string))
		t2, _ := time.Parse("2006-01-02", slice[j]["date"].(string))
		return t1.Before(t2)
	})
}

func SortByPeriod(slice []map[string]interface{}, interval string) {
	var layout string

	// Determine the time layout based on the interval
	switch interval {
	case "hour":
		layout = "15"
	case "month":
		layout = "2006-01"
	case "day":
		fallthrough
	default:
		layout = "2006-01-02"
	}

	sort.Slice(slice, func(i, j int) bool {
		t1, _ := time.Parse(layout, slice[i]["period"].(string))
		t2, _ := time.Parse(layout, slice[j]["period"].(string))
		return t1.Before(t2)
	})
}
