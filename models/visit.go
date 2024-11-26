package models

import (
	"database/sql"
	"encoding/json"
	"time"

	"github.com/mvavassori/bare-analytics/utils"
)

type Visit struct {
	ID              int            `json:"id"`
	WebsiteID       int            `json:"websiteId"` // Foreign key to Website model
	WebsiteDomain   string         `json:"websiteDomain"`
	Timestamp       time.Time      `json:"timestamp"`
	Referrer        string         `json:"referrer"`
	URL             string         `json:"url"`
	Pathname        string         `json:"pathname"`
	DeviceType      string         `json:"deviceType"`
	OS              string         `json:"os"`
	Browser         string         `json:"browser"`
	Language        string         `json:"language"`
	Country         string         `json:"country"`
	Region          string         `json:"region"`
	City            string         `json:"city"`
	TimeSpentOnPage int            `json:"timeSpentOnPage"`
	IsUnique        bool           `json:"isUnique"`
	UTMSource       sql.NullString `json:"utmSource"`
	UTMMedium       sql.NullString `json:"utmMedium"`
	UTMCampaign     sql.NullString `json:"utmCampaign"`
	UTMTerm         sql.NullString `json:"utmTerm"`
	UTMContent      sql.NullString `json:"utmContent"`
}

type VisitReceiver struct {
	Timestamp       time.Time `json:"timestamp"`
	Referrer        string    `json:"referrer"`
	URL             string    `json:"url"`
	Pathname        string    `json:"pathname"`
	UserAgent       string    `json:"userAgent"`
	Language        string    `json:"language"`
	TimeSpentOnPage int       `json:"timeSpentOnPage"`
}

type VisitInsert struct {
	WebsiteID       int            `json:"websiteId"`
	WebsiteDomain   string         `json:"websiteDomain"`
	Timestamp       time.Time      `json:"timestamp"`
	Referrer        string         `json:"referrer"`
	URL             string         `json:"url"`
	Pathname        string         `json:"pathname"`
	DeviceType      string         `json:"deviceType"`
	OS              string         `json:"os"`
	Browser         string         `json:"browser"`
	Language        string         `json:"language"`
	Country         string         `json:"country"`
	Region          string         `json:"region"`
	City            string         `json:"city"`
	TimeSpentOnPage int            `json:"timeSpentOnPage"`
	IsUnique        bool           `json:"isUnique"`
	UTMSource       sql.NullString `json:"utmSource"`
	UTMMedium       sql.NullString `json:"utmMedium"`
	UTMCampaign     sql.NullString `json:"utmCampaign"`
	UTMTerm         sql.NullString `json:"utmTerm"`
	UTMContent      sql.NullString `json:"utmContent"`
}

// MarshalJSON customizes the JSON encoding for the Visit struct.
// This ensures that NullString fields are marshaled as `null` when empty.
func (v *Visit) MarshalJSON() ([]byte, error) {
	type Alias Visit // Create an alias to avoid infinite recursion

	return json.Marshal(&struct {
		*Alias
		UTMSource   interface{} `json:"utmSource"`
		UTMMedium   interface{} `json:"utmMedium"`
		UTMCampaign interface{} `json:"utmCampaign"`
		UTMTerm     interface{} `json:"utmTerm"`
		UTMContent  interface{} `json:"utmContent"`
	}{
		Alias:       (*Alias)(v),
		UTMSource:   utils.NullableStringToJSON(v.UTMSource),
		UTMMedium:   utils.NullableStringToJSON(v.UTMMedium),
		UTMCampaign: utils.NullableStringToJSON(v.UTMCampaign),
		UTMTerm:     utils.NullableStringToJSON(v.UTMTerm),
		UTMContent:  utils.NullableStringToJSON(v.UTMContent),
	})
}
