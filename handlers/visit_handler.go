package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
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

		// Perform the SELECT query to get the visit with the specified ID
		row := db.QueryRow("SELECT * FROM visits WHERE id = $1", id)

		// Create a Visit struct to hold the retrieved data
		var visit models.Visit
		// row.Scan copies the column values from the matched row into the provided variables, each field in the Visit struct corresponds to a column in the "visits" table.
		// It reads the values from the database row and populates the fields in the visit variable.
		err := row.Scan(
			&visit.ID,
			&visit.Timestamp,
			&visit.Referrer,
			&visit.URL,
			&visit.Pathname,
			&visit.Hash,
			&visit.UserAgent,
			&visit.Language,
			&visit.ScreenWidth,
			&visit.ScreenHeight,
			&visit.Location,
		)

		if err == sql.ErrNoRows {
			http.Error(w, fmt.Sprintf("Visit with ID %s not found", id), http.StatusNotFound)
			return
		} else if err != nil {
			log.Println("Error retrieving visit:", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		// Marshal the visit data to JSON
		jsonResponse, err := json.Marshal(visit)
		if err != nil {
			log.Println("Error encoding JSON:", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		// Set response headers and write the JSON response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(jsonResponse)
	}
}

func GetVisits(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		// Perform the SELECT query to get all visits
		rows, err := db.Query("SELECT id, timestamp, referrer, url, pathname, hash, userAgent, language, screen_width, screen_height, location FROM visits")
		if err != nil {
			log.Println("Error querying visits:", err)
			http.Error(w, "Error querying visits", http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		// A visits slice to hold the retrieved visits
		var visits []models.Visit

		// Loop through rows, using Scan to assign column data to struct fields.
		for rows.Next() {
			var visit models.Visit
			err := rows.Scan(&visit.ID, &visit.Timestamp, &visit.Referrer, &visit.URL, &visit.Pathname, &visit.Hash, &visit.UserAgent, &visit.Language, &visit.ScreenWidth, &visit.ScreenHeight, &visit.Location)
			if err != nil {
				log.Println("Error scanning visit:", err)
				http.Error(w, "Error scanning visit", http.StatusInternalServerError)
				return
			}
			visits = append(visits, visit)
		}

		if err := rows.Err(); err != nil {
			log.Println("Error iterating visits:", err)
			http.Error(w, "Error iterating visits", http.StatusInternalServerError)
			return
		}

		// Now, 'visits' contains all the retrieved visits
		// You can encode and send the visits as a JSON response
		jsonResponse, err := json.Marshal(visits)
		if err != nil {
			log.Println("Error encoding JSON:", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		// Set response headers and write the JSON response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(jsonResponse)
	}
}

func CreateVisit(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Create a VisitInsert struct to hold the request data
		var visit models.VisitInsert

		// Decode the JSON data from the request body into the VisitInsert struct
		err := json.NewDecoder(r.Body).Decode(&visit) // The Decode function modifies the contents of the passed object based on the input JSON data. By passing a pointer, any changes made by Decode will directly update the original struct rather than creating a copy and updating that.
		if err != nil {
			log.Println("Error decoding input data", err)
			http.Error(w, "Invalid JSON format", http.StatusBadRequest)
			return
		}

		// Perform the INSERT query to add the new visit to the database
		insertQuery := `
			INSERT INTO visits 
				(timestamp, referrer, url, pathname, hash, userAgent, language, screen_width, screen_height, location) 
			VALUES 
				($1, $2, $3, $4, $5, $6, $7, $8, $9, $10);
		`
		_, err = db.Exec(insertQuery,
			time.Now(),
			visit.Referrer,
			visit.URL,
			visit.Pathname,
			visit.Hash,
			visit.UserAgent,
			visit.Language,
			visit.ScreenWidth,
			visit.ScreenHeight,
			visit.Location,
		)
		if err != nil {
			fmt.Println("Error inserting visit:", err)
			return
		}

		fmt.Println("Visit inserted successfully")
	}
}

func UpdateVisit(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Extract the value of the 'id' variable from the URL path
		vars := mux.Vars(r)
		id, ok := vars["id"]

		if !ok {
			http.Error(w, "ID not provided in the URL", http.StatusBadRequest)
			return
		}

		// Read the request body to get the updated visit data
		var updatedVisit models.Visit
		err := json.NewDecoder(r.Body).Decode(&updatedVisit)
		if err != nil {
			log.Println("Error decoding JSON:", err)
			http.Error(w, "Invalid JSON format", http.StatusBadRequest)
			return
		}

		updateQuery := `
			UPDATE visits
			SET timestamp = $1, referrer = $2, url = $3, pathname = $4, hash = $5,
				userAgent = $6, language = $7, screen_width = $8, screen_height = $9, location = $10
			WHERE id = $11
		`
		// Perform the UPDATE query to modify the visit with the specified ID
		_, err = db.Exec(updateQuery,
			updatedVisit.Timestamp,
			updatedVisit.Referrer,
			updatedVisit.URL,
			updatedVisit.Pathname,
			updatedVisit.Hash,
			updatedVisit.UserAgent,
			updatedVisit.Language,
			updatedVisit.ScreenWidth,
			updatedVisit.ScreenHeight,
			updatedVisit.Location,
			id,
		)
		if err != nil {
			log.Println("Error updating visit:", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		fmt.Fprintf(w, "Visit with ID %s updated successfully", id)

	}
}

func DeleteVisit(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Extract the value of the 'id' variable from the URL path
		vars := mux.Vars(r)
		id, ok := vars["id"]

		if !ok {
			http.Error(w, "ID not provided in the URL", http.StatusBadRequest)
			return
		}

		// Perform the DELETE query to delete the visit with the specified ID
		_, err := db.Exec("DELETE FROM visits WHERE id = $1", id)
		if err != nil {
			log.Println("Error deleting visit:", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		fmt.Fprintf(w, "Visit with ID %s deleted successfully", id)
	}
}
