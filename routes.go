package main

import (
	"database/sql"

	"github.com/gorilla/mux"
	"github.com/mvavassori/flockcounter/handlers"
	"github.com/mvavassori/flockcounter/middleware"
	"github.com/oschwald/geoip2-golang"
)

func SetupRouter(postgresDB *sql.DB, geoipDB *geoip2.Reader) *mux.Router {

	router := mux.NewRouter()

	// visit routes
	router.Handle("/api/visits", middleware.Admin(handlers.GetVisits(postgresDB))).Methods("GET")
	router.HandleFunc("/api/visit", handlers.CreateVisit(postgresDB, geoipDB)).Methods("POST")
	router.Handle("/api/visit/{id}", middleware.Admin(handlers.DeleteVisit(postgresDB))).Methods("DELETE")

	// user routes
	router.Handle("/api/users", middleware.Admin(handlers.GetUsers(postgresDB))).Methods("GET")
	router.Handle("/api/user/{id}", middleware.AdminOrOwner(handlers.GetUser(postgresDB))).Methods("GET")
	router.HandleFunc("/api/user", handlers.CreateUser(postgresDB, false)).Methods("POST") // false to indicate that we'll create a regular user
	router.Handle("/api/user/{id}", middleware.AdminOrOwner(handlers.UpdateUser(postgresDB))).Methods("PATCH")
	router.Handle("/api/user/{id}", middleware.AdminOrOwner(handlers.DeleteUser(postgresDB))).Methods("DELETE")

	// auth routes
	router.HandleFunc("/api/user/login", handlers.Login(postgresDB)).Methods("POST")
	router.HandleFunc("/api/user/refresh-token", handlers.RefreshToken(postgresDB)).Methods("POST")
	router.Handle("/api/user/change-password/{id}", middleware.AdminOrOwner(handlers.ChangePassword(postgresDB))).Methods("PATCH")

	// admin user routes
	router.Handle("/api/admin/user", middleware.Admin(handlers.CreateUser(postgresDB, true))).Methods("POST") // true to indicate that we'll create an admin user
	// router.HandleFunc("/api/admin/user", handlers.CreateUser(postgresDB, true)).Methods("POST") // just to create the first admin user

	// website routes
	router.Handle("/api/websites", middleware.Admin(handlers.GetWebsites(postgresDB))).Methods("GET")
	router.Handle("/api/websites/user/{id}", middleware.AdminOrOwner(handlers.GetUserWebsites(postgresDB))).Methods("GET")
	router.Handle("/api/website", middleware.AdminOrAuth(handlers.CreateWebsite(postgresDB))).Methods("POST")
	// router.Handle("/api/website/{domain}", middleware.AdminOrUserWebsite(postgresDB)(handlers.UpdateWebsite(postgresDB))).Methods("PUT")
	router.Handle("/api/website/{domain}", middleware.AdminOrUserWebsite(postgresDB)(handlers.DeleteWebsite(postgresDB))).Methods("DELETE")

	// dashboard routes
	router.Handle("/api/dashboard/top-stats/{domain}", middleware.AdminOrUserWebsite(postgresDB)(handlers.GetTopStats(postgresDB))).Methods("GET")
	router.Handle("/api/dashboard/pages/{domain}", middleware.AdminOrUserWebsite(postgresDB)(handlers.GetPages(postgresDB))).Methods("GET")
	router.Handle("/api/dashboard/referrers/{domain}", middleware.AdminOrUserWebsite(postgresDB)(handlers.GetReferrers(postgresDB))).Methods("GET")
	router.Handle("/api/dashboard/device-types/{domain}", middleware.AdminOrUserWebsite(postgresDB)(handlers.GetDeviceTypes(postgresDB))).Methods("GET")
	router.Handle("/api/dashboard/oses/{domain}", middleware.AdminOrUserWebsite(postgresDB)(handlers.GetOSes(postgresDB))).Methods("GET")
	router.Handle("/api/dashboard/browsers/{domain}", middleware.AdminOrUserWebsite(postgresDB)(handlers.GetBrowsers(postgresDB))).Methods("GET")
	router.Handle("/api/dashboard/languages/{domain}", middleware.AdminOrUserWebsite(postgresDB)(handlers.GetLanguages(postgresDB))).Methods("GET")
	router.Handle("/api/dashboard/countries/{domain}", middleware.AdminOrUserWebsite(postgresDB)(handlers.GetCountries(postgresDB))).Methods("GET")
	router.Handle("/api/dashboard/regions/{domain}", middleware.AdminOrUserWebsite(postgresDB)(handlers.GetRegions(postgresDB))).Methods("GET")
	router.Handle("/api/dashboard/cities/{domain}", middleware.AdminOrUserWebsite(postgresDB)(handlers.GetCities(postgresDB))).Methods("GET")
	router.Handle("/api/dashboard/utm_sources/{domain}", middleware.AdminOrUserWebsite(postgresDB)(handlers.GetUTMParameters(postgresDB, "utm_source"))).Methods("GET")
	router.Handle("/api/dashboard/utm_mediums/{domain}", middleware.AdminOrUserWebsite(postgresDB)(handlers.GetUTMParameters(postgresDB, "utm_medium"))).Methods("GET")
	router.Handle("/api/dashboard/utm_campaigns/{domain}", middleware.AdminOrUserWebsite(postgresDB)(handlers.GetUTMParameters(postgresDB, "utm_campaign"))).Methods("GET")
	router.Handle("/api/dashboard/utm_terms/{domain}", middleware.AdminOrUserWebsite(postgresDB)(handlers.GetUTMParameters(postgresDB, "utm_term"))).Methods("GET")
	router.Handle("/api/dashboard/utm_contents/{domain}", middleware.AdminOrUserWebsite(postgresDB)(handlers.GetUTMParameters(postgresDB, "utm_content"))).Methods("GET")
	router.Handle("/api/dashboard/live-pageviews/{domain}", middleware.AdminOrUserWebsite(postgresDB)(handlers.GetLivePageViews(postgresDB))).Methods("GET")

	// events routes
	router.Handle("/api/events/{domain}", middleware.AdminOrUserWebsite(postgresDB)(handlers.GetEvents(postgresDB))).Methods("GET")
	router.HandleFunc("/api/event", handlers.CreateEvent(postgresDB, geoipDB)).Methods("POST")

	// payment routes
	router.Handle("/api/payment/checkout", middleware.AdminOrAuth(handlers.CreateCheckoutSession(postgresDB))).Methods("POST")
	router.HandleFunc("/api/payment/webhook", handlers.StripeWebhook(postgresDB)).Methods("POST")

	// check limits
	router.Handle("/api/user/limits/{id}", middleware.AdminOrOwner(handlers.GetUserWebsiteLimits(postgresDB))).Methods("GET")

	// router.HandleFunc("/api/test-email", handlers.TestEmailSending()).Methods("POST")

	return router
}
