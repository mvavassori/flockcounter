package handlers

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/mvavassori/bare-analytics/middleware"
	"github.com/mvavassori/bare-analytics/models"
	"github.com/mvavassori/bare-analytics/utils"
)

func GetWebsites(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Extract limit and offset from query string
		limit, err := strconv.Atoi(r.URL.Query().Get("limit"))
		if err != nil || limit <= 0 {
			limit = 10 // default limit
		}
		offset, err := strconv.Atoi(r.URL.Query().Get("offset"))
		if err != nil || offset < 0 {
			offset = 0 // default offset
		}

		// Prepare the SQL query with LIMIT and OFFSET for pagination
		query := `
			SELECT id, domain, user_id 
			FROM websites 
			LIMIT $1 OFFSET $2
		`

		rows, err := db.Query(query, limit, offset)
		if err != nil {
			log.Println("Error querying websites:", err)
			http.Error(w, "Error retrieving websites", http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		// A slice to hold the retrieved websites
		var websites []models.Website

		// Iterate through the result rows and scan into the website struct
		for rows.Next() {
			var website models.Website
			err := rows.Scan(&website.ID, &website.Domain, &website.UserID)
			if err != nil {
				log.Println("Error scanning website:", err)
				http.Error(w, "Error scanning website", http.StatusInternalServerError)
				return
			}
			websites = append(websites, website)
		}

		// Check if there were any errors during row iteration
		if err := rows.Err(); err != nil {
			log.Println("Error iterating websites:", err)
			http.Error(w, "Error iterating websites", http.StatusInternalServerError)
			return
		}

		// Marshal the slice of websites into JSON
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
		// Extract userID from the URL
		userID, err := utils.ExtractIDFromURL(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// Check if the user exists in the database
		var exists bool
		err = db.QueryRow("SELECT EXISTS(SELECT 1 FROM users WHERE id = $1)", userID).Scan(&exists)
		if err != nil {
			log.Println("Error checking if user exists:", err)
			http.Error(w, "Error checking user existence", http.StatusInternalServerError)
			return
		}
		if !exists {
			http.Error(w, "User not found", http.StatusNotFound)
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

func CreateWebsite(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Get the userId from the context
		userId, ok := r.Context().Value(middleware.UserIdKey).(int)
		if !ok {
			utils.WriteErrorResponse(w, http.StatusInternalServerError, errors.New("internal Server Error"))
			return
		}

		// Decode request JSON into WebsiteReceiver
		var websiteReceiver models.WebsiteReceiver
		if err := json.NewDecoder(r.Body).Decode(&websiteReceiver); err != nil {
			log.Println("Error decoding JSON:", err)
			utils.WriteErrorResponse(w, http.StatusBadRequest, errors.New("invalid request format"))
			return
		}
		// Check that URL is not empty
		if websiteReceiver.URL == "" {
			utils.WriteErrorResponse(w, http.StatusBadRequest, errors.New("URL cannot be empty"))
			return
		}

		// Parse and validate the URL
		parsedURL, err := url.Parse(websiteReceiver.URL)
		if err != nil || (parsedURL.Scheme != "http" && parsedURL.Scheme != "https") || parsedURL.Hostname() == "" {
			log.Println("Invalid URL format:", websiteReceiver.URL)
			utils.WriteErrorResponse(w, http.StatusBadRequest, errors.New("invalid URL format"))
			return
		}

		domain := parsedURL.Hostname()

		// Check if a website with the same domain already exists in the database
		var existingDomain string
		err = db.QueryRow(`
			SELECT domain
			FROM websites
			WHERE domain = $1
		`, domain).Scan(&existingDomain)

		if err == nil {
			// If a website with the same domain already exists, return a conflict error
			utils.WriteErrorResponse(w, http.StatusConflict, errors.New("domain already exists"))
			return
		} else if err != sql.ErrNoRows {
			// If there was an error executing the query, return an internal server error
			log.Println("Error checking for existing domain:", err)
			utils.WriteErrorResponse(w, http.StatusInternalServerError, errors.New("internal server error"))
			return
		}

		// Map data to WebsiteInsert struct and set timestamps
		websiteInsert := models.WebsiteInsert{
			Domain:    domain,
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
			utils.WriteErrorResponse(w, http.StatusInternalServerError, errors.New("internal server error"))
			return
		}

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{
			"domain":  domain,
			"message": "Website created successfully",
		})
	}
}

func DeleteWebsite(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Extract the domain from the URL
		domain, err := utils.ExtractDomainFromURL(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// Begin a transaction to ensure atomicity
		tx, err := db.Begin()
		if err != nil {
			log.Println("Error beginning transaction:", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		// Ensure transaction rollback in case of an error
		defer func() {
			if err != nil {
				tx.Rollback()
			}
		}()

		// Delete associated visits from the visits table (ignoring row count)
		_, err = tx.Exec("DELETE FROM visits WHERE website_domain = $1", domain)
		if err != nil {
			log.Println("Error deleting visits:", err)
			http.Error(w, "Error deleting visits", http.StatusInternalServerError)
			return
		}

		// Delete the website from the websites table
		result, err := tx.Exec("DELETE FROM websites WHERE domain = $1", domain)
		if err != nil {
			log.Println("Error deleting website:", err)
			http.Error(w, "Error deleting website", http.StatusInternalServerError)
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

		// Commit the transaction if both deletes are successful
		if err = tx.Commit(); err != nil {
			log.Println("Error committing transaction:", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "Website and associated visits deleted successfully")
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
