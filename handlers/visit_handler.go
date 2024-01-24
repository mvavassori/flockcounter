package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	_ "github.com/lib/pq"
	"github.com/mvavassori/bare-analytics/models"

	"github.com/gorilla/mux"
)

func GetVisit(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Extract the value of the 'id' variable from the URL path
		vars := mux.Vars(r)
		id, ok := vars["id"]

		if !ok {
			http.Error(w, "ID not provided in the URL", http.StatusBadRequest)
			return
		}
		// Handle the specific visit with the provided ID
		// ...

		fmt.Fprintf(w, "Getting visit with ID %s", id)
	}
}

func GetVisits(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Handle getting all visits
		// ...
		fmt.Fprintf(w, "Getting all visits")
	}
}

func CreateVisit(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		data, err := io.ReadAll(r.Body)
		if err != nil {
			log.Println("Error reading the request body", err)
			http.Error(w, "Error reading the request body", http.StatusBadRequest)
			return
		}

		fmt.Println("raw json data:", string(data))

		// convert JSON data to a Struct
		var visit models.Visit
		err = json.Unmarshal([]byte(data), &visit)
		if err != nil {
			log.Println("Error unmarshaling JSON data:", err)
			http.Error(w, "Error unmarshaling JSON data", http.StatusBadRequest)
			return
		}

		insertQuery := "INSERT INTO visits (timestamp, referrer, url, pathname, hash, userAgent, language, screen_width, screen_height, location) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10);"

		_, err = db.Exec(insertQuery, time.Now(), visit.Referrer, visit.URL, visit.Pathname, visit.Hash, visit.UserAgent, visit.Language, visit.ScreenWidth, visit.ScreenHeight, visit.Location)
		if err != nil {
			fmt.Println("Error inserting visit:", err)
			return
		}

		fmt.Println("Visit inserted successfully!") // result is the id of the inserted row
	}
}
