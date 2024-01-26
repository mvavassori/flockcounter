package models

type Website struct {
	ID     int    `json:"id"`
	Domain string `json:"domain"`
	UserID int    `json:"userId"` // Foreign key to User model
}

type WebsiteInsert struct {
	Domain string `json:"domain"`
	UserID int    `json:"userId"` // Foreign key to User model
}
