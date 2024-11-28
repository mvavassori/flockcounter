package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"sort"

	// "strings"
	// "math"
	"net/http"
	// "net/url"
	"strconv"
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
			switch interval {
			case "hour":
				// Include the current hour, counting from 00:00 to the current hour
				d = d.Truncate(time.Hour)
				periods = append(periods, d)
				d = d.Add(time.Hour)
			case "month":
				// Include the first day of each month up to the current month
				d = time.Date(d.Year(), d.Month(), 1, 0, 0, 0, 0, time.UTC)
				periods = append(periods, d)
				d = d.AddDate(0, 1, 0)
			case "day":
				// Include each day, capturing 00:00 to 23:59 for each day
				d = d.Truncate(24 * time.Hour)
				periods = append(periods, d)
				d = d.AddDate(0, 0, 1)
			}
		}

		// Initialize base query and parameters for filtering
		baseQuery := fmt.Sprintf(`
			SELECT DATE_TRUNC('%s', timestamp) AS period, COUNT(*) AS count
			FROM visits
			WHERE website_domain = $1 AND timestamp BETWEEN $2 AND $3`, interval)
		params := []interface{}{domain, start, end}
		paramIndex := 4

		// Map query parameter names to column names
		filters := map[string]string{
			"referrer":    "referrer",
			"pathname":    "pathname",
			"device_type": "device_type",
			"os":          "os",
			"browser":     "browser",
			"language":    "language",
			"country":     "country",
			"city":        "city",
			"region":      "region",
		}

		// Add filters to the query
		for param, column := range filters {
			value := r.URL.Query().Get(param)
			if value != "" {
				log.Printf(" - %s: %s", param, value)
				baseQuery += fmt.Sprintf(" AND %s = $%d", column, paramIndex)
				params = append(params, value)
				paramIndex++
			}
		}

		// Complete the query with grouping and ordering
		baseQuery += " GROUP BY period ORDER BY period ASC"

		// Goroutine 1: Total visits
		wg.Add(1)
		go func() {
			defer wg.Done()
			rows, err := db.Query(baseQuery, params...)
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
					"period": period.Format(time.RFC3339),
					"count":  count,
				})
				totalVisitsAggregate += count
			}

			// Fill in missing periods with zero values
			for _, p := range periods {
				found := false
				for _, dp := range dataPoints {
					if dp["period"] == p.Format(time.RFC3339) {
						found = true
						break
					}
				}
				if !found {
					dataPoints = append(dataPoints, map[string]interface{}{
						"period": p.Format(time.RFC3339),
						"count":  0,
					})
				}
			}

			// Sort data points by period
			sort.Slice(dataPoints, func(i, j int) bool {
				return dataPoints[i]["period"].(string) < dataPoints[j]["period"].(string)
			})

			mu.Lock()
			totalVisits = dataPoints
			mu.Unlock()
		}()

		// Goroutine 2: Unique visitors
		wg.Add(1)
		go func() {
			defer wg.Done()

			// Initialize base query and parameters for filtering
			baseQuery := fmt.Sprintf(`
		SELECT DATE_TRUNC('%s', timestamp) AS period, COUNT(*) AS count
		FROM visits
		WHERE website_domain = $1 AND timestamp BETWEEN $2 AND $3 AND is_unique = true`, interval)
			params := []interface{}{domain, start, end}
			paramIndex := 4

			// Map query parameter names to column names
			filters := map[string]string{
				"referrer":    "referrer",
				"pathname":    "pathname",
				"device_type": "device_type",
				"os":          "os",
				"browser":     "browser",
				"language":    "language",
				"country":     "country",
				"city":        "city",
				"region":      "region",
			}

			// Add filters to the query
			for param, column := range filters {
				value := r.URL.Query().Get(param)
				if value != "" {
					log.Printf(" - %s: %s", param, value)
					baseQuery += fmt.Sprintf(" AND %s = $%d", column, paramIndex)
					params = append(params, value)
					paramIndex++
				}
			}

			// Complete the query with grouping and ordering
			baseQuery += " GROUP BY period ORDER BY period ASC"

			rows, err := db.Query(baseQuery, params...)
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
					"period": period.Format(time.RFC3339),
					"count":  count,
				})
				uniqueVisitorsAggregate += count
			}

			// Fill in missing periods with zero values
			for _, p := range periods {
				found := false
				for _, dp := range dataPoints {
					if dp["period"] == p.Format(time.RFC3339) {
						found = true
						break
					}
				}
				if !found {
					dataPoints = append(dataPoints, map[string]interface{}{
						"period": p.Format(time.RFC3339),
						"count":  0,
					})
				}
			}

			// Sort data points by period
			sort.Slice(dataPoints, func(i, j int) bool {
				return dataPoints[i]["period"].(string) < dataPoints[j]["period"].(string)
			})

			mu.Lock()
			uniqueVisitors = dataPoints
			mu.Unlock()
		}()

		// Goroutine 3: Median visit duration
		wg.Add(1)
		go func() {
			defer wg.Done()

			// Initialize base query and parameters for filtering
			baseQuery := fmt.Sprintf(`
		SELECT DATE_TRUNC('%s', timestamp) AS period, time_spent_on_page
		FROM visits
		WHERE website_domain = $1 AND timestamp BETWEEN $2 AND $3`, interval)
			params := []interface{}{domain, start, end}
			paramIndex := 4

			// Map query parameter names to column names
			filters := map[string]string{
				"referrer":    "referrer",
				"pathname":    "pathname",
				"device_type": "device_type",
				"os":          "os",
				"browser":     "browser",
				"language":    "language",
				"country":     "country",
				"city":        "city",
				"region":      "region",
			}

			// Add filters to the query
			for param, column := range filters {
				value := r.URL.Query().Get(param)
				if value != "" {
					log.Printf(" - %s: %s", param, value)
					baseQuery += fmt.Sprintf(" AND %s = $%d", column, paramIndex)
					params = append(params, value)
					paramIndex++
				}
			}

			// Complete the query with ordering
			baseQuery += " ORDER BY period ASC"

			rows, err := db.Query(baseQuery, params...)
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

				// Convert the median time spent to the correct time format
				var timeFormat string
				if median < 60 {
					timeFormat = fmt.Sprintf("%ds", int(median))
				} else if median < 3600 {
					minutes := int(median / 60)
					seconds := int(median) % 60
					timeFormat = fmt.Sprintf("%dm %ds", minutes, seconds)
				} else {
					hours := int(median / 3600)
					minutes := (int(median) % 3600) / 60
					seconds := int(median) % 60
					timeFormat = fmt.Sprintf("%dh %dm %ds", hours, minutes, seconds)
				}

				dataPoints = append(dataPoints, map[string]interface{}{
					"period":          period.Format(time.RFC3339),
					"medianTimeSpent": timeFormat,
				})
				medianVisitDurationAggregate += median
				visitPeriodsCount++
			}

			// Fill in missing periods with zero seconds values
			for _, p := range periods {
				found := false
				for _, dp := range dataPoints {
					if dp["period"] == p.Format(time.RFC3339) {
						found = true
						break
					}
				}
				if !found {
					dataPoints = append(dataPoints, map[string]interface{}{
						"period":          p.Format(time.RFC3339),
						"medianTimeSpent": "0s",
					})
				}
			}

			// Sort data points by period
			sort.Slice(dataPoints, func(i, j int) bool {
				return dataPoints[i]["period"].(string) < dataPoints[j]["period"].(string)
			})

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
		// utils.SortByPeriod(totalVisits, interval)
		// utils.SortByPeriod(uniqueVisitors, interval)
		// utils.SortByPeriod(medianVisitDuration, interval)

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
		// Extract the domain from the URL
		domain, err := utils.ExtractDomainFromURL(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// Extract start and end dates from the request query parameters
		startDate := r.URL.Query().Get("startDate")
		endDate := r.URL.Query().Get("endDate")

		// Convert the dates to a format suitable for the database
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

		// Initialize queries and parameters
		baseQuery := "SELECT pathname, COUNT(*) FROM visits WHERE website_domain = $1 AND timestamp BETWEEN $2 AND $3"
		countQuery := "SELECT COUNT(DISTINCT pathname) FROM visits WHERE website_domain = $1 AND timestamp BETWEEN $2 AND $3"
		params := []interface{}{domain, start, end}
		paramIndex := 4

		// Map query parameter names to column names
		filters := map[string]string{
			"referrer":    "referrer",
			"pathname":    "pathname",
			"device_type": "device_type",
			"os":          "os",
			"browser":     "browser",
			"language":    "language",
			"country":     "country",
			"city":        "city",
			"region":      "region",
		}

		// Add filters to the query
		for param, column := range filters {
			value := r.URL.Query().Get(param)
			if value != "" {
				baseQuery += fmt.Sprintf(" AND %s = $%d", column, paramIndex)
				countQuery += fmt.Sprintf(" AND %s = $%d", column, paramIndex)
				params = append(params, value)
				paramIndex++
			}
		}

		// Complete the queries
		dataQuery := baseQuery + fmt.Sprintf(" GROUP BY pathname ORDER BY COUNT(*) DESC LIMIT $%d OFFSET $%d", paramIndex, paramIndex+1)
		dataParams := append(params, limit, offset)

		var wg sync.WaitGroup
		var totalCount int
		var paths []string
		var counts []int
		var countErr, dataErr error

		// Goroutine for count query
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := db.QueryRow(countQuery, params...).Scan(&totalCount)
			if err != nil {
				countErr = err
			}
		}()

		// Goroutine for data query
		wg.Add(1)
		go func() {
			defer wg.Done()
			rows, err := db.Query(dataQuery, dataParams...)
			if err != nil {
				dataErr = err
				return
			}
			defer rows.Close()

			for rows.Next() {
				var path string
				var count int
				if err := rows.Scan(&path, &count); err != nil {
					dataErr = err
					return
				}
				paths = append(paths, path)
				counts = append(counts, count)
			}

			if err := rows.Err(); err != nil {
				dataErr = err
			}
		}()

		// Wait for both goroutines to finish
		wg.Wait()

		// Check for errors
		if countErr != nil {
			log.Println("Error getting total count:", countErr)
			http.Error(w, countErr.Error(), http.StatusInternalServerError)
			return
		}
		if dataErr != nil {
			log.Println("Error getting page data:", dataErr)
			http.Error(w, dataErr.Error(), http.StatusInternalServerError)
			return
		}

		// Prepare and send the JSON response
		jsonStats, err := json.Marshal(map[string]interface{}{
			"paths":      paths,
			"counts":     counts,
			"totalCount": totalCount,
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

		// Initialize query and parameters
		baseQuery := "SELECT referrer, COUNT(*) FROM visits WHERE website_domain = $1 AND timestamp BETWEEN $2 AND $3"
		countQuery := "SELECT COUNT(DISTINCT referrer) FROM visits WHERE website_domain = $1 AND timestamp BETWEEN $2 AND $3"
		params := []interface{}{domain, start, end}
		paramIndex := 4 // Start the parameter index at 4 because $1, $2, and $3 are already used

		// Map query parameter names to column names
		filters := map[string]string{
			"referrer":    "referrer",
			"pathname":    "pathname",
			"device_type": "device_type",
			"os":          "os",
			"browser":     "browser",
			"language":    "language",
			"country":     "country",
			"city":        "city",
			"region":      "region",
		}

		// Add filters to the query
		for param, column := range filters {
			value := r.URL.Query().Get(param)
			if value != "" {
				baseQuery += fmt.Sprintf(" AND %s = $%d", column, paramIndex) // Add the filter to the query with the current parameter index
				countQuery += fmt.Sprintf(" AND %s = $%d", column, paramIndex)
				params = append(params, value) // Add the filter value to the parameters list
				paramIndex++                   // Increment the parameter index for the next filter
			}
		}

		// Complete the queryies
		dataQuery := baseQuery + fmt.Sprintf(" GROUP BY referrer ORDER BY COUNT(*) DESC LIMIT $%d OFFSET $%d", paramIndex, paramIndex+1)
		dataParams := append(params, limit, offset)

		var wg sync.WaitGroup
		var totalCount int
		var referrers []string
		var counts []int
		var countErr, dataErr error

		// Goroutine for count query
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := db.QueryRow(countQuery, params...).Scan(&totalCount)
			if err != nil {
				countErr = err
			}
		}()

		// Goroutine for data query
		wg.Add(1)
		go func() {
			defer wg.Done()
			rows, err := db.Query(dataQuery, dataParams...)
			if err != nil {
				dataErr = err
				return
			}
			defer rows.Close()

			for rows.Next() {
				var referrer string
				var count int
				if err := rows.Scan(&referrer, &count); err != nil {
					dataErr = err
					return
				}
				referrers = append(referrers, referrer)
				counts = append(counts, count)
			}

			if err := rows.Err(); err != nil {
				dataErr = err
			}
		}()

		// Wait for both goroutines to finish
		wg.Wait()

		// Check for errors
		if countErr != nil {
			log.Println("Error getting total count:", countErr)
			http.Error(w, countErr.Error(), http.StatusInternalServerError)
			return
		}
		if dataErr != nil {
			log.Println("Error getting referrer data:", dataErr)
			http.Error(w, dataErr.Error(), http.StatusInternalServerError)
			return
		}

		// Prepare and send the JSON response
		jsonStats, err := json.Marshal(map[string]interface{}{
			"referrers":  referrers,
			"counts":     counts,
			"totalCount": totalCount,
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

		// Initialize query and parameters
		baseQuery := "SELECT device_type, COUNT(*) FROM visits WHERE website_domain = $1 AND timestamp BETWEEN $2 AND $3"
		params := []interface{}{domain, start, end}
		paramIndex := 4 // Start the parameter index at 4 because $1, $2, and $3 are already used

		// Map query parameter names to column names
		filters := map[string]string{
			"referrer":    "referrer",
			"pathname":    "pathname",
			"device_type": "device_type",
			"os":          "os",
			"browser":     "browser",
			"language":    "language",
			"country":     "country",
			"city":        "city",
			"region":      "region",
		}

		// Add filters to the query
		for param, column := range filters {
			value := r.URL.Query().Get(param)
			if value != "" {
				log.Printf(" - %s: %s", param, value)                         // Print the filter being added
				baseQuery += fmt.Sprintf(" AND %s = $%d", column, paramIndex) // Add the filter to the query with the current parameter index
				params = append(params, value)                                // Add the filter value to the parameters list
				paramIndex++                                                  // Increment the parameter index for the next filter
			}
		}

		// Complete the query
		baseQuery += " GROUP BY device_type ORDER BY COUNT(*) DESC"

		// Query the database for statistics
		stats, err := db.Query(baseQuery, params...)
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

		// Initialize query and parameters
		baseQuery := "SELECT os, COUNT(*) FROM visits WHERE website_domain = $1 AND timestamp BETWEEN $2 AND $3"
		params := []interface{}{domain, start, end}
		paramIndex := 4 // Start the parameter index at 4 because $1, $2, and $3 are already used

		// Map query parameter names to column names
		filters := map[string]string{
			"referrer":    "referrer",
			"pathname":    "pathname",
			"device_type": "device_type",
			"os":          "os",
			"browser":     "browser",
			"language":    "language",
			"country":     "country",
			"city":        "city",
			"region":      "region",
		}

		// Add filters to the query
		for param, column := range filters {
			value := r.URL.Query().Get(param)
			if value != "" {
				log.Printf(" - %s: %s", param, value)                         // Print the filter being added
				baseQuery += fmt.Sprintf(" AND %s = $%d", column, paramIndex) // Add the filter to the query with the current parameter index
				params = append(params, value)                                // Add the filter value to the parameters list
				paramIndex++                                                  // Increment the parameter index for the next filter
			}
		}

		// Complete the query
		baseQuery += " GROUP BY os ORDER BY COUNT(*) DESC"

		// Query the database for statistics
		stats, err := db.Query(baseQuery, params...)
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

		// Initialize query and parameters
		baseQuery := "SELECT browser, COUNT(*) FROM visits WHERE website_domain = $1 AND timestamp BETWEEN $2 AND $3"
		params := []interface{}{domain, start, end}
		paramIndex := 4 // Start the parameter index at 4 because $1, $2, and $3 are already used

		// Map query parameter names to column names
		filters := map[string]string{
			"referrer":    "referrer",
			"pathname":    "pathname",
			"device_type": "device_type",
			"os":          "os",
			"browser":     "browser",
			"language":    "language",
			"country":     "country",
			"city":        "city",
			"region":      "region",
		}

		// Add filters to the query
		for param, column := range filters {
			value := r.URL.Query().Get(param)
			if value != "" {
				log.Printf(" - %s: %s", param, value)                         // Print the filter being added
				baseQuery += fmt.Sprintf(" AND %s = $%d", column, paramIndex) // Add the filter to the query with the current parameter index
				params = append(params, value)                                // Add the filter value to the parameters list
				paramIndex++                                                  // Increment the parameter index for the next filter
			}
		}

		// Complete the query
		baseQuery += " GROUP BY browser ORDER BY COUNT(*) DESC"

		// Query the database for statistics
		stats, err := db.Query(baseQuery, params...)
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

		// Initialize query and parameters
		baseQuery := "SELECT language, COUNT(*) FROM visits WHERE website_domain = $1 AND timestamp BETWEEN $2 AND $3"
		countQuery := "SELECT COUNT(DISTINCT language) FROM visits WHERE website_domain = $1 AND timestamp BETWEEN $2 AND $3"
		params := []interface{}{domain, start, end}
		paramIndex := 4 // Start the parameter index at 4 because $1, $2, and $3 are already used

		// Map query parameter names to column names
		filters := map[string]string{
			"referrer":    "referrer",
			"pathname":    "pathname",
			"device_type": "device_type",
			"os":          "os",
			"browser":     "browser",
			"language":    "language",
			"country":     "country",
			"city":        "city",
			"region":      "region",
		}

		// Add filters to the query
		for param, column := range filters {
			value := r.URL.Query().Get(param)
			if value != "" {
				baseQuery += fmt.Sprintf(" AND %s = $%d", column, paramIndex) // Add the filter to the query with the current parameter index
				countQuery += fmt.Sprintf(" AND %s = $%d", column, paramIndex)
				params = append(params, value) // Add the filter value to the parameters list
				paramIndex++                   // Increment the parameter index for the next filter
			}
		}

		// Complete the query
		dataQuery := baseQuery + fmt.Sprintf(" GROUP BY language ORDER BY COUNT(*) DESC LIMIT $%d OFFSET $%d", paramIndex, paramIndex+1)
		dataParams := append(params, limit, offset)

		var wg sync.WaitGroup
		var totalCount int
		var languages []string
		var counts []int
		var countErr, dataErr error

		// Goroutine for count query
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := db.QueryRow(countQuery, params...).Scan(&totalCount)
			if err != nil {
				countErr = err
			}
		}()

		// Goroutine for data query
		wg.Add(1)
		go func() {
			defer wg.Done()
			rows, err := db.Query(dataQuery, dataParams...)
			if err != nil {
				dataErr = err
				return
			}
			defer rows.Close()

			for rows.Next() {
				var language string
				var count int
				if err := rows.Scan(&language, &count); err != nil {
					dataErr = err
					return
				}
				languages = append(languages, language)
				counts = append(counts, count)
			}

			if err := rows.Err(); err != nil {
				dataErr = err
			}
		}()

		// Wait for both goroutines to finish
		wg.Wait()

		// Check for errors
		if countErr != nil {
			log.Println("Error getting total count:", countErr)
			http.Error(w, countErr.Error(), http.StatusInternalServerError)
			return
		}
		if dataErr != nil {
			log.Println("Error getting language data:", dataErr)
			http.Error(w, dataErr.Error(), http.StatusInternalServerError)
			return
		}

		// Prepare and send the JSON response
		jsonStats, err := json.Marshal(map[string]interface{}{
			"languages":  languages,
			"counts":     counts,
			"totalCount": totalCount,
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

		// Initialize query and parameters
		baseQuery := "SELECT country, COUNT(*) FROM visits WHERE website_domain = $1 AND timestamp BETWEEN $2 AND $3"
		countQuery := "SELECT COUNT(DISTINCT country) FROM visits WHERE website_domain = $1 AND timestamp BETWEEN $2 AND $3"
		params := []interface{}{domain, start, end}
		paramIndex := 4 // Start the parameter index at 4 because $1, $2, and $3 are already used

		// Map query parameter names to column names
		filters := map[string]string{
			"referrer":    "referrer",
			"pathname":    "pathname",
			"device_type": "device_type",
			"os":          "os",
			"browser":     "browser",
			"language":    "language",
			"country":     "country",
			"city":        "city",
			"region":      "region",
		}

		// Add filters to the query
		for param, column := range filters {
			value := r.URL.Query().Get(param)
			if value != "" {
				baseQuery += fmt.Sprintf(" AND %s = $%d", column, paramIndex)
				countQuery += fmt.Sprintf(" AND %s = $%d", column, paramIndex)
				params = append(params, value)
				paramIndex++
			}
		}

		// Complete the query
		dataQuery := baseQuery + fmt.Sprintf(" GROUP BY country ORDER BY COUNT(*) DESC LIMIT $%d OFFSET $%d", paramIndex, paramIndex+1)
		dataParams := append(params, limit, offset)

		var wg sync.WaitGroup
		var totalCount int
		var countries []string
		var counts []int
		var countErr, dataErr error

		// Goroutine for count query
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := db.QueryRow(countQuery, params...).Scan(&totalCount)
			if err != nil {
				countErr = err
			}
		}()

		// Goroutine for data query
		wg.Add(1)
		go func() {
			defer wg.Done()
			rows, err := db.Query(dataQuery, dataParams...)
			if err != nil {
				dataErr = err
				return
			}
			defer rows.Close()

			for rows.Next() {
				var country string
				var count int
				if err := rows.Scan(&country, &count); err != nil {
					dataErr = err
					return
				}
				countries = append(countries, country)
				counts = append(counts, count)
			}

			if err := rows.Err(); err != nil {
				dataErr = err
			}
		}()

		// Wait for both goroutines to finish
		wg.Wait()

		// Check for errors
		if countErr != nil {
			log.Println("Error getting total count:", countErr)
			http.Error(w, countErr.Error(), http.StatusInternalServerError)
			return
		}
		if dataErr != nil {
			log.Println("Error getting country data:", dataErr)
			http.Error(w, dataErr.Error(), http.StatusInternalServerError)
			return
		}

		// Prepare and send the JSON response
		jsonStats, err := json.Marshal(map[string]interface{}{
			"countries":  countries,
			"counts":     counts,
			"totalCount": totalCount,
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

		// Initialize query and parameters
		baseQuery := "SELECT region, COUNT(*) FROM visits WHERE website_domain = $1 AND timestamp BETWEEN $2 AND $3"
		countQuery := "SELECT COUNT(DISTINCT region) FROM visits WHERE website_domain = $1 AND timestamp BETWEEN $2 AND $3"
		params := []interface{}{domain, start, end}
		paramIndex := 4 // Start the parameter index at 4 because $1, $2, and $3 are already used

		// Map query parameter names to column names
		filters := map[string]string{
			"referrer":    "referrer",
			"pathname":    "pathname",
			"device_type": "device_type",
			"os":          "os",
			"browser":     "browser",
			"language":    "language",
			"country":     "country",
			"city":        "city",
			"region":      "region",
		}

		// Add filters to the query
		for param, column := range filters {
			value := r.URL.Query().Get(param)
			if value != "" {
				baseQuery += fmt.Sprintf(" AND %s = $%d", column, paramIndex)
				countQuery += fmt.Sprintf(" AND %s = $%d", column, paramIndex)
				params = append(params, value)
				paramIndex++
			}
		}

		// Complete the query
		dataQuery := baseQuery + fmt.Sprintf(" GROUP BY region ORDER BY COUNT(*) DESC LIMIT $%d OFFSET $%d", paramIndex, paramIndex+1)
		dataParams := append(params, limit, offset)

		var wg sync.WaitGroup
		var totalCount int
		var regions []string
		var counts []int
		var countErr, dataErr error

		// Goroutine for count query
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := db.QueryRow(countQuery, params...).Scan(&totalCount)
			if err != nil {
				countErr = err
			}
		}()

		// Goroutine for data query
		wg.Add(1)
		go func() {
			defer wg.Done()
			rows, err := db.Query(dataQuery, dataParams...)
			if err != nil {
				dataErr = err
				return
			}
			defer rows.Close()

			for rows.Next() {
				var region string
				var count int
				if err := rows.Scan(&region, &count); err != nil {
					dataErr = err
					return
				}
				regions = append(regions, region)
				counts = append(counts, count)
			}

			if err := rows.Err(); err != nil {
				dataErr = err
			}
		}()

		// Wait for both goroutines to finish
		wg.Wait()

		// Check for errors
		if countErr != nil {
			log.Println("Error getting total count:", countErr)
			http.Error(w, countErr.Error(), http.StatusInternalServerError)
			return
		}
		if dataErr != nil {
			log.Println("Error getting region data:", dataErr)
			http.Error(w, dataErr.Error(), http.StatusInternalServerError)
			return
		}

		// Prepare and send the JSON response
		jsonStats, err := json.Marshal(map[string]interface{}{
			"regions":    regions,
			"counts":     counts,
			"totalCount": totalCount,
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

		// Initialize query and parameters
		baseQuery := "SELECT city, COUNT(*) FROM visits WHERE website_domain = $1 AND timestamp BETWEEN $2 AND $3"
		countQuery := "SELECT COUNT(DISTINCT city) FROM visits WHERE website_domain = $1 AND timestamp BETWEEN $2 AND $3"
		params := []interface{}{domain, start, end}
		paramIndex := 4 // Start the parameter index at 4 because $1, $2, and $3 are already used

		// Map query parameter names to column names
		filters := map[string]string{
			"referrer":    "referrer",
			"pathname":    "pathname",
			"device_type": "device_type",
			"os":          "os",
			"browser":     "browser",
			"language":    "language",
			"country":     "country",
			"city":        "city",
			"region":      "region",
		}

		// Add filters to the query
		for param, column := range filters {
			value := r.URL.Query().Get(param)
			if value != "" {
				baseQuery += fmt.Sprintf(" AND %s = $%d", column, paramIndex)
				countQuery += fmt.Sprintf(" AND %s = $%d", column, paramIndex)
				params = append(params, value)
				paramIndex++
			}
		}

		// Complete the query
		dataQuery := baseQuery + fmt.Sprintf(" GROUP BY city ORDER BY COUNT(*) DESC LIMIT $%d OFFSET $%d", paramIndex, paramIndex+1)
		dataParams := append(params, limit, offset)

		var wg sync.WaitGroup
		var totalCount int
		var cities []string
		var counts []int
		var countErr, dataErr error

		// Goroutine for count query
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := db.QueryRow(countQuery, params...).Scan(&totalCount)
			if err != nil {
				countErr = err
			}
		}()

		// Goroutine for data query
		wg.Add(1)
		go func() {
			defer wg.Done()
			rows, err := db.Query(dataQuery, dataParams...)
			if err != nil {
				dataErr = err
				return
			}
			defer rows.Close()

			for rows.Next() {
				var city string
				var count int
				if err := rows.Scan(&city, &count); err != nil {
					dataErr = err
					return
				}
				cities = append(cities, city)
				counts = append(counts, count)
			}

			if err := rows.Err(); err != nil {
				dataErr = err
			}
		}()

		// Wait for both goroutines to finish
		wg.Wait()

		// Check for errors
		if countErr != nil {
			log.Println("Error getting total count:", countErr)
			http.Error(w, countErr.Error(), http.StatusInternalServerError)
			return
		}
		if dataErr != nil {
			log.Println("Error getting city data:", dataErr)
			http.Error(w, dataErr.Error(), http.StatusInternalServerError)
			return
		}

		// Prepare and send the JSON response
		jsonStats, err := json.Marshal(map[string]interface{}{
			"cities":     cities,
			"counts":     counts,
			"totalCount": totalCount,
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

func GetUTMParameters(db *sql.DB, utm_parameter string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Extract the domain from the url
		domain, err := utils.ExtractDomainFromURL(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
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

		// Dynamic query construction using the utm_parameter
		baseQuery := fmt.Sprintf("SELECT %s, COUNT(*) FROM visits WHERE website_domain = $1 AND timestamp BETWEEN $2 AND $3 AND %s IS NOT NULL AND %s != ''", utm_parameter, utm_parameter, utm_parameter)
		countQuery := fmt.Sprintf("SELECT COUNT(DISTINCT %s) FROM visits WHERE website_domain = $1 AND timestamp BETWEEN $2 AND $3 AND %s IS NOT NULL AND %s != ''", utm_parameter, utm_parameter, utm_parameter)
		params := []interface{}{domain, start, end}
		paramIndex := 4 // Start the parameter index at 4 because $1, $2, and $3 are already used

		// Map query parameter names to column names
		filters := map[string]string{
			"referrer":     "referrer",
			"pathname":     "pathname",
			"device_type":  "device_type",
			"os":           "os",
			"browser":      "browser",
			"language":     "language",
			"country":      "country",
			"city":         "city",
			"region":       "region",
			"utm_source":   "utm_source",
			"utm_medium":   "utm_medium",
			"utm_campaign": "utm_campaign",
			"utm_term":     "utm_term",
			"utm_content":  "utm_content",
		}

		// Add filters to the query
		for param, column := range filters {
			value := r.URL.Query().Get(param)
			if value != "" {
				baseQuery += fmt.Sprintf(" AND %s = $%d", column, paramIndex)
				countQuery += fmt.Sprintf(" AND %s = $%d", column, paramIndex)
				params = append(params, value)
				paramIndex++
			}
		}

		// Complete the query
		dataQuery := baseQuery + fmt.Sprintf(" GROUP BY %s ORDER BY COUNT(*) DESC LIMIT $%d OFFSET $%d", utm_parameter, paramIndex, paramIndex+1)
		dataParams := append(params, limit, offset)

		var wg sync.WaitGroup
		var totalCount int
		var utm_values []string
		var counts []int
		var countErr, dataErr error

		// Goroutine for count query
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := db.QueryRow(countQuery, params...).Scan(&totalCount)
			if err != nil {
				countErr = err
			}
		}()

		// Goroutine for data query
		wg.Add(1)
		go func() {
			defer wg.Done()
			rows, err := db.Query(dataQuery, dataParams...)
			if err != nil {
				dataErr = err
				return
			}
			defer rows.Close()

			for rows.Next() {
				var utm_value string
				var count int
				if err := rows.Scan(&utm_value, &count); err != nil {
					dataErr = err
					return
				}
				utm_values = append(utm_values, utm_value)
				counts = append(counts, count)
			}

			if err := rows.Err(); err != nil {
				dataErr = err
			}
		}()

		// Wait for both goroutines to finish
		wg.Wait()

		// Check for errors
		if countErr != nil {
			log.Println("Error getting total count:", countErr)
			http.Error(w, countErr.Error(), http.StatusInternalServerError)
			return
		}
		if dataErr != nil {
			log.Println("Error getting utm parameters data:", dataErr)
			http.Error(w, dataErr.Error(), http.StatusInternalServerError)
			return
		}

		// Prepare and send the JSON response
		jsonStats, err := json.Marshal(map[string]interface{}{
			"utm_values": utm_values,
			"counts":     counts,
			"totalCount": totalCount,
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
