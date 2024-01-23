package utils

import (
	"regexp"
)

// ExtractIDFromURL extracts the numeric ID from the given URL path using the provided pattern.
func ExtractIDFromURL(path, pattern string) string {
	re := regexp.MustCompile(pattern)
	matches := re.FindStringSubmatch(path)

	if len(matches) > 1 {
		return matches[1]
	}

	return ""
}
