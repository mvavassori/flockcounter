package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"sync"
	"time"

	"github.com/mvavassori/bare-analytics/utils"
)

func GetTopStats(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Extract the domain from the url
		domain, err := utils.ExtractDomainFromURL(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// Check if the website exists
		var exists bool
		err = db.QueryRow("SELECT EXISTS(SELECT 1 FROM websites WHERE domain = $1)", domain).Scan(&exists)
		if err != nil {
			log.Println("Error checking website existence:", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		if !exists {
			http.Error(w, fmt.Sprintf("Website with domain %s doesn't exist", domain), http.StatusNotFound)
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

		var wg sync.WaitGroup
		var mu sync.Mutex

		var totalVisits []map[string]interface{}
		var uniqueVisitors []map[string]interface{}
		var averageVisitDuration []map[string]interface{}

		// Goroutine 1: Total visits
		wg.Add(1)
		go func() {
			defer wg.Done()
			rows, err := db.Query(`
                SELECT DATE_TRUNC('day', timestamp) AS date, COUNT(*) AS count
                FROM visits
                WHERE website_domain = $1 AND timestamp BETWEEN $2 AND $3
                GROUP BY date
                ORDER BY date ASC`, domain, start, end)
			if err != nil {
				log.Println("Error getting total visits:", err)
				return
			}
			defer rows.Close()

			var dataPoints []map[string]interface{}
			for rows.Next() {
				var date time.Time
				var count int
				err = rows.Scan(&date, &count)
				if err != nil {
					log.Println("Error scanning total visits:", err)
					return
				}
				dataPoints = append(dataPoints, map[string]interface{}{
					"date":  date.Format("2006-01-02"),
					"count": count,
				})
			}
			mu.Lock()
			totalVisits = dataPoints
			mu.Unlock()
		}()

		// Goroutine 2: Unique visitors
		wg.Add(1)
		go func() {
			defer wg.Done()
			rows, err := db.Query(`
			SELECT
				DATE_TRUNC('day', timestamp) AS date,
				COUNT(*) AS count
			FROM visits
			WHERE
				website_domain = $1
				AND timestamp BETWEEN $2 AND $3
				AND is_unique = true
			GROUP BY date
			ORDER BY date ASC`, domain, start, end)
			if err != nil {
				log.Println("Error getting unique visitors:", err)
				return
			}
			defer rows.Close()

			var dataPoints []map[string]interface{}
			for rows.Next() {
				var date time.Time
				var count int
				err = rows.Scan(&date, &count)
				if err != nil {
					log.Println("Error scanning unique visitors:", err)
					return
				}
				dataPoints = append(dataPoints, map[string]interface{}{
					"date":  date.Format("2006-01-02"),
					"count": count,
				})
			}
			mu.Lock()
			uniqueVisitors = dataPoints
			mu.Unlock()
		}()

		// Goroutine 3: Average visit duration
		wg.Add(1)
		go func() {
			defer wg.Done()
			rows, err := db.Query(`
			SELECT DATE_TRUNC('day', timestamp) AS date, PERCENTILE_CONT(0.5) WITHIN GROUP (ORDER BY time_spent_on_page) AS median_time_spent
			FROM visits
			WHERE website_domain = $1 AND timestamp BETWEEN $2 AND $3
			GROUP BY date
			ORDER BY date ASC`, domain, start, end)
			if err != nil {
				log.Println("Error getting average visit duration:", err)
				return
			}
			defer rows.Close()

			var dataPoints []map[string]interface{}
			for rows.Next() {
				var date time.Time
				var medianTimeSpent float64
				err = rows.Scan(&date, &medianTimeSpent)
				if err != nil {
					log.Println("Error scanning average visit duration:", err)
					return
				}

				// Convert the time spent on page from milliseconds to seconds
				medianTimeSpent = medianTimeSpent / 1000

				// Convert the median time spent to minutes and seconds
				minutes := int(medianTimeSpent / 60)
				seconds := int(math.Mod(medianTimeSpent, 60))

				dataPoints = append(dataPoints, map[string]interface{}{
					"date":            date.Format("2006-01-02"),
					"medianTimeSpent": fmt.Sprintf("%dm %ds", minutes, seconds),
				})
			}
			mu.Lock()
			averageVisitDuration = dataPoints
			mu.Unlock()
		}()

		// Wait for all goroutines to complete
		wg.Wait()

		// Combine the results into a single JSON response
		jsonStats, err := json.Marshal(map[string]interface{}{
			"totalVisits":          totalVisits,
			"uniqueVisitors":       uniqueVisitors,
			"averageVisitDuration": averageVisitDuration,
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

func GetTopStats2(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Extract the domain from the url
		domain, err := utils.ExtractDomainFromURL(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// Check if the website exists
		var exists bool
		err = db.QueryRow("SELECT EXISTS(SELECT 1 FROM websites WHERE domain = $1)", domain).Scan(&exists)
		if err != nil {
			log.Println("Error checking website existence:", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		if !exists {
			http.Error(w, fmt.Sprintf("Website with domain %s doesn't exist", domain), http.StatusNotFound)
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

		// Query the database for statistics grouped by date
		rows, err := db.Query(`
			SELECT DATE_TRUNC('day', timestamp) AS date, COUNT(*) AS count
			FROM visits
			WHERE website_domain = $1 AND timestamp BETWEEN $2 AND $3
			GROUP BY date
			ORDER BY date ASC`, domain, start, end)
		if err != nil {
			log.Println("Error getting website statistics:", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		// Create a slice to store the data points
		var dataPoints []map[string]interface{}

		// Scan the results into the dataPoints slice
		for rows.Next() {
			var date time.Time
			var count int
			err = rows.Scan(&date, &count)
			if err != nil {
				log.Println("Error scanning statistics:", err)
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			dataPoints = append(dataPoints, map[string]interface{}{
				"date":  date.Format("2006-01-02"),
				"count": count,
			})
		}

		// Convert the dataPoints slice to JSON
		jsonStats, err := json.Marshal(dataPoints)
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
		// Extract the domain from the url
		domain, err := utils.ExtractDomainFromURL(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// Check if the website exists
		var exists bool
		err = db.QueryRow("SELECT EXISTS(SELECT 1 FROM websites WHERE domain = $1)", domain).Scan(&exists)
		if err != nil {
			log.Println("Error checking website existence:", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		if !exists {
			http.Error(w, fmt.Sprintf("Website with domain %s doesn't exist", domain), http.StatusNotFound)
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
		stats, err := db.Query("SELECT pathname, COUNT(*) FROM visits WHERE website_domain = $1 AND timestamp BETWEEN $2 AND $3 GROUP BY pathname ORDER BY COUNT(*) DESC LIMIT 10", domain, start, end)
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

func GetReferrers(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Extract the domain from the url
		domain, err := utils.ExtractDomainFromURL(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// Check if the website exists
		var exists bool
		err = db.QueryRow("SELECT EXISTS(SELECT 1 FROM websites WHERE domain = $1)", domain).Scan(&exists)
		if err != nil {
			log.Println("Error checking website existence:", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		if !exists {
			http.Error(w, fmt.Sprintf("Website with domain %s doesn't exist", domain), http.StatusNotFound)
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
		stats, err := db.Query("SELECT referrer, COUNT(*) FROM visits WHERE website_domain = $1 AND timestamp BETWEEN $2 AND $3 GROUP BY referrer ORDER BY COUNT(*) DESC LIMIT 10", domain, start, end)
		if err != nil {
			log.Println("Error getting website statistics:", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Convert the statistics to JSON
		defer stats.Close() // Close the result set after we're done with it
		var referrer string
		var count int
		var referrers []string
		var counts []int
		for stats.Next() {
			err = stats.Scan(&referrer, &count)
			if err != nil {
				log.Println("Error scanning statistics:", err)
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			referrers = append(referrers, referrer)
			counts = append(counts, count)
		}

		jsonStats, err := json.Marshal(map[string]interface{}{
			"referrers": referrers,
			"counts":    counts,
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

func GetDeviceTypes(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Extract the domain from the url
		domain, err := utils.ExtractDomainFromURL(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// Check if the website exists
		var exists bool
		err = db.QueryRow("SELECT EXISTS(SELECT 1 FROM websites WHERE domain = $1)", domain).Scan(&exists)
		if err != nil {
			log.Println("Error checking website existence:", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		if !exists {
			http.Error(w, fmt.Sprintf("Website with domain %s doesn't exist", domain), http.StatusNotFound)
			return
		}

		// Extract start and end dates from the request query parameters
		startDate := r.URL.Query().Get("startDate")
		endDate := r.URL.Query().Get("endDate")

		// Convert the dates to a format suitable for your database
		start, err := time.Parse("2006-01-02 15:04:05.999", startDate)
		if err != nil {
			log.Println("Error time parsing start date:", err)
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		end, err := time.Parse("2006-01-02 15:04:05.999", endDate)
		if err != nil {
			log.Println("Error time parsing end date:", err)
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// Query the database for statistics
		stats, err := db.Query("SELECT device_type, COUNT(*) FROM visits WHERE website_domain = $1 AND timestamp BETWEEN $2 AND $3 GROUP BY device_type ORDER BY COUNT(*) DESC LIMIT 10", domain, start, end)
		if err != nil {
			log.Println("Error getting website statistics:", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Convert the statistics to JSON
		defer stats.Close() // Close the result set after we're done with it
		var deviceType string
		var count int
		var deviceTypes []string
		var counts []int
		for stats.Next() {
			err = stats.Scan(&deviceType, &count)
			if err != nil {
				log.Println("Error scanning statistics:", err)
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			deviceTypes = append(deviceTypes, deviceType)
			counts = append(counts, count)
		}
		jsonStats, err := json.Marshal(map[string]interface{}{
			"deviceTypes": deviceTypes,
			"counts":      counts,
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

func GetOSes(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Extract the domain from the url
		domain, err := utils.ExtractDomainFromURL(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// Check if the website exists
		var exists bool
		err = db.QueryRow("SELECT EXISTS(SELECT 1 FROM websites WHERE domain = $1)", domain).Scan(&exists)
		if err != nil {
			log.Println("Error checking website existence:", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		if !exists {
			http.Error(w, fmt.Sprintf("Website with domain %s doesn't exist", domain), http.StatusNotFound)
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
		stats, err := db.Query("SELECT os, COUNT(*) FROM visits WHERE website_domain = $1 AND timestamp BETWEEN $2 AND $3 GROUP BY os ORDER BY COUNT(*) DESC LIMIT 10", domain, start, end)
		if err != nil {
			log.Println("Error getting website statistics:", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Convert the statistics to JSON
		defer stats.Close() // Close the result set after we're done with it
		var os string
		var count int
		var oses []string
		var counts []int
		for stats.Next() {
			err = stats.Scan(&os, &count)
			if err != nil {
				log.Println("Error scanning statistics:", err)
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			oses = append(oses, os)
			counts = append(counts, count)
		}
		jsonStats, err := json.Marshal(map[string]interface{}{
			"oses":   oses,
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

func GetBrowsers(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Extract the domain from the url
		domain, err := utils.ExtractDomainFromURL(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// Check if the website exists
		var exists bool
		err = db.QueryRow("SELECT EXISTS(SELECT 1 FROM websites WHERE domain = $1)", domain).Scan(&exists)
		if err != nil {
			log.Println("Error checking website existence:", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		if !exists {
			http.Error(w, fmt.Sprintf("Website with domain %s doesn't exist", domain), http.StatusNotFound)
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
		stats, err := db.Query("SELECT browser, COUNT(*) FROM visits WHERE website_domain = $1 AND timestamp BETWEEN $2 AND $3 GROUP BY browser ORDER BY COUNT(*) DESC LIMIT 10", domain, start, end)
		if err != nil {
			log.Println("Error getting website statistics:", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Convert the statistics to JSON
		defer stats.Close() // Close the result set after we're done with it
		var browser string
		var count int
		var browsers []string
		var counts []int
		for stats.Next() {
			err = stats.Scan(&browser, &count)
			if err != nil {
				log.Println("Error scanning statistics:", err)
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			browsers = append(browsers, browser)
			counts = append(counts, count)
		}
		jsonStats, err := json.Marshal(map[string]interface{}{
			"browsers": browsers,
			"counts":   counts,
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

func GetLanguages(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Extract the domain from the url
		domain, err := utils.ExtractDomainFromURL(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// Check if the website exists
		var exists bool
		err = db.QueryRow("SELECT EXISTS(SELECT 1 FROM websites WHERE domain = $1)", domain).Scan(&exists)
		if err != nil {
			log.Println("Error checking website existence:", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		if !exists {
			http.Error(w, fmt.Sprintf("Website with domain %s doesn't exist", domain), http.StatusNotFound)
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
		stats, err := db.Query("SELECT language, COUNT(*) FROM visits WHERE website_domain = $1 AND timestamp BETWEEN $2 AND $3 GROUP BY language ORDER BY COUNT(*) DESC LIMIT 10", domain, start, end)
		if err != nil {
			log.Println("Error getting website statistics:", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Convert the statistics to JSON
		defer stats.Close() // Close the result set after we're done with it
		var language string
		var count int
		var languages []string
		var counts []int
		for stats.Next() {
			err = stats.Scan(&language, &count)
			if err != nil {
				log.Println("Error scanning statistics:", err)
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			languages = append(languages, language)
			counts = append(counts, count)
		}
		jsonStats, err := json.Marshal(map[string]interface{}{
			"languages": languages,
			"counts":    counts,
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

func GetCountries(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Extract the domain from the url
		domain, err := utils.ExtractDomainFromURL(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// Check if the website exists
		var exists bool
		err = db.QueryRow("SELECT EXISTS(SELECT 1 FROM websites WHERE domain = $1)", domain).Scan(&exists)
		if err != nil {
			log.Println("Error checking website existence:", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		if !exists {
			http.Error(w, fmt.Sprintf("Website with domain %s doesn't exist", domain), http.StatusNotFound)
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
		stats, err := db.Query("SELECT country, COUNT(*) FROM visits WHERE website_domain = $1 AND timestamp BETWEEN $2 AND $3 GROUP BY country ORDER BY COUNT(*) DESC LIMIT 10", domain, start, end)
		if err != nil {
			log.Println("Error getting website statistics:", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Convert the statistics to JSON
		defer stats.Close() // Close the result set after we're done with it
		var country string
		var count int
		var countries []string
		var counts []int
		for stats.Next() {
			err = stats.Scan(&country, &count)
			if err != nil {
				log.Println("Error scanning statistics:", err)
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			countries = append(countries, country)
			counts = append(counts, count)
		}
		jsonStats, err := json.Marshal(map[string]interface{}{
			"countries": countries,
			"counts":    counts,
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

func GetStates(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Extract the domain from the url
		domain, err := utils.ExtractDomainFromURL(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// Check if the website exists
		var exists bool
		err = db.QueryRow("SELECT EXISTS(SELECT 1 FROM websites WHERE domain = $1)", domain).Scan(&exists)
		if err != nil {
			log.Println("Error checking website existence:", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		if !exists {
			http.Error(w, fmt.Sprintf("Website with domain %s doesn't exist", domain), http.StatusNotFound)
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
		stats, err := db.Query("SELECT state, COUNT(*) FROM visits WHERE website_domain = $1 AND timestamp BETWEEN $2 AND $3 GROUP BY state ORDER BY COUNT(*) DESC LIMIT 10", domain, start, end)
		if err != nil {
			log.Println("Error getting website statistics:", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Convert the statistics to JSON
		defer stats.Close() // Close the result set after we're done with it
		var state string
		var count int
		var states []string
		var counts []int
		for stats.Next() {
			err = stats.Scan(&state, &count)
			if err != nil {
				log.Println("Error scanning statistics:", err)
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			states = append(states, state)
			counts = append(counts, count)
		}
		jsonStats, err := json.Marshal(map[string]interface{}{
			"states": states,
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
