package main

import (
	"database/sql"

	"github.com/gorilla/mux"
	"github.com/mvavassori/bare-analytics/handlers"
	"github.com/mvavassori/bare-analytics/middleware"
)

func SetupRouter(db *sql.DB) *mux.Router {

	router := mux.NewRouter()

	// visit routes
	router.HandleFunc("/api/visits", handlers.GetVisits(db)).Methods("GET")
	router.HandleFunc("/api/visit/{id}", handlers.GetVisit(db)).Methods("GET")
	router.HandleFunc("/api/visit", handlers.CreateVisit(db)).Methods("POST")
	router.HandleFunc("/api/visit/{id}", handlers.UpdateVisit(db)).Methods("PUT")
	router.HandleFunc("/api/visit/{id}", handlers.DeleteVisit(db)).Methods("DELETE")

	// user routes
	router.HandleFunc("/api/users", handlers.GetUsers(db)).Methods("GET")
	router.Handle("/api/user/{id}", middleware.AuthMiddleware(handlers.GetUser(db))).Methods("GET")
	router.HandleFunc("/api/user", handlers.CreateUser(db, false)).Methods("POST") // false to indicate that it's not an admin
	// router.HandleFunc("/api/admin/user", handlers.CreateUser(db, true)).Methods("POST") // just to create the first admin user
	router.Handle("/api/admin/user", middleware.AuthMiddleware(middleware.AdminMiddleware(db)(handlers.CreateUser(db, true)))).Methods("POST") // must be authenticated and an admin
	router.HandleFunc("/api/user/{id}", handlers.UpdateUser(db)).Methods("PUT")
	router.HandleFunc("/api/user/{id}", handlers.DeleteUser(db)).Methods("DELETE")

	router.HandleFunc("/api/user/login", handlers.Login(db)).Methods("POST")
	router.HandleFunc("/api/user/refresh-token", handlers.RefreshToken(db)).Methods("POST")

	// website routes
	router.HandleFunc("/api/websites", handlers.GetWebsites(db)).Methods("GET")
	router.HandleFunc("/api/website/{id}", handlers.GetWebsite(db)).Methods("GET")
	router.HandleFunc("/api/website", handlers.CreateWebsite(db)).Methods("POST")
	router.HandleFunc("/api/website/{id}", handlers.UpdateWebsite(db)).Methods("PUT")
	router.HandleFunc("/api/website/{id}", handlers.DeleteWebsite(db)).Methods("DELETE")

	// dashboard routes
	router.HandleFunc("/api/dashboard/top-stats/{id}", handlers.GetTopStats(db)).Methods("GET")
	router.HandleFunc("/api/dashboard/pages/{id}", handlers.GetPages(db)).Methods("GET")
	router.HandleFunc("/api/dashboard/referrers/{id}", handlers.GetReferrers(db)).Methods("GET")
	router.HandleFunc("/api/dashboard/device-types/{id}", handlers.GetDeviceTypes(db)).Methods("GET")
	router.HandleFunc("/api/dashboard/oses/{id}", handlers.GetOSes(db)).Methods("GET")
	router.HandleFunc("/api/dashboard/browsers/{id}", handlers.GetBrowsers(db)).Methods("GET")
	router.HandleFunc("/api/dashboard/languages/{id}", handlers.GetLanguages(db)).Methods("GET")
	router.HandleFunc("/api/dashboard/countries/{id}", handlers.GetCountries(db)).Methods("GET")
	router.HandleFunc("/api/dashboard/states/{id}", handlers.GetStates(db)).Methods("GET")

	return router
}
