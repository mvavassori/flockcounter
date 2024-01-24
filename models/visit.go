package models

import "time"

type Visit struct {
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
}
