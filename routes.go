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
	router.Handle("/api/visits", middleware.AdminMiddleware(handlers.GetVisits(db))).Methods("GET")
	router.Handle("/api/visit/{id}", middleware.AdminMiddleware(handlers.GetVisit(db))).Methods("GET")
	router.HandleFunc("/api/visit", handlers.CreateVisit(db)).Methods("POST")
	router.Handle("/api/visit/{id}", middleware.AdminMiddleware(handlers.UpdateVisit(db))).Methods("PUT")
	router.Handle("/api/visit/{id}", middleware.AdminMiddleware(handlers.DeleteVisit(db))).Methods("DELETE")

	// user routes
	router.Handle("/api/users", middleware.AdminMiddleware(handlers.GetUsers(db))).Methods("GET")
	router.Handle("/api/user/{id}", middleware.AdminOrOwnerMiddleware(handlers.GetUser(db))).Methods("GET")
	router.HandleFunc("/api/user", handlers.CreateUser(db, false)).Methods("POST") // false to indicate that we'll create a regular user
	router.Handle("/api/user/{id}", middleware.AdminOrOwnerMiddleware(handlers.UpdateUser(db))).Methods("PUT")
	router.Handle("/api/user/{id}", middleware.AdminOrOwnerMiddleware(handlers.DeleteUser(db))).Methods("DELETE")

	// auth routes
	router.HandleFunc("/api/user/login", handlers.Login(db)).Methods("POST")
	router.HandleFunc("/api/user/refresh-token", handlers.RefreshToken(db)).Methods("POST")

	// admin user routes
	router.Handle("/api/admin/user", middleware.AdminMiddleware(handlers.CreateUser(db, true))).Methods("POST") // true to indicate that we'll create an admin user
	// router.HandleFunc("/api/admin/user", handlers.CreateUser(db, true)).Methods("POST") // just to create the first admin user

	// website routes
	router.Handle("/api/websites", middleware.AdminMiddleware(handlers.GetWebsites(db))).Methods("GET")
	router.Handle("/api/website/{id}", middleware.AdminOrUserWebsiteMiddleware(db)(handlers.GetWebsite(db))).Methods("GET")
	router.Handle("/api/website", middleware.AdminOrAuthMiddleware(handlers.CreateWebsite(db))).Methods("POST")
	router.Handle("/api/website/{id}", middleware.AdminOrUserWebsiteMiddleware(db)(handlers.UpdateWebsite(db))).Methods("PUT")
	router.Handle("/api/website/{id}", middleware.AdminOrUserWebsiteMiddleware(db)(handlers.DeleteWebsite(db))).Methods("DELETE")

	// dashboard routes
	router.Handle("/api/dashboard/top-stats/{id}", middleware.AdminOrUserWebsiteMiddleware(db)(handlers.GetTopStats(db))).Methods("GET")
	router.Handle("/api/dashboard/pages/{id}", middleware.AdminOrUserWebsiteMiddleware(db)(handlers.GetPages(db))).Methods("GET")
	router.Handle("/api/dashboard/referrers/{id}", middleware.AdminOrUserWebsiteMiddleware(db)(handlers.GetReferrers(db))).Methods("GET")
	router.Handle("/api/dashboard/device-types/{id}", middleware.AdminOrUserWebsiteMiddleware(db)(handlers.GetDeviceTypes(db))).Methods("GET")
	router.Handle("/api/dashboard/oses/{id}", middleware.AdminOrUserWebsiteMiddleware(db)(handlers.GetOSes(db))).Methods("GET")
	router.Handle("/api/dashboard/browsers/{id}", middleware.AdminOrUserWebsiteMiddleware(db)(handlers.GetBrowsers(db))).Methods("GET")
	router.Handle("/api/dashboard/languages/{id}", middleware.AdminOrUserWebsiteMiddleware(db)(handlers.GetLanguages(db))).Methods("GET")
	router.Handle("/api/dashboard/countries/{id}", middleware.AdminOrUserWebsiteMiddleware(db)(handlers.GetCountries(db))).Methods("GET")
	router.Handle("/api/dashboard/states/{id}", middleware.AdminOrUserWebsiteMiddleware(db)(handlers.GetStates(db))).Methods("GET")

	return router
}
