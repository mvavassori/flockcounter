package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/mvavassori/bare-analytics/middleware"
	"github.com/mvavassori/bare-analytics/models"
	"github.com/mvavassori/bare-analytics/utils"
)

func GetWebsites(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		rows, err := db.Query("SELECT id, domain, user_id FROM websites")
		if err != nil {
			log.Println("Error querying websites:", err)
			http.Error(w, "Error retrieving websites", http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		// A websites slice to hold the retrieved websites
		var websites []models.Website

		for rows.Next() {
			var website models.Website
			err := rows.Scan(&website.ID, &website.Domain, &website.UserID)
			if err != nil {
				log.Println("Error scanning website:", err)
				http.Error(w, "Error scanning website", http.StatusInternalServerError)
				return
			}
			websites = append(websites, website)

			if err := rows.Err(); err != nil {
				log.Println("Error iterating websites:", err)
				http.Error(w, "Error iterating websites", http.StatusInternalServerError)
				return
			}

		}
		// Marshal the slice of websites into JSON without the Valid key for each nullable field -> see models/website.go
		jsonResponse, err := json.Marshal(websites)
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

func GetUserWebsites(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, err := utils.ExtractIDFromURL(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		rows, err := db.Query("SELECT id, domain, user_id, created_at, updated_at FROM websites WHERE user_id = $1", userID)
		if err != nil {
			log.Println("Error querying user websites:", err)
			http.Error(w, "Error retrieving user websites", http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		var websites []models.Website

		for rows.Next() {
			var website models.Website
			err := rows.Scan(&website.ID, &website.Domain, &website.UserID, &website.CreatedAt, &website.UpdatedAt)
			if err != nil {
				log.Println("Error scanning user website:", err)
				http.Error(w, "Error scanning user website", http.StatusInternalServerError)
				return
			}
			websites = append(websites, website)

			if err := rows.Err(); err != nil {
				log.Println("Error iterating user websites:", err)
				http.Error(w, "Error iterating user websites", http.StatusInternalServerError)
				return
			}

		}
		jsonResponse, err := json.Marshal(websites)
		if err != nil {
			log.Println("Error encoding JSON:", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(jsonResponse)
	}
}

func GetWebsite(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// extract the value id from the url
		id, err := utils.ExtractIDFromURL(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// Perform the SELECT query to get the websit7e with the specified ID
		row := db.QueryRow("SELECT * FROM websites WHERE id = $1", id)

		// Creating a new instance of the Website struct from the models package and getting a pointer to it.
		website := &models.Website{}

		err = row.Scan(&website.ID, &website.Domain, &website.UserID, &website.CreatedAt, &website.UpdatedAt)
		if err == sql.ErrNoRows {
			http.Error(w, fmt.Sprintf("Website with id %d doesn't exist", id), http.StatusNotFound)
			return
		} else if err != nil {
			log.Println("Error retrieving website:", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		// Marshal the website data to JSON
		jsonResponse, err := json.Marshal(website)
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

func CreateWebsite(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Get the userId from the context
		userId, ok := r.Context().Value(middleware.UserIdKey).(int)
		if !ok {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		// Decode request JSON into WebsiteReceiver
		var websiteReceiver models.WebsiteReceiver
		if err := json.NewDecoder(r.Body).Decode(&websiteReceiver); err != nil {
			log.Println("Error decoding JSON:", err)
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}

		// Check if a website with the same domain already exists in the database
		var existingDomain string
		err := db.QueryRow(`
			SELECT domain
			FROM websites
			WHERE domain = $1
		`, websiteReceiver.Domain).Scan(&existingDomain)

		if err == nil {
			// If a website with the same domain already exists, return a conflict error
			http.Error(w, "Domain already exists", http.StatusConflict)
			return
		} else if err != sql.ErrNoRows {
			// If there was an error executing the query, return an internal server error
			log.Println("Error checking for existing domain:", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		// Map data to WebsiteInsert struct and set timestamps
		websiteInsert := models.WebsiteInsert{
			Domain:    websiteReceiver.Domain,
			UserID:    userId,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		// Insert the website into the database
		_, err = db.Exec(
			`INSERT INTO websites (domain, user_id, created_at, updated_at) VALUES ($1, $2, $3, $4)`,
			websiteInsert.Domain, websiteInsert.UserID, websiteInsert.CreatedAt, websiteInsert.UpdatedAt,
		)
		if err != nil {
			log.Println("Error inserting website:", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusCreated)
		fmt.Fprintf(w, "Website created successfully")
	}
}

func DeleteWebsite(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Extract the domain from the url
		domain, err := utils.ExtractDomainFromURL(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// Delete the website from the database
		result, err := db.Exec("DELETE FROM websites WHERE domain = $1", domain)
		if err != nil {
			log.Println("Error deleting website:", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		rowsAffected, err := result.RowsAffected()
		if err != nil {
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		if rowsAffected == 0 {
			http.Error(w, fmt.Sprintf("Website with domain %s doesn't exist", domain), http.StatusNotFound)
			return
		}

		// return a 200 response
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "Website deleted successfully")
	}
}

// // todo fix this. IT DOESN'T WORK CORRECTLY AT THE MOMENT
// // UpdateWebsite updates an existing website in the database
// func UpdateWebsite(db *sql.DB) http.HandlerFunc {
// 	return func(w http.ResponseWriter, r *http.Request) {
// 		// Extract website ID from URL
// 		id, err := utils.ExtractIDFromURL(r)
// 		if err != nil {
// 			http.Error(w, err.Error(), http.StatusBadRequest)
// 			return
// 		}

// 		// Decode the request body into a WebsiteInsert struct
// 		var websiteUpdate models.WebsiteReceiver
// 		err = json.NewDecoder(r.Body).Decode(&websiteUpdate)
// 		if err != nil {
// 			http.Error(w, "Invalid JSON in request body", http.StatusBadRequest)
// 			return
// 		}

// 		// Check if website exists and if the user has the right permissions (handled by AdminOrUserWebsite middleware)
// 		var existingWebsite models.Website
// 		err = db.QueryRow(`SELECT id, domain, user_id FROM websites WHERE id = $1`, id).Scan(&existingWebsite.ID, &existingWebsite.Domain, &existingWebsite.UserID)
// 		if err == sql.ErrNoRows {
// 			http.Error(w, "Website not found", http.StatusNotFound)
// 			return
// 		} else if err != nil {
// 			log.Println("Error fetching website:", err)
// 			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
// 			return
// 		}

// 		// Prepare WebsiteInsert struct with updated values
// 		websiteInsert := models.WebsiteInsert{
// 			Domain:    websiteUpdate.Domain,
// 			UserID:    int(existingWebsite.UserID.Int64), // keep existing user_id if not changed
// 			UpdatedAt: time.Now(),
// 		}

// 		// Perform the update query
// 		updateQuery := "UPDATE websites SET domain = $1, user_id = $2, updated_at = $3 WHERE id = $4"
// 		result, err := db.Exec(updateQuery, websiteInsert.Domain, websiteInsert.UserID, websiteInsert.UpdatedAt, id)
// 		if err != nil {
// 			log.Println("Error updating website:", err)
// 			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
// 			return
// 		}

// 		rowsAffected, err := result.RowsAffected()
// 		if err != nil {
// 			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
// 			return
// 		}

// 		if rowsAffected == 0 {
// 			http.Error(w, "No changes made to the website", http.StatusNotModified)
// 			return
// 		}

// 		// Prepare response
// 		websiteUpdateResponse := models.WebsiteUpdateResponse{
// 			ID:     int64(id),
// 			Domain: websiteInsert.Domain,
// 			UserID: websiteInsert.UserID,
// 		}

// 		// Send JSON response
// 		w.Header().Set("Content-Type", "application/json")
// 		w.WriteHeader(http.StatusOK)
// 		if err := json.NewEncoder(w).Encode(websiteUpdateResponse); err != nil {
// 			log.Println("Error encoding response:", err)
// 			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
// 		}
// 	}
// }
