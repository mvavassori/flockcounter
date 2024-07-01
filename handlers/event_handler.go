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

	"github.com/mileusna/useragent"
	"github.com/mvavassori/bare-analytics/utils"
	"github.com/oschwald/geoip2-golang"

	"github.com/mvavassori/bare-analytics/models"
)

func MakeEvent(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

	}
}

// todo: Will be displayed in the dashboard or a dedicated different section/page
func GetEvents(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// extract the value id from the url
		domain, err := utils.ExtractDomainFromURL(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		var exists bool
		err = db.QueryRow("SELECT EXISTS(SELECT 1 FROM websites WHERE domain = $1)", domain).Scan(&exists)
		if err != nil {
			log.Println("Error checking if website exists:", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if !exists {
			http.Error(w, fmt.Sprintf("Website with domain %s doesn't exist", domain), http.StatusBadRequest)
			return
		}
	}

	// here i should return back three main metrics: total people who have completed the goal, unique people who have completed the goal, and Conversion rate is calculated as the number of unique visitors who have achieved the goal divided by the total number of unique visitors to the website
}

//? GetEvent <- should i add also a way to display data for a single event?

func CreateEvent(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		// todo
		// //Get IP address
		// ip, _, err := net.SplitHostPort(r.RemoteAddr)
		// if err != nil {
		// 	log.Println("Error getting ip from remote addr", err)
		// } else {
		// 	fmt.Println("Received request from IP:", ip)
		// }

		// Get home directory
		homeDir, err := os.UserHomeDir()
		if err != nil {
			log.Fatal("Error getting home directory:", err)
		}

		fmt.Println("Home directory:", homeDir)
		// Construct full path to GeoLite2-City.mmdb file
		dbPath := filepath.Join(homeDir, ".geoip2", "GeoLite2-City.mmdb")

		fmt.Println("Database path:", dbPath)

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

		fmt.Println("Parsed IP:", parsedIP)

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

		// create a EventReceiver struct to hold the request data
		var eventReceiver models.EventReceiver
		err = json.NewDecoder(r.Body).Decode(&eventReceiver)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		ua := useragent.Parse(eventReceiver.UserAgent)

		fmt.Println(ua)

		url, err := url.Parse(eventReceiver.URL)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		domain := url.Hostname()
		fmt.Println(domain)

		// extract the referrer
		referrer := eventReceiver.Referrer

		// remove the protocol from the referrer
		u, err := url.Parse(referrer)
		if err != nil {
			log.Println("Error parsing referrer", err)
			http.Error(w, "Invalid referrer format", http.StatusBadRequest)
			return
		}

		referrerWithoutProtocol := u.Host + u.Path

		fmt.Println("referrerWithoutProtocol", referrerWithoutProtocol)

		fmt.Println("Frontend sent: ", eventReceiver)

		// Look up the websiteId using the domain
		var websiteId int
		err = db.QueryRow("SELECT id FROM websites WHERE domain = $1", domain).Scan(&websiteId)
		if err != nil {
			log.Println("Error looking up websiteId", err)
			http.Error(w, "Website not found", http.StatusNotFound)
			return
		}

		// todo: check isUnique

		event := models.EventInsert{
			Type:       eventReceiver.Type,
			Timestamp:  eventReceiver.Timestamp,
			Referrer:   referrerWithoutProtocol,
			URL:        eventReceiver.URL,
			Pathname:   eventReceiver.Pathname,
			DeviceType: utils.GetDeviceType(&ua),
			OS:         ua.OS,
			Browser:    ua.Name,
			Language:   eventReceiver.Language,
			Country:    country,
			Region:     region,
			City:       city,
			IsUnique:   true, // todo: check isUnique
		}

		fmt.Println(event)

		// perform the INSERT query to insert the event into the database

	}
}
