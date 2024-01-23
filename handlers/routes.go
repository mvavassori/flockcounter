package handlers

import (
	"net/http"
)

func SetupRouter() *http.ServeMux {
	router := http.NewServeMux()

	router.HandleFunc("/visit/{id}", GetVisitHandler)
	router.HandleFunc("/visits", GetVisitsHandler)

	return router
}
