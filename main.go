package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/mvavassori/bare-analytics/handlers"
)

func main() {

	router := handlers.SetupRouter()

	port := 8080
	//  formats a string according to the specified format specifier -> :8080
	address := fmt.Sprintf(":%d", port)

	fmt.Printf("Server is listening on port %d...\n", port)
	err := http.ListenAndServe(address, router)
	if err != nil {
		log.Fatalf("Failed to start server: %v\n", err)
	}

}
