package models

import "time"

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

type VisitReceiver struct {
	Timestamp time.Time `json:"timestamp"`
	Referrer  string    `json:"referrer"`
	URL       string    `json:"url"`
	Pathname  string    `json:"pathname"`
	// Hash         string    `json:"hash"`
	UserAgent string `json:"userAgent"`
	Language  string `json:"language"`
	// ScreenWidth  int       `json:"screenWidth"`
	// ScreenHeight int       `json:"screenHeight"`
	Country string `json:"country"`
	State   string `json:"state"`
	// WebsiteID    int       `json:"websiteId"` // Foreign key to Website model
}

// todo
type VisitInsert2 struct {
	Timestamp  time.Time `json:"timestamp"`
	Referrer   string    `json:"referrer"`
	URL        string    `json:"url"`
	Pathname   string    `json:"pathname"`
	DeviceType string    `json:"deviceType"`
	OS         string    `json:"os"`
	Browser    string    `json:"browser"`
	Language   string    `json:"language"`
	Location   string    `json:"location"`
	WebsiteID  int       `json:"websiteId"` // Foreign key to Website model
}

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

type VisitUpdateResponse struct {
	VisitInsert
	ID        int64 `json:"id"`
	WebsiteID int   `json:"websiteId"`
}
