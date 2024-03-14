package models

import (
	"database/sql"
	"encoding/json"
	"time"
)

type Website struct {
	ID        sql.NullInt64  `json:"id"`
	Domain    sql.NullString `json:"domain"`
	UserID    sql.NullInt64  `json:"userId"` // Foreign key to User model
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
}

type WebsiteInsert struct {
	Domain    string    `json:"domain"`
	UserID    int       `json:"userId"` // Foreign key to User model
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type WebsiteUpdateResponse struct {
	ID     int64  `json:"id"`
	Domain string `json:"domain"`
	UserID int    `json:"userId"` // Foreign key to User model
}

// This method is used to control how the Website struct is converted into JSON. Before using this function the json reponse also included the Valid key for each nullable field in the Website struct. Now it's not included anymore. The MarshalJSON is a special method in Go that gets automatically triggered on json marshalling.
func (w *Website) MarshalJSON() ([]byte, error) {
	type Alias Website
	return json.Marshal(&struct {
		ID        int64     `json:"id"`
		Domain    string    `json:"domain"`
		UserID    int64     `json:"userId"`
		CreatedAt time.Time `json:"created_at"`
		UpdatedAt time.Time `json:"updated_at"`
		*Alias
	}{
		ID:        w.ID.Int64,
		Domain:    w.Domain.String,
		UserID:    w.UserID.Int64,
		CreatedAt: w.CreatedAt,
		UpdatedAt: w.UpdatedAt,
		Alias:     (*Alias)(w),
	})
}
