package handlers

import (
	"net/http"
)

func SetupRouter() *http.ServeMux {
	router := http.NewServeMux()

	router.HandleFunc("/visits", GetVisitsHandler)
	router.HandleFunc("/visit/", GetVisitHandler)
	router.HandleFunc("/visit", PostVisitHandler)

	return router
}
