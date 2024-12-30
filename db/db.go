package db

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"

	_ "github.com/lib/pq" // imported for side-effects only, not for direct use in the code.
	"github.com/oschwald/geoip2-golang"
)

func CreatePostgresConnection() (*sql.DB, error) {
	connStr := fmt.Sprintf("user=%s password=%s host=%s port=%s dbname=%s sslmode=%s",
		os.Getenv("POSTGRES_USER"),
		os.Getenv("POSTGRES_PASSWORD"),
		os.Getenv("POSTGRES_HOST"),
		os.Getenv("POSTGRES_PORT"),
		os.Getenv("POSTGRES_DB"),
		os.Getenv("POSTGRES_SSLMODE"),
	)
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, err
	}

	err = db.Ping()
	if err != nil {
		return nil, err
	}

	log.Println("Successfully connected to the Postgres Database")

	return db, nil
}

func CreateGeoIPConnection() (*geoip2.Reader, error) {
	dbPath := os.Getenv("GEOIP_DB_PATH")
	if dbPath == "" {
		// Fallback to local development path if env var not set
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("home directory error: %w", err)
		}
		dbPath = filepath.Join(homeDir, ".geoip2", "GeoLite2-City.mmdb")
	}

	db, err := geoip2.Open(dbPath)
	if err != nil {
		return nil, fmt.Errorf("geoip connection error: %w", err)
	}

	log.Println("Successfully connected to GeoIP Database")
	return db, nil
}
