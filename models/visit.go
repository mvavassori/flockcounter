package models

import "time"

type Visit struct {
	ID         int       `json:"id"`
	Timestamp  time.Time `json:"timestamp"`
	Referrer   string    `json:"referrer"`
	URL        string    `json:"url"`
	Pathname   string    `json:"pathname"`
	DeviceType string    `json:"deviceType"`
	OS         string    `json:"os"`
	Browser    string    `json:"browser"`
	Language   string    `json:"language"`
	Country    string    `json:"country"`
	State      string    `json:"state"`
	WebsiteID  int       `json:"websiteId"` // Foreign key to Website model
}

type VisitReceiver struct {
	Timestamp time.Time `json:"timestamp"`
	Referrer  string    `json:"referrer"`
	URL       string    `json:"url"`
	Pathname  string    `json:"pathname"`
	UserAgent string    `json:"userAgent"`
	Language  string    `json:"language"`
	Country   string    `json:"country"`
	State     string    `json:"state"`
}

type VisitInsert struct {
	Timestamp  time.Time `json:"timestamp"`
	Referrer   string    `json:"referrer"`
	URL        string    `json:"url"`
	Pathname   string    `json:"pathname"`
	DeviceType string    `json:"deviceType"`
	OS         string    `json:"os"`
	Browser    string    `json:"browser"`
	Language   string    `json:"language"`
	Country    string    `json:"country"`
	State      string    `json:"state"`
	WebsiteID  int       `json:"websiteId"` // Foreign key to Website model
}

type VisitUpdateResponse struct {
	VisitInsert
	ID        int `json:"id"`
	WebsiteID int `json:"websiteId"`
}
