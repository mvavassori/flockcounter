package utils

import "github.com/mileusna/useragent"

func GetDeviceType(ua *useragent.UserAgent) string {
	if ua.Mobile {
		return "Mobile"
	} else if ua.Tablet {
		return "Tablet"
	} else if ua.Desktop {
		return "Desktop"
	} else if ua.Bot {
		return "Bot"
	} else {
		return "Unknown"
	}
}
