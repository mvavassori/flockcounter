package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/mileusna/useragent"
	"github.com/mvavassori/bare-analytics/utils"
	"github.com/oschwald/geoip2-golang"

	"github.com/mvavassori/bare-analytics/models"
)

// Will be displayed in the dashboard or a dedicated different section/page
func GetEvents(postgresDB *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// extract the value id from the url
		domain, err := utils.ExtractDomainFromURL(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		var exists bool
		err = postgresDB.QueryRow("SELECT EXISTS(SELECT 1 FROM websites WHERE domain = $1)", domain).Scan(&exists)
		if err != nil {
			log.Println("Error checking if website exists:", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if !exists {
			http.Error(w, fmt.Sprintf("Website with domain %s doesn't exist", domain), http.StatusBadRequest)
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

		// Query the database for events within the specified date range
		query := `
			SELECT name, COUNT(*) as counts
			FROM events
			WHERE website_domain = $1 AND timestamp BETWEEN $2 AND $3
			GROUP BY name
		`

		rows, err := postgresDB.Query(query, domain, start, end)
		if err != nil {
			log.Println("Error querying events:", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		// Convert the statistics to JSON
		defer rows.Close() // Close the result set after we're done with it
		var eventName string
		var count int
		var eventNames []string
		var counts []int
		for rows.Next() {
			err = rows.Scan(&eventName, &count)
			if err != nil {
				log.Println("Error scanning statistics:", err)
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			eventNames = append(eventNames, eventName)
			counts = append(counts, count)
		}
		jsonStats, err := json.Marshal(map[string]interface{}{
			"eventNames": eventNames,
			"counts":     counts,
		})
		if err != nil {
			log.Println("Error marshalling statistics:", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(jsonStats)
		// here i should return back three main metrics: total people who have completed the goal, unique people who have completed the goal, and Conversion rate is calculated as the number of unique visitors who have achieved the goal divided by the total number of unique visitors to the website
	}

}

func CreateEvent(postgresDB *sql.DB, geoipDB *geoip2.Reader) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		var parsedIP net.IP
		if os.Getenv("ENV") == "production" {
			// Try different headers first, then fall back to RemoteAddr
			ipAddress := utils.GetIPAddress(r)
			if ipAddress == "" {
				log.Println("Could not determine IP address")
				http.Error(w, "Could not determine IP address", http.StatusInternalServerError)
				return
			}
			parsedIP = net.ParseIP(ipAddress)
		} else {
			parsedIP = net.ParseIP("151.30.13.167") // test IP
		}

		if parsedIP == nil {
			log.Println("Invalid IP format")
			http.Error(w, "Invalid IP format", http.StatusBadRequest)
			return
		}

		record, err := geoipDB.City(parsedIP)
		if err != nil {
			log.Printf("Error retrieving location for IP %v: %v", parsedIP, err)
			http.Error(w, "Error retrieving location", http.StatusInternalServerError)
			return
		}

		location := utils.GetLocationInfo(record)

		// create a EventReceiver struct to hold the request data
		var eventReceiver models.EventReceiver
		err = json.NewDecoder(r.Body).Decode(&eventReceiver)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		ua := useragent.Parse(eventReceiver.UserAgent)

		url, err := url.Parse(eventReceiver.URL)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		domain := url.Hostname()

		// extract the referrer
		referrer := eventReceiver.Referrer

		// Check if the referrer is empty or null
		if referrer == "" {
			referrer = "Direct"
		} else {
			// Remove the protocol from the referrer
			referrerURL, err := url.Parse(referrer)
			if err != nil {
				log.Println("Error parsing referrer:", err)
				http.Error(w, "Invalid referrer format", http.StatusBadRequest)
				return
			}

			referrer = referrerURL.Host + referrerURL.Path
		}

		// Look up the websiteId using the domain
		var websiteId int
		err = postgresDB.QueryRow("SELECT id FROM websites WHERE domain = $1", domain).Scan(&websiteId)
		if err != nil {
			log.Println("Error looking up websiteId", err)
			http.Error(w, "Website not found", http.StatusNotFound)
			return
		}

		// Generate daily salt or grab from cache if already generated
		dailySalt, err := utils.GenerateDailySalt()
		if err != nil {
			log.Println("Error generating or grabbing daily salt", err)
			return
		}

		// Generate a unique identifier
		uniqueIdentifier, err := utils.GenerateUniqueIdentifier(dailySalt, domain, "45.14.71.8", eventReceiver.UserAgent) // todo: change to ip address variable later
		if err != nil {
			log.Println("Error generating a unique identifier", err)
			return
		}

		// Check if the unique identifier exists in the daily_unique_identifiers table
		var isUnique bool
		err = postgresDB.QueryRow("SELECT EXISTS(SELECT 1 FROM daily_unique_identifiers WHERE unique_identifier = $1)", uniqueIdentifier).Scan(&isUnique)
		if err != nil {
			log.Println("Error checking for existing unique identifier", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		// Update the IsUnique field based on whether the unique identifier exists
		if isUnique {
			isUnique = false
		} else {
			// Add the unique identifier to the daily_unique_identifiers table
			_, err := postgresDB.Exec("INSERT INTO daily_unique_identifiers (unique_identifier) VALUES ($1)", uniqueIdentifier)
			if err != nil {
				log.Println("Error inserting unique identifier", err)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}
			isUnique = true
		}

		event := models.EventInsert{
			Type:       eventReceiver.Type,
			Name:       eventReceiver.Name,
			Timestamp:  eventReceiver.Timestamp,
			Referrer:   referrer,
			URL:        eventReceiver.URL,
			Pathname:   eventReceiver.Pathname,
			DeviceType: utils.GetDeviceType(&ua),
			OS:         ua.OS,
			Browser:    ua.Name,
			Language:   eventReceiver.Language,
			Country:    location.Country,
			Region:     location.Region,
			City:       location.City,
			IsUnique:   isUnique,
		}

		// perform the INSERT query to insert the event into the database
		insertQuery := `
			INSERT INTO events 
				(website_id, website_domain, type, name, timestamp, referrer, url, pathname, device_type, os, browser, language, country, region, city, is_unique)
			VALUES
				($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)
		`

		_, err = postgresDB.Exec(insertQuery,
			websiteId,
			domain,
			event.Type,
			event.Name,
			event.Timestamp,
			event.Referrer,
			event.URL,
			event.Pathname,
			event.DeviceType,
			event.OS,
			event.Browser,
			event.Language,
			event.Country,
			event.Region,
			event.City,
			event.IsUnique,
		)
		if err != nil {
			log.Println("Error inserting event", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusCreated)
	}
}
