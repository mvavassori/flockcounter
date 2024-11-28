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
	router.Handle("/api/visits", middleware.Admin(handlers.GetVisits(db))).Methods("GET")
	router.HandleFunc("/api/visit", handlers.CreateVisit(db)).Methods("POST")
	router.Handle("/api/visit/{id}", middleware.Admin(handlers.DeleteVisit(db))).Methods("DELETE")

	// user routes
	router.Handle("/api/users", middleware.Admin(handlers.GetUsers(db))).Methods("GET")
	router.Handle("/api/user/{id}", middleware.AdminOrOwner(handlers.GetUser(db))).Methods("GET")
	router.HandleFunc("/api/user", handlers.CreateUser(db, false)).Methods("POST") // false to indicate that we'll create a regular user
	router.Handle("/api/user/{id}", middleware.AdminOrOwner(handlers.UpdateUser(db))).Methods("PUT")
	router.Handle("/api/user/{id}", middleware.AdminOrOwner(handlers.DeleteUser(db))).Methods("DELETE")

	// auth routes
	router.HandleFunc("/api/user/login", handlers.Login(db)).Methods("POST")
	router.HandleFunc("/api/user/refresh-token", handlers.RefreshToken(db)).Methods("POST")

	// admin user routes
	router.Handle("/api/admin/user", middleware.Admin(handlers.CreateUser(db, true))).Methods("POST") // true to indicate that we'll create an admin user
	// router.HandleFunc("/api/admin/user", handlers.CreateUser(db, true)).Methods("POST") // just to create the first admin user

	// website routes
	router.Handle("/api/websites", middleware.Admin(handlers.GetWebsites(db))).Methods("GET")
	router.Handle("/api/websites/user/{id}", middleware.AdminOrOwner(handlers.GetUserWebsites(db))).Methods("GET")
	router.Handle("/api/website", middleware.AdminOrAuth(handlers.CreateWebsite(db))).Methods("POST")
	// router.Handle("/api/website/{domain}", middleware.AdminOrUserWebsite(db)(handlers.UpdateWebsite(db))).Methods("PUT")
	router.Handle("/api/website/{domain}", middleware.AdminOrUserWebsite(db)(handlers.DeleteWebsite(db))).Methods("DELETE")

	// dashboard routes
	router.Handle("/api/dashboard/top-stats/{domain}", middleware.AdminOrUserWebsite(db)(handlers.GetTopStats(db))).Methods("GET")
	router.Handle("/api/dashboard/pages/{domain}", middleware.AdminOrUserWebsite(db)(handlers.GetPages(db))).Methods("GET")
	router.Handle("/api/dashboard/referrers/{domain}", middleware.AdminOrUserWebsite(db)(handlers.GetReferrers(db))).Methods("GET")
	router.Handle("/api/dashboard/device-types/{domain}", middleware.AdminOrUserWebsite(db)(handlers.GetDeviceTypes(db))).Methods("GET")
	router.Handle("/api/dashboard/oses/{domain}", middleware.AdminOrUserWebsite(db)(handlers.GetOSes(db))).Methods("GET")
	router.Handle("/api/dashboard/browsers/{domain}", middleware.AdminOrUserWebsite(db)(handlers.GetBrowsers(db))).Methods("GET")
	router.Handle("/api/dashboard/languages/{domain}", middleware.AdminOrUserWebsite(db)(handlers.GetLanguages(db))).Methods("GET")
	router.Handle("/api/dashboard/countries/{domain}", middleware.AdminOrUserWebsite(db)(handlers.GetCountries(db))).Methods("GET")
	router.Handle("/api/dashboard/regions/{domain}", middleware.AdminOrUserWebsite(db)(handlers.GetRegions(db))).Methods("GET")
	router.Handle("/api/dashboard/cities/{domain}", middleware.AdminOrUserWebsite(db)(handlers.GetCities(db))).Methods("GET")
	router.Handle("/api/dashboard/utm_sources/{domain}", middleware.AdminOrUserWebsite(db)(handlers.GetUTMParameters(db, "utm_source"))).Methods("GET")
	router.Handle("/api/dashboard/utm_mediums/{domain}", middleware.AdminOrUserWebsite(db)(handlers.GetUTMParameters(db, "utm_medium"))).Methods("GET")
	router.Handle("/api/dashboard/utm_campaigns/{domain}", middleware.AdminOrUserWebsite(db)(handlers.GetUTMParameters(db, "utm_campaign"))).Methods("GET")
	router.Handle("/api/dashboard/utm_terms/{domain}", middleware.AdminOrUserWebsite(db)(handlers.GetUTMParameters(db, "utm_term"))).Methods("GET")
	router.Handle("/api/dashboard/utm_contents/{domain}", middleware.AdminOrUserWebsite(db)(handlers.GetUTMParameters(db, "utm_content"))).Methods("GET")

	// events routes
	router.Handle("/api/events/{domain}", middleware.AdminOrUserWebsite(db)(handlers.GetEvents(db))).Methods("GET")
	router.HandleFunc("/api/event", handlers.CreateEvent(db)).Methods("POST")

	// payment routes
	router.Handle("/api/payment/checkout", middleware.AdminOrAuth(handlers.CreateCheckoutSession(db))).Methods("POST")
	router.HandleFunc("/api/payment/webhook", handlers.StripeWebhook(db)).Methods("POST")

	// check limits
	router.Handle("/api/user/limits/{id}", middleware.AdminOrOwner(handlers.GetUserWebsiteLimits(db))).Methods("GET")

	// router.HandleFunc("/api/test-email", handlers.TestEmailSending()).Methods("POST")

	return router
}
