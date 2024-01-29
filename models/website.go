package models

import "database/sql"

type Website struct {
	ID     sql.NullInt64  `json:"id"`
	Domain sql.NullString `json:"domain"`
	UserID sql.NullString `json:"userId"` // Foreign key to User model
}

type WebsiteInsert struct {
	Domain string `json:"domain"`
	UserID int    `json:"userId"` // Foreign key to User model
}
