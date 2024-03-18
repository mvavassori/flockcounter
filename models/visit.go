package models

import "time"

type Visit struct {
	ID              int       `json:"id"`
	WebsiteID       int       `json:"websiteId"` // Foreign key to Website model
	WebsiteDomain   string    `json:"websiteDomain"`
	Timestamp       time.Time `json:"timestamp"`
	Referrer        string    `json:"referrer"`
	URL             string    `json:"url"`
	Pathname        string    `json:"pathname"`
	DeviceType      string    `json:"deviceType"`
	OS              string    `json:"os"`
	Browser         string    `json:"browser"`
	Language        string    `json:"language"`
	Country         string    `json:"country"`
	State           string    `json:"state"`
	IsUnique        bool      `json:"isUnique"`
	TimeSpentOnPage int       `json:"timeSpentOnPage"`
}

type VisitReceiver struct {
	Timestamp       time.Time `json:"timestamp"`
	Referrer        string    `json:"referrer"`
	URL             string    `json:"url"`
	Pathname        string    `json:"pathname"`
	UserAgent       string    `json:"userAgent"`
	Language        string    `json:"language"`
	Country         string    `json:"country"`
	State           string    `json:"state"`
	IsUnique        bool      `json:"isUnique"`
	TimeSpentOnPage int       `json:"timeSpentOnPage"`
}

type VisitInsert struct {
	WebsiteID       int       `json:"websiteId"` // Foreign key to Website model
	WebsiteDomain   string    `json:"websiteDomain"`
	Timestamp       time.Time `json:"timestamp"`
	Referrer        string    `json:"referrer"`
	URL             string    `json:"url"`
	Pathname        string    `json:"pathname"`
	DeviceType      string    `json:"deviceType"`
	OS              string    `json:"os"`
	Browser         string    `json:"browser"`
	Language        string    `json:"language"`
	Country         string    `json:"country"`
	State           string    `json:"state"`
	IsUnique        bool      `json:"isUnique"`
	TimeSpentOnPage int       `json:"timeSpentOnPage"`
}

type VisitUpdateResponse struct {
	VisitInsert
	ID            int    `json:"id"`
	WebsiteID     int    `json:"websiteId"`
	WebsiteDomain string `json:"websiteDomain"`
}
