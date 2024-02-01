package handlers

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"

	// "github.com/mvavassori/bare-analytics/models"
	"github.com/mvavassori/bare-analytics/utils"
)

// GetWebsiteStatistics returns statistics for a given website
func GetWebsiteStatistics(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Extract the id from the url
		id, err := utils.ExtractIDFromURL(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// Query the database for statistics
		// This is just a placeholder, replace with your actual queries
		stats, err := db.Query("SELECT COUNT(*) FROM visits WHERE website_id = $1", id)
		if err != nil {
			log.Println("Error getting website statistics:", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Convert the statistics to JSON
		defer stats.Close() // Close the result set after we're done with it
		var count int
		if stats.Next() {
			err = stats.Scan(&count)
			if err != nil {
				log.Println("Error scanning statistics:", err)
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}

		jsonStats, err := json.Marshal(count)
		if err != nil {
			log.Println("Error marshalling statistics:", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Return the statistics
		w.Header().Set("Content-Type", "application/json")
		w.Write(jsonStats)
	}
}
