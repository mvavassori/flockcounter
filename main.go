package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/gorilla/handlers"
	"github.com/mvavassori/flockcounter/db"
)

func main() {
	// Connect to Postgres
	postgresDB, err := db.CreatePostgresConnection()
	if err != nil {
		log.Fatal(err)
	}
	defer postgresDB.Close()

	// Connect to GeoIP
	geoipDB, err := db.CreateGeoIPConnection()
	if err != nil {
		log.Fatal(err)
	}
	defer geoipDB.Close()

	// router
	router := SetupRouter(postgresDB, geoipDB)

	port := 8080
	address := fmt.Sprintf(":%d", port) // :8080

	log.Printf("Server is listening on port %d...\n", port)

	if os.Getenv("ENV") == "development" {
		err = http.ListenAndServe(address, handlers.CORS( // cors config for development
			handlers.AllowedOrigins([]string{"*"}),
			handlers.AllowedMethods([]string{"GET", "POST", "PUT", "DELETE", "OPTIONS", "PATCH"}),
			handlers.AllowedHeaders([]string{"Content-Type", "Authorization"}),
		)(router))
	} else {
		err = http.ListenAndServe(address, router) // nginx will handle cors
	}

	if err != nil {
		log.Fatalf("Failed to start server: %v\n", err)
	}

}

// Explanation to novices:
// 1. The main.go file imports the db package and initializes the database connection using the CreateDBConnection function from the github.com/mvavassori/flockcounter/db package.
// 2. The main.go file calls the SetupRouter function from the github.com/mvavassori/flockcounter/handlers package, passing the db connection as an argument.
// 3. The SetupRouter function in the handlers/routes.go file creates a new Gorilla Mux router and sets up the API routes with their corresponding handlers. It passes the db connection to the route handlers (defined for exmaple in handlers/visit_handler.go).
// 4. The route handlers in handlers/visit_handler.go use the db connection to interact with the Postgres database and handle incoming requests.
