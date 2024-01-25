package main

import (
	"database/sql"

	"github.com/gorilla/mux"
	"github.com/mvavassori/bare-analytics/handlers"
)

func SetupRouter(db *sql.DB) *mux.Router {

	router := mux.NewRouter()

	router.HandleFunc("/api/visits", handlers.GetVisits(db)).Methods("GET")
	router.HandleFunc("/api/visit/{id}", handlers.GetVisit(db)).Methods("GET")
	router.HandleFunc("/api/visit", handlers.CreateVisit(db)).Methods("POST")
	router.HandleFunc("/api/visit/{id}", handlers.UpdateVisit(db)).Methods("PUT")
	router.HandleFunc("/api/visit/{id}", handlers.DeleteVisit(db)).Methods("DELETE")

	// Add other routes as needed...

	return router
}
