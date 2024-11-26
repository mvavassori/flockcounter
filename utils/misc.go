package utils

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"regexp"
	"sort"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"github.com/mileusna/useragent"
)

type ErrorResponse struct {
	Message string `json:"message"`
}

func WriteErrorResponse(w http.ResponseWriter, statusCode int, err error) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(ErrorResponse{Message: err.Error()})
}

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

// Helper Functions for Password Rules
func HasSpecialChar(s string) bool {
	return regexp.MustCompile(`[!@#$%^&*(),.?":{}|<>]`).MatchString(s)
}

func HasNumber(s string) bool {
	return regexp.MustCompile(`[0-9]`).MatchString(s)
}

func HasUppercase(s string) bool {
	return regexp.MustCompile(`[A-Z]`).MatchString(s)
}

// nullableStringToJSON converts a sql.NullString to an interface{} that can be marshaled into JSON.
func NullableStringToJSON(ns sql.NullString) interface{} {
	if ns.Valid {
		return ns.String
	}
	return nil
}
