package utils

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"time"
)

type dailySalt struct {
	salt []byte
	date time.Time
}

// Stored in memory
var dailySaltCache = make(map[string]dailySalt)

// getDailySalt generates a random 16-byte salt
func getDailySalt() ([]byte, error) {
	// Generate a random 16-byte salt
	salt := make([]byte, 16)
	_, err := rand.Read(salt)
	if err != nil {
		return nil, err
	}
	return salt, nil
}

// GenerateDailySalt generates a unique salt for the current day if it hasn't been generated yet.
func GenerateDailySalt() ([]byte, error) {
	now := time.Now()
	dateString := now.Format("2006-01-02")

	if salt, ok := dailySaltCache[dateString]; ok {
		return salt.salt, nil
	}

	salt, err := getDailySalt()
	if err != nil {
		return nil, err
	}

	dailySaltCache[dateString] = dailySalt{salt: salt, date: now}
	return salt, nil
}

func GenerateUniqueIdentifier(dailySalt []byte, websiteDomain, ipAddress, userAgent string) (string, error) {
	// Combine daily salt, website domain, IP address, and user agent
	combinedString := string(dailySalt) + websiteDomain + ipAddress + userAgent

	// Hash the combined string using SHA-256
	hasher := sha256.New()
	hasher.Write([]byte(combinedString))
	hashedBytes := hasher.Sum(nil)

	// Convert hashed bytes to a hexadecimal string
	hashedString := hex.EncodeToString(hashedBytes)

	return hashedString, nil
}