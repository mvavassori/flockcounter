package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/mvavassori/bare-analytics/models"
	"github.com/mvavassori/bare-analytics/utils"
)

// GetWebsites retrieves all websites from the database
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

// GetWebsite retrieves a single website by ID from the database
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

		err = row.Scan(&website.ID, &website.Domain, &website.UserID)
		if err == sql.ErrNoRows {
			http.Error(w, fmt.Sprintf("Website with ID %s not found", strconv.Itoa(id)), http.StatusNotFound)
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

// CreateWebsite creates a new website in the database
func CreateWebsite(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Create a WbsiteInsert struct to hold the request body data
		var website models.WebsiteInsert

		// Decide the JSON data from the request body into the WebsiteInsert struct
		err := json.NewDecoder(r.Body).Decode(&website)
		if err != nil {
			log.Println("Error decoding JSON:", err)
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}

		// Insert the website into the database
		_, err = db.Exec(`
			INSERT INTO websites (domain, user_id)
			VALUES ($1, $2)
		`, website.Domain, website.UserID)

		if err != nil {
			log.Println("Error inserting website:", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusCreated)
	}
}

// UpdateWebsite updates an existing website in the database
func UpdateWebsite(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := utils.ExtractIDFromURL(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		var website models.WebsiteInsert
		// Decode the request body into the website variable
		err = json.NewDecoder(r.Body).Decode(&website)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// Update the website in the database
		updateQuery := "UPDATE websites SET domain = $1, user_id = $2 WHERE id = $3"
		_, err = db.Exec(updateQuery, website.Domain, website.UserID, id)
		if err != nil {
			log.Println("Error updating website:", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Create a WebsiteResponse object to return in the response
		websiteUpdateResponse := models.WebsiteUpdateResponse{
			ID:     int64(id),
			Domain: website.Domain,
			UserID: website.UserID,
		}

		// Convert the website object to JSON
		jsonResponse, err := json.Marshal(websiteUpdateResponse)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Set the content type header and write the response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(jsonResponse)

	}
}

// DeleteWebsite deletes a website from the database
func DeleteWebsite(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Extract the id from the url
		id, err := utils.ExtractIDFromURL(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// Delete the website from the database
		_, err = db.Exec("DELETE FROM websites WHERE id = $1", id)
		if err != nil {
			log.Println("Error deleting website:", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// return a 200 response
		w.WriteHeader(http.StatusOK)
	}
}
