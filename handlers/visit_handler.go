package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"

	// "time"

	_ "github.com/lib/pq"
	"github.com/mileusna/useragent"
	"github.com/mvavassori/bare-analytics/models"
	"github.com/mvavassori/bare-analytics/utils"
)

func GetVisits(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		// Perform the SELECT query to get all visits
		rows, err := db.Query("SELECT id, timestamp, referrer, url, pathname, hash, user_agent, language, screen_width, screen_height, location, website_id FROM visits")
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
			err := rows.Scan(&visit.ID, &visit.Timestamp, &visit.Referrer, &visit.URL, &visit.Pathname, &visit.Hash, &visit.UserAgent, &visit.Language, &visit.ScreenWidth, &visit.ScreenHeight, &visit.Location, &visit.WebsiteID)
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

func GetVisit(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Extract the value of the 'id' variable from the URL path
		id, err := utils.ExtractIDFromURL(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// Perform the SELECT query to get the visit with the specified ID
		row := db.QueryRow("SELECT * FROM visits WHERE id = $1", id)

		// Create a Visit struct to hold the retrieved data
		var visit models.Visit
		// row.Scan copies the column values from the matched row into the provided variables, each field in the Visit struct corresponds to a column in the "visits" table.
		// It reads the values from the database row and populates the fields in the visit variable.
		err = row.Scan(
			&visit.ID,
			&visit.Timestamp,
			&visit.Referrer,
			&visit.URL,
			&visit.Pathname,
			&visit.Hash,
			&visit.UserAgent,
			&visit.Location,
			&visit.Language,
			&visit.ScreenWidth,
			&visit.ScreenHeight,
			&visit.WebsiteID,
		)

		if err == sql.ErrNoRows {
			http.Error(w, fmt.Sprintf("Visit with id %d doesn't exist", id), http.StatusNotFound)
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

// todo
func CreateVisit(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Create a VisitReceiver struct to hold the request data
		var visit models.VisitReceiver

		// Decode the JSON data from the request body into the VisitReceiver struct
		err := json.NewDecoder(r.Body).Decode(&visit) // The Decode function modifies the contents of the passed object based on the input JSON data. By passing a pointer, any changes made by Decode will directly update the original struct rather than creating a copy and updating that.
		if err != nil {
			log.Println("Error decoding input data", err)
			http.Error(w, "Invalid JSON format", http.StatusBadRequest)
			return
		}

		ua := useragent.Parse(visit.UserAgent)

		fmt.Println(ua.OS)
		fmt.Println(ua.Name)
		fmt.Println(ua.Mobile)
		fmt.Println(ua.Tablet)
		fmt.Println(ua.Desktop)
		fmt.Println(ua.Bot)

		url, err := url.Parse(visit.URL)
		if err != nil {
			log.Println("Error parsing URL", err)
			http.Error(w, "Invalid URL format", http.StatusBadRequest)
			return
		}
		domain := url.Hostname()

		fmt.Println(domain)
		fmt.Println(visit.Country)
		fmt.Println(visit.State)

		// // Look up the websiteId using the domain
		// var websiteId int
		// err = db.QueryRow("SELECT id FROM websites WHERE domain = $1", domain).Scan(&websiteId)
		// if err != nil {
		// 	log.Println("Error looking up websiteId", err)
		// 	http.Error(w, "Website not found", http.StatusNotFound)
		// 	return
		// }

		// // Perform the INSERT query to add the new visit to the database
		// insertQuery := `
		// 	INSERT INTO visits
		// 		(timestamp, referrer, url, pathname, hash, user_agent, location, language, screen_width, screen_height, website_id)
		// 	VALUES
		// 		($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11);
		// `
		// _, err = db.Exec(insertQuery,
		// 	time.Now(),
		// 	visit.Referrer,
		// 	visit.URL,
		// 	visit.Pathname,
		// 	visit.Hash,
		// 	visit.UserAgent,
		// 	visit.Location,
		// 	visit.Language,
		// 	visit.ScreenWidth,
		// 	visit.ScreenHeight,
		// 	websiteId,
		// )
		// if err != nil {
		// 	fmt.Println("Error inserting visit:", err)
		// 	return
		// }

		w.WriteHeader(http.StatusCreated)
	}
}

// todo update visit with new model.
func UpdateVisit(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Extract the value of the 'id' variable from the URL path
		id, err := utils.ExtractIDFromURL(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// Read the request body to get the updated visit data
		var updatedVisit models.VisitInsert
		err = json.NewDecoder(r.Body).Decode(&updatedVisit)
		if err != nil {
			log.Println("Error decoding JSON:", err)
			http.Error(w, "Invalid JSON format", http.StatusBadRequest)
			return
		}

		url, err := url.Parse(updatedVisit.URL)
		if err != nil {
			log.Println("Error parsing URL", err)
			http.Error(w, "Invalid URL format", http.StatusBadRequest)
			return
		}
		domain := url.Hostname()

		fmt.Println(domain)

		// Look up the websiteId using the domain
		var websiteId int
		err = db.QueryRow("SELECT id FROM websites WHERE domain = $1", domain).Scan(&websiteId)
		if err != nil {
			log.Println("Error looking up websiteId", err)
			http.Error(w, "Website not found", http.StatusNotFound)
			return
		}

		updateQuery := `
			UPDATE visits
			SET timestamp = $1, referrer = $2, url = $3, pathname = $4, hash = $5,
				user_agent = $6, location = $7, language = $8, screen_width = $9,
				screen_height = $10, website_id = $11
			WHERE id = $12
		`
		// Perform the UPDATE query to modify the visit with the specified ID
		result, err := db.Exec(updateQuery,
			updatedVisit.Timestamp,
			updatedVisit.Referrer,
			updatedVisit.URL,
			updatedVisit.Pathname,
			updatedVisit.Hash,
			updatedVisit.UserAgent,
			updatedVisit.Location,
			updatedVisit.Language,
			updatedVisit.ScreenWidth,
			updatedVisit.ScreenHeight,
			websiteId,
			id,
		)
		if err != nil {
			log.Println("Error updating visit:", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		// Check if the visit with the specified ID exists
		rowsAffected, err := result.RowsAffected()
		if err != nil {
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		if rowsAffected == 0 {
			http.Error(w, fmt.Sprintf("Visit with id %d doesn't exist", id), http.StatusNotFound)
			return
		}

		// Create a VisitUpdateResponse object to return in the response
		visitUpdateResponse := models.VisitUpdateResponse{
			VisitInsert: updatedVisit,
			ID:          int64(id),
			WebsiteID:   int(websiteId),
		}

		// Convert the visitResponse object to JSON
		jsonResponse, err := json.Marshal(visitUpdateResponse)
		if err != nil {
			log.Println("Error encoding JSON:", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		// Set the content type header and write the response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(jsonResponse)
	}
}

func DeleteVisit(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Extract the value of the 'id' variable from the URL path
		id, err := utils.ExtractIDFromURL(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// Perform the DELETE query to delete the visit with the specified ID
		result, err := db.Exec("DELETE FROM visits WHERE id = $1", id)
		if err != nil {
			log.Println("Error deleting visit:", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		// Check if the visit with the specified ID exists
		rowsAffected, err := result.RowsAffected()
		if err != nil {
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		if rowsAffected == 0 {
			http.Error(w, fmt.Sprintf("Visit with id %d doesn't exist", id), http.StatusNotFound)
			return
		}

		w.WriteHeader(http.StatusOK)
	}
}
