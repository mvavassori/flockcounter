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
	"path/filepath"

	_ "github.com/lib/pq"
	"github.com/mileusna/useragent"
	"github.com/mvavassori/bare-analytics/models"
	"github.com/mvavassori/bare-analytics/utils"

	"github.com/oschwald/geoip2-golang"
)

func GetVisits(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		// Perform the SELECT query to get all visits
		rows, err := db.Query("SELECT id, website_id, website_domain, timestamp, referrer, url, pathname, device_type, os, browser, language, country, region, city, time_spent_on_page, is_unique, utm_source, utm_medium, utm_campaign, utm_term, utm_content FROM visits")
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
			err := rows.Scan(&visit.ID, &visit.WebsiteID, &visit.WebsiteDomain, &visit.Timestamp, &visit.Referrer, &visit.URL, &visit.Pathname, &visit.DeviceType, &visit.OS, &visit.Browser, &visit.Language, &visit.Country, &visit.Region, &visit.City, &visit.TimeSpentOnPage, &visit.IsUnique, &visit.UTMSource, &visit.UTMMedium, &visit.UTMCampaign, &visit.UTMTerm, &visit.UTMContent)
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

func CreateVisit(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		// todo uncomment this when in production
		// //Get IP address
		// ip, _, err := net.SplitHostPort(r.RemoteAddr)
		// if err != nil {
		// 	log.Println("Error getting ip from remote addr", err)
		// } else {
		// 	fmt.Println("Received request from IP:", ip)
		// }

		// todo make a separate function to get location data
		// Get home directory
		homeDir, err := os.UserHomeDir()
		if err != nil {
			log.Fatal("Error getting home directory:", err)
		}

		// Construct full path to GeoLite2-City.mmdb file
		dbPath := filepath.Join(homeDir, ".geoip2", "GeoLite2-City.mmdb")

		geoip2DB, err := geoip2.Open(dbPath)
		if err != nil {
			log.Fatal("Error initilizing geoip2 database", err)
		}
		defer geoip2DB.Close()

		// todo: enable this in production
		// parsedIP := net.ParseIP(ip)

		// for testing just use this one ip address
		parsedIP := net.ParseIP("151.30.13.167")

		if parsedIP == nil {
			log.Println("Error parsing IP", err)
			http.Error(w, "Invalid IP format", http.StatusBadRequest)
			return
		}

		record, err := geoip2DB.City(parsedIP)
		if err != nil {
			log.Println("Error retrieving location", err)
		}

		// Default values if country, region, or city are not found
		country := "Unknown"
		region := "Unknown"
		city := "Unknown"

		// Retrieve country name if available
		if countryName, ok := record.Country.Names["en"]; ok {
			country = countryName
		}

		// Retrieve region name if available
		if len(record.Subdivisions) > 0 {
			if regionName, ok := record.Subdivisions[0].Names["en"]; ok {
				region = regionName
			}
		} else {
			log.Println("No subdivision information available")
		}

		// Retrieve city name if available
		if cityName, ok := record.City.Names["en"]; ok {
			city = cityName
		}

		// Create a VisitReceiver struct to hold the request data
		var visitReceiver models.VisitReceiver

		// Decode the JSON data from the request body into the VisitReceiver struct
		err = json.NewDecoder(r.Body).Decode(&visitReceiver) // The Decode function modifies the contents of the passed object based on the input JSON data. By passing a pointer, any changes made by Decode will directly update the original struct rather than creating a copy and updating that.
		if err != nil {
			log.Println("Error decoding input data", err)
			http.Error(w, "Invalid JSON format", http.StatusBadRequest)
			return
		}

		ua := useragent.Parse(visitReceiver.UserAgent)

		url, err := url.Parse(visitReceiver.URL)
		if err != nil {
			log.Println("Error parsing URL", err)
			http.Error(w, "Invalid URL format", http.StatusBadRequest)
			return
		}

		domain := url.Hostname()

		utmSource := url.Query().Get("utm_source")
		utmMedium := url.Query().Get("utm_medium")
		utmCampaign := url.Query().Get("utm_campaign")
		utmTerm := url.Query().Get("utm_term")
		utmContent := url.Query().Get("utm_content")

		// extract the referrer
		referrer := visitReceiver.Referrer

		// Check if the referrer is empty or null
		if referrer == "" {
			referrer = "Direct"
		} else if referrer == "Direct" {
			// Explicitly handle the case where the referrer is "Direct" (my script sends it when ther's no referrer)
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
		err = db.QueryRow("SELECT id FROM websites WHERE domain = $1", domain).Scan(&websiteId)
		if err != nil {
			log.Println("Error looking up websiteId", err)
			http.Error(w, "Website not found", http.StatusNotFound)
			return
		}

		// Generate daily salt or grab from cache if already generated
		dailySalt, err := utils.GenerateDailySalt()
		if err != nil {
			fmt.Println(err)
			return
		}

		// Generate a unique identifier
		uniqueIdentifier, err := utils.GenerateUniqueIdentifier(dailySalt, domain, "45.14.71.8", visitReceiver.UserAgent) // todo: change to ip address variable later
		if err != nil {
			fmt.Println(err)
			return
		}

		// Check if the unique identifier exists in the daily_unique_identifiers table
		var isUnique bool
		err = db.QueryRow("SELECT EXISTS(SELECT 1 FROM daily_unique_identifiers WHERE unique_identifier = $1)", uniqueIdentifier).Scan(&isUnique)
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
			_, err := db.Exec("INSERT INTO daily_unique_identifiers (unique_identifier) VALUES ($1)", uniqueIdentifier)
			if err != nil {
				log.Println("Error inserting unique identifier", err)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}
			isUnique = true
		}

		// Create a VisitInsert struct to hold the data to be inserted into the database
		visit := models.VisitInsert{
			Timestamp:       visitReceiver.Timestamp,
			Referrer:        referrer,
			URL:             visitReceiver.URL,
			Pathname:        visitReceiver.Pathname,
			DeviceType:      utils.GetDeviceType(&ua),
			OS:              ua.OS,
			Browser:         ua.Name,
			Language:        visitReceiver.Language,
			Country:         country,
			Region:          region,
			City:            city,
			IsUnique:        isUnique,
			TimeSpentOnPage: visitReceiver.TimeSpentOnPage,
			UTMSource: sql.NullString{
				String: utmSource,
				Valid:  utmSource != "",
			},
			UTMMedium: sql.NullString{
				String: utmMedium,
				Valid:  utmMedium != "",
			},
			UTMCampaign: sql.NullString{
				String: utmCampaign,
				Valid:  utmCampaign != "",
			},
			UTMTerm: sql.NullString{
				String: utmTerm,
				Valid:  utmTerm != "",
			},
			UTMContent: sql.NullString{
				String: utmContent,
				Valid:  utmContent != "",
			},
		}

		// Perform the INSERT query to add the new visit to the database
		insertQuery := `
			INSERT INTO visits
				(website_id, website_domain, timestamp, referrer, url, pathname, device_type, os, browser, language, country, region, city, is_unique, time_spent_on_page, utm_source, utm_medium, utm_campaign, utm_term, utm_content)
			VALUES
				($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20);
		`
		_, err = db.Exec(insertQuery,
			websiteId,
			domain,
			visit.Timestamp,
			visit.Referrer,
			visit.URL,
			visit.Pathname,
			visit.DeviceType,
			visit.OS,
			visit.Browser,
			visit.Language,
			visit.Country,
			visit.Region,
			visit.City,
			visit.IsUnique,
			visit.TimeSpentOnPage,
			visit.UTMSource,
			visit.UTMMedium,
			visit.UTMCampaign,
			visit.UTMTerm,
			visit.UTMContent,
		)
		if err != nil {
			fmt.Println("Error inserting visit:", err)
			return
		}

		w.WriteHeader(http.StatusCreated)
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
