package handlers

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"time"

	// "github.com/mvavassori/bare-analytics/models"
	"github.com/mvavassori/bare-analytics/utils"
)

func GetTopStats(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Extract the id from the url
		id, err := utils.ExtractIDFromURL(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// Extract start and end dates from the request query parameters
		startDate := r.URL.Query().Get("startDate")
		endDate := r.URL.Query().Get("endDate")

		// Convert the dates to a format suitable for your database
		// This is just an example, adjust according to your needs
		start, err := time.Parse("2006-01-02 15:04:05.999", startDate)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		end, err := time.Parse("2006-01-02 15:04:05.999", endDate)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// Query the database for statistics
		// This is just a placeholder, replace with your actual queries
		stats, err := db.Query("SELECT COUNT(*) FROM visits WHERE website_id = $1 AND timestamp BETWEEN $2 AND $3", id, start, end)
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

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(jsonStats)
	}
}

func GetPages(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Extract the id from the url
		id, err := utils.ExtractIDFromURL(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// Extract start and end dates from the request query parameters
		startDate := r.URL.Query().Get("startDate")
		endDate := r.URL.Query().Get("endDate")

		// Convert the dates to a format suitable for your database
		start, err := time.Parse("2006-01-02 15:04:05.999", startDate)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		end, err := time.Parse("2006-01-02 15:04:05.999", endDate)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// Query the database for statistics
		stats, err := db.Query("SELECT path, COUNT(*) FROM visits WHERE website_id = $1 AND timestamp BETWEEN $2 AND $3 GROUP BY pathname ORDER BY COUNT(*) DESC LIMIT 10", id, start, end)
		if err != nil {
			log.Println("Error getting website statistics:", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Convert the statistics to JSON
		defer stats.Close() // Close the result set after we're done with it
		var path string
		var count int
		var paths []string
		var counts []int
		for stats.Next() {
			err = stats.Scan(&path, &count)
			if err != nil {
				log.Println("Error scanning statistics:", err)
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			paths = append(paths, path)
			counts = append(counts, count)
		}

		jsonStats, err := json.Marshal(map[string]interface{}{
			"paths":  paths,
			"counts": counts,
		})
		if err != nil {
			log.Println("Error marshalling statistics:", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(jsonStats)
	}
}
