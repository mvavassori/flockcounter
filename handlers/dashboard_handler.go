package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"sort"

	// "math"
	"net/http"
	"net/url"
	"strconv"
	"sync"
	"time"

	"github.com/mvavassori/bare-analytics/utils"
)

// todo fix hour interval format
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

		// Convert the dates to a format suitable for my database
		start, err := time.Parse("2006-01-02T15:04:05.999Z07:00", startDate)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		end, err := time.Parse("2006-01-02T15:04:05.999Z07:00", endDate)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// Extract the interval from the request query parameters
		interval := r.URL.Query().Get("interval")
		if interval != "hour" && interval != "day" && interval != "month" {
			http.Error(w, "Invalid interval", http.StatusBadRequest)
			return
		}

		// Determine the time layout based on the interval
		var layout string
		switch interval {
		case "hour":
			layout = "15"
		case "month":
			layout = "2006-01"
		case "day":
			fallthrough
		default:
			layout = "2006-01-02"
		}

		var wg sync.WaitGroup
		var mu sync.Mutex

		var totalVisits []map[string]interface{}
		var uniqueVisitors []map[string]interface{}
		var medianVisitDuration []map[string]interface{}

		var totalVisitsAggregate int
		var uniqueVisitorsAggregate int
		var medianVisitDurationAggregate float64
		var visitPeriodsCount int

		// Generate a list of all periods in the range
		periods := make([]time.Time, 0)
		for d := start; !d.After(end); {
			periods = append(periods, d)
			switch interval {
			case "hour":
				d = d.Add(time.Hour)
			case "month":
				d = d.AddDate(0, 1, 0)
			case "day":
				fallthrough
			default:
				d = d.AddDate(0, 0, 1)
			}
		}

		// Goroutine 1: Total visits
		wg.Add(1)
		go func() {
			defer wg.Done()
			rows, err := db.Query(fmt.Sprintf(`
				SELECT DATE_TRUNC('%s', timestamp) AS period, COUNT(*) AS count
				FROM visits
				WHERE website_domain = $1 AND timestamp BETWEEN $2 AND $3
				GROUP BY period
				ORDER BY period ASC`, interval), domain, start, end)
			if err != nil {
				log.Println("Error getting total visits:", err)
				return
			}
			defer rows.Close()

			var dataPoints []map[string]interface{}
			for rows.Next() {
				var period time.Time
				var count int
				err = rows.Scan(&period, &count)
				if err != nil {
					log.Println("Error scanning total visits:", err)
					return
				}
				dataPoints = append(dataPoints, map[string]interface{}{
					"period": period.Format(layout),
					"count":  count,
				})
				totalVisitsAggregate += count
			}

			// Fill in missing periods with zero values
			for _, p := range periods {
				found := false
				for _, dp := range dataPoints {
					if dp["period"] == p.Format(layout) {
						found = true
						break
					}
				}
				if !found {
					dataPoints = append(dataPoints, map[string]interface{}{
						"period": p.Format(layout),
						"count":  0,
					})
				}
			}

			mu.Lock()
			totalVisits = dataPoints
			mu.Unlock()
		}()

		// Goroutine 2: Unique visitors
		wg.Add(1)
		go func() {
			defer wg.Done()
			rows, err := db.Query(fmt.Sprintf(`
				SELECT DATE_TRUNC('%s', timestamp) AS period, COUNT(*) AS count
				FROM visits
				WHERE website_domain = $1 AND timestamp BETWEEN $2 AND $3 AND is_unique = true
				GROUP BY period
				ORDER BY period ASC`, interval), domain, start, end)
			if err != nil {
				log.Println("Error getting unique visitors:", err)
				return
			}
			defer rows.Close()

			var dataPoints []map[string]interface{}
			for rows.Next() {
				var period time.Time
				var count int
				err = rows.Scan(&period, &count)
				if err != nil {
					log.Println("Error scanning unique visitors:", err)
					return
				}
				dataPoints = append(dataPoints, map[string]interface{}{
					"period": period.Format(layout),
					"count":  count,
				})
				uniqueVisitorsAggregate += count
			}

			// Fill in missing periods with zero values
			for _, p := range periods {
				found := false
				for _, dp := range dataPoints {
					if dp["period"] == p.Format(layout) {
						found = true
						break
					}
				}
				if !found {
					dataPoints = append(dataPoints, map[string]interface{}{
						"period": p.Format(layout),
						"count":  0,
					})
				}
			}

			mu.Lock()
			uniqueVisitors = dataPoints
			mu.Unlock()
		}()

		// Goroutine 3: Median visit duration
		wg.Add(1)
		go func() {
			defer wg.Done()
			rows, err := db.Query(fmt.Sprintf(`
				SELECT DATE_TRUNC('%s', timestamp) AS period, time_spent_on_page
				FROM visits
				WHERE website_domain = $1 AND timestamp BETWEEN $2 AND $3
				ORDER BY period ASC`, interval), domain, start, end)
			if err != nil {
				log.Println("Error getting median visit duration:", err)
				return
			}
			defer rows.Close()

			periodDurations := make(map[time.Time][]float64)
			for rows.Next() {
				var period time.Time
				var timeSpent float64
				err = rows.Scan(&period, &timeSpent)
				if err != nil {
					log.Println("Error scanning median visit duration:", err)
					return
				}
				periodDurations[period] = append(periodDurations[period], timeSpent)
			}

			var dataPoints []map[string]interface{}
			for period, durations := range periodDurations {
				sort.Float64s(durations)
				medianIndex := len(durations) / 2
				var median float64
				if len(durations)%2 == 0 {
					median = (durations[medianIndex-1] + durations[medianIndex]) / 2
				} else {
					median = durations[medianIndex]
				}

				// Convert the median time spent to seconds
				median /= 1000

				// Convert the median time spent to minutes and seconds
				minutes := int(median / 60)
				seconds := int(median) % 60

				dataPoints = append(dataPoints, map[string]interface{}{
					"period":          period.Format(layout),
					"medianTimeSpent": fmt.Sprintf("%dm %ds", minutes, seconds),
				})
				medianVisitDurationAggregate += median
				visitPeriodsCount++
			}

			// Fill in missing periods with zero seconds values
			for _, p := range periods {
				found := false
				for _, dp := range dataPoints {
					if dp["period"] == p.Format(layout) {
						found = true
						break
					}
				}
				if !found {
					dataPoints = append(dataPoints, map[string]interface{}{
						"period":          p.Format(layout),
						"medianTimeSpent": "0m 0s",
					})
				}
			}

			mu.Lock()
			medianVisitDuration = dataPoints
			mu.Unlock()
		}()

		// Wait for all goroutines to complete
		wg.Wait()

		// Calculate the median visit duration aggregate
		if visitPeriodsCount > 0 {
			medianVisitDurationAggregate /= float64(visitPeriodsCount)
		}

		// Format the median visit duration aggregate in a readable format
		hours := int(medianVisitDurationAggregate / 3600)
		minutes := int(math.Mod(medianVisitDurationAggregate, 3600) / 60)
		seconds := int(math.Mod(medianVisitDurationAggregate, 60))

		var medianVisitDurationAggregateFormatted string
		if hours > 0 {
			medianVisitDurationAggregateFormatted = fmt.Sprintf("%dh %dm %ds", hours, minutes, seconds)
		} else if minutes > 0 {
			medianVisitDurationAggregateFormatted = fmt.Sprintf("%dm %ds", minutes, seconds)
		} else {
			medianVisitDurationAggregateFormatted = fmt.Sprintf("%ds", seconds)
		}

		// Sort the results by period
		utils.SortByPeriod(totalVisits, interval)
		utils.SortByPeriod(uniqueVisitors, interval)
		utils.SortByPeriod(medianVisitDuration, interval)

		// Combine the results into a single JSON response
		jsonStats, err := json.Marshal(map[string]interface{}{
			"perIntervalStats": map[string]interface{}{
				"totalVisits":         totalVisits,
				"uniqueVisitors":      uniqueVisitors,
				"medianVisitDuration": medianVisitDuration,
			},
			"aggregates": map[string]interface{}{
				"totalVisits":         totalVisitsAggregate,
				"uniqueVisitors":      uniqueVisitorsAggregate,
				"medianVisitDuration": medianVisitDurationAggregateFormatted,
			},
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

		// Convert the dates to a format suitable for my database
		start, err := time.Parse("2006-01-02T15:04:05.999Z07:00", startDate)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		end, err := time.Parse("2006-01-02T15:04:05.999Z07:00", endDate)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// Extract limit and offset from query string
		limit, err := strconv.Atoi(r.URL.Query().Get("limit"))
		if err != nil || limit <= 0 {
			limit = 10 // default limit
		}
		offset, err := strconv.Atoi(r.URL.Query().Get("offset"))
		if err != nil || offset < 0 {
			offset = 0 // default offset
		}

		// Query the database for statistics
		stats, err := db.Query("SELECT pathname, COUNT(*) FROM visits WHERE website_domain = $1 AND timestamp BETWEEN $2 AND $3 GROUP BY pathname ORDER BY COUNT(*) DESC LIMIT $4 OFFSET $5", domain, start, end, limit, offset)
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

		// Convert the dates to a format suitable for my database
		start, err := time.Parse("2006-01-02T15:04:05.999Z07:00", startDate)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		end, err := time.Parse("2006-01-02T15:04:05.999Z07:00", endDate)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// Extract limit and offset from query string
		limit, err := strconv.Atoi(r.URL.Query().Get("limit"))
		if err != nil || limit <= 0 {
			limit = 10 // default limit
		}
		offset, err := strconv.Atoi(r.URL.Query().Get("offset"))
		if err != nil || offset < 0 {
			offset = 0 // default offset
		}

		// Query the database for statistics
		stats, err := db.Query("SELECT referrer, COUNT(*) FROM visits WHERE website_domain = $1 AND timestamp BETWEEN $2 AND $3 GROUP BY referrer ORDER BY COUNT(*) DESC LIMIT $4 OFFSET $5", domain, start, end, limit, offset)
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
			// Leave out the http:// or https:// part from the referrer url
			u, err := url.Parse(referrer)
			if err != nil {
				log.Println("Error parsing referrer URL:", err)
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			referrerWithoutProtocol := u.Host + u.Path

			referrers = append(referrers, referrerWithoutProtocol)
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

		// Convert the dates to a format suitable for my database
		start, err := time.Parse("2006-01-02T15:04:05.999Z07:00", startDate)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		end, err := time.Parse("2006-01-02T15:04:05.999Z07:00", endDate)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// Query the database for statistics
		stats, err := db.Query("SELECT device_type, COUNT(*) FROM visits WHERE website_domain = $1 AND timestamp BETWEEN $2 AND $3 GROUP BY device_type ORDER BY COUNT(*) DESC", domain, start, end)
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

		// Convert the dates to a format suitable for my database
		start, err := time.Parse("2006-01-02T15:04:05.999Z07:00", startDate)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		end, err := time.Parse("2006-01-02T15:04:05.999Z07:00", endDate)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// Query the database for statistics
		stats, err := db.Query("SELECT os, COUNT(*) FROM visits WHERE website_domain = $1 AND timestamp BETWEEN $2 AND $3 GROUP BY os ORDER BY COUNT(*) DESC", domain, start, end)
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

		// Convert the dates to a format suitable for my database
		start, err := time.Parse("2006-01-02T15:04:05.999Z07:00", startDate)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		end, err := time.Parse("2006-01-02T15:04:05.999Z07:00", endDate)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// Query the database for statistics
		stats, err := db.Query("SELECT browser, COUNT(*) FROM visits WHERE website_domain = $1 AND timestamp BETWEEN $2 AND $3 GROUP BY browser ORDER BY COUNT(*) DESC", domain, start, end)
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

		// Convert the dates to a format suitable for my database
		start, err := time.Parse("2006-01-02T15:04:05.999Z07:00", startDate)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		end, err := time.Parse("2006-01-02T15:04:05.999Z07:00", endDate)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// Extract limit and offset from query string
		limit, err := strconv.Atoi(r.URL.Query().Get("limit"))
		if err != nil || limit <= 0 {
			limit = 10 // default limit
		}
		offset, err := strconv.Atoi(r.URL.Query().Get("offset"))
		if err != nil || offset < 0 {
			offset = 0 // default offset
		}

		stats, err := db.Query("SELECT language, COUNT(*) FROM visits WHERE website_domain = $1 AND timestamp BETWEEN $2 AND $3 GROUP BY language ORDER BY COUNT(*) DESC LIMIT $4 OFFSET $5", domain, start, end, limit, offset)
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

		// Convert the dates to a format suitable for my database
		start, err := time.Parse("2006-01-02T15:04:05.999Z07:00", startDate)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		end, err := time.Parse("2006-01-02T15:04:05.999Z07:00", endDate)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// Extract limit and offset from query string
		limit, err := strconv.Atoi(r.URL.Query().Get("limit"))
		if err != nil || limit <= 0 {
			limit = 10 // default limit
		}
		offset, err := strconv.Atoi(r.URL.Query().Get("offset"))
		if err != nil || offset < 0 {
			offset = 0 // default offset
		}

		// Query the database for statistics
		stats, err := db.Query("SELECT country, COUNT(*) FROM visits WHERE website_domain = $1 AND timestamp BETWEEN $2 AND $3 GROUP BY country ORDER BY COUNT(*) DESC LIMIT $4 OFFSET $5", domain, start, end, limit, offset)
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

func GetRegions(db *sql.DB) http.HandlerFunc {
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

		// Convert the dates to a format suitable for my database
		start, err := time.Parse("2006-01-02T15:04:05.999Z07:00", startDate)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		end, err := time.Parse("2006-01-02T15:04:05.999Z07:00", endDate)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// Extract limit and offset from query string
		limit, err := strconv.Atoi(r.URL.Query().Get("limit"))
		if err != nil || limit <= 0 {
			limit = 10 // default limit
		}
		offset, err := strconv.Atoi(r.URL.Query().Get("offset"))
		if err != nil || offset < 0 {
			offset = 0 // default offset
		}

		// Query the database for statistics
		stats, err := db.Query("SELECT region, COUNT(*) FROM visits WHERE website_domain = $1 AND timestamp BETWEEN $2 AND $3 GROUP BY region ORDER BY COUNT(*) DESC LIMIT $4 OFFSET $5", domain, start, end, limit, offset)
		if err != nil {
			log.Println("Error getting website statistics:", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Convert the statistics to JSON
		defer stats.Close() // Close the result set after we're done with it
		var region string
		var count int
		var regions []string
		var counts []int
		for stats.Next() {
			err = stats.Scan(&region, &count)
			if err != nil {
				log.Println("Error scanning statistics:", err)
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			regions = append(regions, region)
			counts = append(counts, count)
		}
		jsonStats, err := json.Marshal(map[string]interface{}{
			"regions": regions,
			"counts":  counts,
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

func GetCities(db *sql.DB) http.HandlerFunc {
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

		// Convert the dates to a format suitable for my database
		start, err := time.Parse("2006-01-02T15:04:05.999Z07:00", startDate)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		end, err := time.Parse("2006-01-02T15:04:05.999Z07:00", endDate)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// Extract limit and offset from query string
		limit, err := strconv.Atoi(r.URL.Query().Get("limit"))
		if err != nil || limit <= 0 {
			limit = 10 // default limit
		}
		offset, err := strconv.Atoi(r.URL.Query().Get("offset"))
		if err != nil || offset < 0 {
			offset = 0 // default offset
		}

		// Query the database for statistics
		stats, err := db.Query("SELECT city, COUNT(*) FROM visits WHERE website_domain = $1 AND timestamp BETWEEN $2 AND $3 GROUP BY city ORDER BY COUNT(*) DESC LIMIT $4 OFFSET $5", domain, start, end, limit, offset)
		if err != nil {
			log.Println("Error getting website statistics:", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Convert the statistics to JSON
		defer stats.Close() // Close the result set after we're done with it
		var city string
		var count int
		var cities []string
		var counts []int
		for stats.Next() {
			err = stats.Scan(&city, &count)
			if err != nil {
				log.Println("Error scanning statistics:", err)
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			cities = append(cities, city)
			counts = append(counts, count)
		}
		jsonStats, err := json.Marshal(map[string]interface{}{
			"cities": cities,
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
