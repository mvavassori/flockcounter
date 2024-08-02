package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"

	// "io"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"

	// "time"

	// "sync"

	_ "github.com/lib/pq"
	"github.com/mileusna/useragent"
	"github.com/mvavassori/bare-analytics/models"
	"github.com/mvavassori/bare-analytics/utils"

	"github.com/oschwald/geoip2-golang"
)

func GetVisits(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		// Perform the SELECT query to get all visits
		rows, err := db.Query("SELECT id, website_id, website_domain, timestamp, referrer, url, pathname, device_type, os, browser, language, country, region, city, time_spent_on_page, is_unique FROM visits")
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
			err := rows.Scan(&visit.ID, &visit.WebsiteID, &visit.WebsiteDomain, &visit.Timestamp, &visit.Referrer, &visit.URL, &visit.Pathname, &visit.DeviceType, &visit.OS, &visit.Browser, &visit.Language, &visit.Country, &visit.Region, &visit.City, &visit.TimeSpentOnPage, &visit.IsUnique)
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

func GetVisit(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Extract the value of the 'id' variable from the URL path
		id, err := utils.ExtractIDFromURL(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// Perform the SELECT query to get the visit with the specified ID
		row := db.QueryRow("SELECT * FROM visits WHERE id = $1", id)

		// Create a Visit struct to hold the retrieved data
		var visit models.Visit
		// row.Scan copies the column values from the matched row into the provided variables, each field in the Visit struct corresponds to a column in the "visits" table.
		// It reads the values from the database row and populates the fields in the visit variable.
		err = row.Scan(
			&visit.ID,
			&visit.WebsiteID,
			&visit.WebsiteDomain,
			&visit.Timestamp,
			&visit.Referrer,
			&visit.URL,
			&visit.Pathname,
			&visit.DeviceType,
			&visit.OS,
			&visit.Browser,
			&visit.Language,
			&visit.Country,
			&visit.Region,
			&visit.City,
			&visit.TimeSpentOnPage,
			&visit.IsUnique,
		)

		if err == sql.ErrNoRows {
			http.Error(w, fmt.Sprintf("Visit with id %d doesn't exist", id), http.StatusNotFound)
			return
		} else if err != nil {
			log.Println("Error retrieving visit:", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		// Marshal the visit data to JSON
		jsonResponse, err := json.Marshal(visit)
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

		// textData, err := io.ReadAll(r.Body)
		// if err != nil {
		// 	http.Error(w, err.Error(), http.StatusBadRequest)
		// 	return
		// }

		// // Print the text data
		// fmt.Println("Received text data:")
		// fmt.Println(string(textData))

		// todo
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

		// fmt.Println("Home directory:", homeDir)
		// Construct full path to GeoLite2-City.mmdb file
		dbPath := filepath.Join(homeDir, ".geoip2", "GeoLite2-City.mmdb")

		// fmt.Println("Database path:", dbPath)

		geoip2DB, err := geoip2.Open(dbPath)
		if err != nil {
			log.Fatal("Error initilizing geoip2 database", err)
		}
		defer geoip2DB.Close()

		// todo
		// parsedIP := net.ParseIP(ip)
		// for testing
		parsedIP := net.ParseIP("45.14.71.8")

		if parsedIP == nil {
			log.Println("Error parsing IP", err)
			http.Error(w, "Invalid IP format", http.StatusBadRequest)
			return
		}

		// fmt.Println("Parsed IP:", parsedIP)

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

		// Print location information
		// fmt.Println("Country:", country)
		// fmt.Println("Region:", region)
		// fmt.Println("City:", city)

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

		// fmt.Println(domain)

		// extract the referrer
		referrer := visitReceiver.Referrer

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

		// fmt.Println("Frontend sent: ", visitReceiver)

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

		// fmt.Println(uniqueIdentifier)

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
		}

		// Perform the INSERT query to add the new visit to the database
		insertQuery := `
			INSERT INTO visits
				(website_id, website_domain , timestamp, referrer, url, pathname, device_type, os, browser, language, country, region, city, is_unique, time_spent_on_page)
			VALUES
				($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15);
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
		)
		if err != nil {
			fmt.Println("Error inserting visit:", err)
			return
		}

		w.WriteHeader(http.StatusCreated)
	}
}

func UpdateVisit(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Extract the value of the 'id' variable from the URL path
		id, err := utils.ExtractIDFromURL(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// Read the request body to get the updated visit data
		var updatedVisit models.VisitInsert
		err = json.NewDecoder(r.Body).Decode(&updatedVisit)
		if err != nil {
			log.Println("Error decoding JSON:", err)
			http.Error(w, "Invalid JSON format", http.StatusBadRequest)
			return
		}

		url, err := url.Parse(updatedVisit.URL)
		if err != nil {
			log.Println("Error parsing URL", err)
			http.Error(w, "Invalid URL format", http.StatusBadRequest)
			return
		}
		domain := url.Hostname()

		fmt.Println(domain)

		// Look up the websiteId using the domain
		var websiteId int
		err = db.QueryRow("SELECT id FROM websites WHERE domain = $1", domain).Scan(&websiteId)
		if err != nil {
			log.Println("Error looking up websiteId", err)
			http.Error(w, "Website not found", http.StatusNotFound)
			return
		}

		updateQuery := `
			UPDATE visits
			SET website_id = $1, website_domain = $2, timestamp = $3, referrer = $4, url = $5, pathname = $6, device_type = $7, os = $8, browser = $9, language = $10, country = $11, region = $12, city = $13, is_unique = $14, time_spent_on_page = $15
			WHERE id = $16;
		`
		// Perform the UPDATE query to modify the visit with the specified ID
		result, err := db.Exec(updateQuery,
			websiteId,
			domain,
			updatedVisit.Timestamp,
			updatedVisit.Referrer,
			updatedVisit.URL,
			updatedVisit.Pathname,
			updatedVisit.DeviceType,
			updatedVisit.OS,
			updatedVisit.Browser,
			updatedVisit.Language,
			updatedVisit.Country,
			updatedVisit.Region,
			updatedVisit.City,
			updatedVisit.IsUnique,
			updatedVisit.TimeSpentOnPage,
			id,
		)
		if err != nil {
			log.Println("Error updating visit:", err)
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

		// Create a VisitUpdateResponse object to return in the response
		visitUpdateResponse := models.VisitUpdateResponse{
			VisitInsert:   updatedVisit,
			ID:            int(id),
			WebsiteID:     int(websiteId),
			WebsiteDomain: domain,
		}

		// Convert the visitResponse object to JSON
		jsonResponse, err := json.Marshal(visitUpdateResponse)
		if err != nil {
			log.Println("Error encoding JSON:", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		// Set the content type header and write the response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(jsonResponse)
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
