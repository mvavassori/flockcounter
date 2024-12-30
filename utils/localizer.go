package utils

import (
	"net"
	"net/http"
	"strings"

	"github.com/oschwald/geoip2-golang"
)

// GetIPAddress tries different methods to get the real IP address
func GetIPAddress(r *http.Request) string {
	// Check X-Forwarded-For header
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		ips := strings.Split(xff, ",")
		// Take the first non-private IP from X-Forwarded-For
		for _, ip := range ips {
			trimmedIP := strings.TrimSpace(ip)
			parsedIP := net.ParseIP(trimmedIP)
			if parsedIP != nil && !isPrivateIP(parsedIP) {
				return trimmedIP
			}
		}
	}

	// Check X-Real-IP header
	if xRealIP := r.Header.Get("X-Real-IP"); xRealIP != "" {
		if parsedIP := net.ParseIP(xRealIP); parsedIP != nil {
			return xRealIP
		}
	}

	// Fall back to RemoteAddr
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		// Try RemoteAddr directly if SplitHostPort fails
		return r.RemoteAddr
	}
	return ip
}

// isPrivateIP checks if an IP address is private
func isPrivateIP(ip net.IP) bool {
	privateNetworks := []struct {
		network *net.IPNet
	}{
		{&net.IPNet{IP: net.ParseIP("10.0.0.0"), Mask: net.CIDRMask(8, 32)}},
		{&net.IPNet{IP: net.ParseIP("172.16.0.0"), Mask: net.CIDRMask(12, 32)}},
		{&net.IPNet{IP: net.ParseIP("192.168.0.0"), Mask: net.CIDRMask(16, 32)}},
	}

	for _, network := range privateNetworks {
		if network.network.Contains(ip) {
			return true
		}
	}
	return false
}

// Location holds the parsed location information
type Location struct {
	Country string
	Region  string
	City    string
}

// GetLocationInfo extracts location information from the GeoIP record
func GetLocationInfo(record *geoip2.City) Location {
	location := Location{
		Country: "Unknown",
		Region:  "Unknown",
		City:    "Unknown",
	}

	if record.Country.Names != nil {
		if countryName, ok := record.Country.Names["en"]; ok {
			location.Country = countryName
		}
	}

	if len(record.Subdivisions) > 0 && record.Subdivisions[0].Names != nil {
		if regionName, ok := record.Subdivisions[0].Names["en"]; ok {
			location.Region = regionName
		}
	}

	if record.City.Names != nil {
		if cityName, ok := record.City.Names["en"]; ok {
			location.City = cityName
		}
	}

	return location
}
