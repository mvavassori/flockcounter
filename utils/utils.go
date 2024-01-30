package utils

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
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

	return id, nil
}
