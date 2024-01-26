package models

import "time"

// VisitInsert represents the structure for inserting new visit records.
type VisitInsert struct {
	Timestamp    time.Time `json:"timestamp"`
	Referrer     string    `json:"referrer"`
	URL          string    `json:"url"`
	Pathname     string    `json:"pathname"`
	Hash         string    `json:"hash"`
	UserAgent    string    `json:"userAgent"`
	Language     string    `json:"language"`
	ScreenWidth  int       `json:"screenWidth"`
	ScreenHeight int       `json:"screenHeight"`
	Location     string    `json:"location"`
	WebsiteID    int       `json:"websiteId"` // Foreign key to Website model
}

// Visit represents the structure for retrieving visit records.
type Visit struct {
	ID           int       `json:"id"`
	Timestamp    time.Time `json:"timestamp"`
	Referrer     string    `json:"referrer"`
	URL          string    `json:"url"`
	Pathname     string    `json:"pathname"`
	Hash         string    `json:"hash"`
	UserAgent    string    `json:"userAgent"`
	Language     string    `json:"language"`
	ScreenWidth  int       `json:"screenWidth"`
	ScreenHeight int       `json:"screenHeight"`
	Location     string    `json:"location"`
	WebsiteID    int       `json:"websiteId"` // Foreign key to Website model
}
