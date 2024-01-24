package db

import (
	"database/sql"
	"log"

	_ "github.com/lib/pq" // imported for side-effects only, not for direct use in the code.
)

func CreateDBConnection() (*sql.DB, error) {
	connStr := "user=postgres password=postgres host=localhost port=5432 dbname=bareanalyticsdb sslmode=disable"
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
