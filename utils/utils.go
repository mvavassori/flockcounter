package utils

import (
	"errors"
	"net/http"
	"strconv"

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
