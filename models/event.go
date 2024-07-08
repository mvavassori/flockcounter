package models

import "time"

type Event struct {
	ID            int64     `json:"id"`
	WebsiteID     int64     `json:"websiteId"` // Foreign key to Website model
	WebsiteDomain string    `json:"websiteDomain"`
	Type          string    `json:"type"`
	Name          string    `json:"name"`
	Timestamp     time.Time `json:"timestamp"`
	Referrer      string    `json:"referrer"`
	URL           string    `json:"url"`
	Pathname      string    `json:"pathname"`
	DeviceType    string    `json:"deviceType"`
	OS            string    `json:"os"`
	Browser       string    `json:"browser"`
	Language      string    `json:"language"`
	Country       string    `json:"country"`
	Region        string    `json:"region"`
	City          string    `json:"city"`
	IsUnique      bool      `json:"isUnique"`
}

type EventReceiver struct {
	Type      string    `json:"type"`
	Name      string    `json:"name"`
	Timestamp time.Time `json:"timestamp"`
	Referrer  string    `json:"referrer"`
	URL       string    `json:"url"`
	Pathname  string    `json:"pathname"`
	UserAgent string    `json:"userAgent"`
	Language  string    `json:"language"`
}

type EventInsert struct {
	WebsiteID     int64     `json:"websiteId"`
	WebsiteDomain string    `json:"websiteDomain"`
	Type          string    `json:"type"`
	Name          string    `json:"name"`
	Timestamp     time.Time `json:"timestamp"`
	Referrer      string    `json:"referrer"`
	URL           string    `json:"url"`
	Pathname      string    `json:"pathname"`
	DeviceType    string    `json:"deviceType"`
	OS            string    `json:"os"`
	Browser       string    `json:"browser"`
	Language      string    `json:"language"`
	Country       string    `json:"country"`
	Region        string    `json:"region"`
	City          string    `json:"city"`
	IsUnique      bool      `json:"isUnique"`
}

type EventUpdateResponse struct {
	EventInsert
	ID            int64     `json:"id"`
	WebsiteID     int64     `json:"websiteId"`
	WebsiteDomain string    `json:"websiteDomain"`
	Timestamp     time.Time `json:"timestamp"`
}
