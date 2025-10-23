package status

import (
	"fmt"
	"time"
)

// CalculateCountdown calculates countdown string from a future timestamp
// Returns empty string if timestamp is zero, "0:00:00" if time has passed
func CalculateCountdown(statusUntil time.Time, currentTime time.Time) string {
	if statusUntil.IsZero() {
		return ""
	}

	duration := statusUntil.Sub(currentTime)
	if duration <= 0 {
		return "0:00:00"
	}

	// Format as H:MM:SS
	hours := int(duration.Hours())
	minutes := int(duration.Minutes()) % 60
	seconds := int(duration.Seconds()) % 60

	return fmt.Sprintf("%d:%02d:%02d", hours, minutes, seconds)
}

// CalculateTravelTimes calculates departure and arrival times for traveling members
func CalculateTravelTimes(
	isTraveling bool,
	existingDeparture string,
	existingArrival string,
	departureTime time.Time,
	statusUntil time.Time,
) (departure string, arrival string) {
	if !isTraveling {
		return "", ""
	}

	// Use existing departure if available
	if existingDeparture != "" {
		departure = existingDeparture
	} else if !departureTime.IsZero() {
		departure = departureTime.UTC().Format("2006-01-02 15:04:05")
	}

	// Use existing arrival if available
	if existingArrival != "" {
		arrival = existingArrival
	} else if !statusUntil.IsZero() {
		arrival = statusUntil.UTC().Format("2006-01-02 15:04:05")
	}

	return departure, arrival
}

// FormatTimestamp formats a time.Time to the standard format used in sheets
func FormatTimestamp(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format("2006-01-02 15:04:05")
}
