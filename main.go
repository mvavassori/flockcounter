package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/handlers"
	"github.com/mvavassori/bare-analytics/db"
)

func main() {
	// db initialization
	db, err := db.CreateDBConnection()
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// router
	router := SetupRouter(db)

	port := 8080
	address := fmt.Sprintf(":%d", port) // :8080

	log.Printf("Server is listening on port %d...\n", port)

	err = http.ListenAndServe(address, handlers.CORS( // cors config
		handlers.AllowedOrigins([]string{"*"}),
		handlers.AllowedMethods([]string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}),
		handlers.AllowedHeaders([]string{"Content-Type", "Authorization"}),
	)(router))
	if err != nil {
		log.Fatalf("Failed to start server: %v\n", err)
	}

}

// Explanation to novices:
// 1. The main.go file imports the db package and initializes the database connection using the CreateDBConnection function from the github.com/mvavassori/bare-analytics/db package.
// 2. The main.go file calls the SetupRouter function from the github.com/mvavassori/bare-analytics/handlers package, passing the db connection as an argument.
// 3. The SetupRouter function in the handlers/routes.go file creates a new Gorilla Mux router and sets up the API routes with their corresponding handlers. It passes the db connection to the route handlers (defined for exmaple in handlers/visit_handler.go).
// 4. The route handlers in handlers/visit_handler.go use the db connection to interact with the Postgres database and handle incoming requests.
