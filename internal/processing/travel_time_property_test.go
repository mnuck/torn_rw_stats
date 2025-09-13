package processing

import (
	"fmt"
	"testing"
	"time"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
)

// TestTravelTimeServiceProperties uses property-based testing for travel time logic
func TestTravelTimeServiceProperties(t *testing.T) {
	service := NewTravelTimeService()

	properties := gopter.NewProperties(nil)

	// Property: Travel time should always be positive
	properties.Property("travel time always positive", prop.ForAll(
		func(destination, travelType string) bool {
			duration := service.GetTravelTime(destination, travelType)
			return duration > 0
		},
		gen.OneConstOf("Mexico", "United Kingdom", "Switzerland", "Hawaii", "Canada", "UnknownPlace"),
		gen.OneConstOf("regular", "airstrip"),
	))

	// Property: Airstrip travel should be faster than regular travel for known destinations
	properties.Property("airstrip faster than regular", prop.ForAll(
		func(destination string) bool {
			regularTime := service.GetTravelTime(destination, "regular")
			airstripTime := service.GetTravelTime(destination, "airstrip")

			// For unknown destinations, both might return default, so skip those
			if regularTime == 30*time.Minute && airstripTime == 30*time.Minute {
				return true // Skip unknown destinations
			}

			return airstripTime <= regularTime
		},
		gen.OneConstOf("Mexico", "United Kingdom", "Switzerland", "Hawaii", "Canada"),
	))

	// Property: FormatTravelTime should handle all durations correctly
	properties.Property("format travel time valid", prop.ForAll(
		func(minutes int) bool {
			duration := time.Duration(minutes) * time.Minute
			formatted := service.FormatTravelTime(duration)

			// Should always be in HH:MM:SS format
			if len(formatted) != 8 {
				return false
			}

			// Should have colons at positions 2 and 5
			if formatted[2] != ':' || formatted[5] != ':' {
				return false
			}

			// Negative durations should format as 00:00:00
			if minutes < 0 {
				return formatted == "00:00:00"
			}

			return true
		},
		gen.IntRange(-60, 1000), // Test negative, zero, and positive durations
	))

	// Property: FormatTravelTime should be consistent with duration components
	properties.Property("format travel time components consistent", prop.ForAll(
		func(hours, minutes int) bool {
			if hours < 0 || minutes < 0 || hours > 23 || minutes > 59 {
				return true // Skip invalid combinations
			}

			duration := time.Duration(hours)*time.Hour + time.Duration(minutes)*time.Minute
			formatted := service.FormatTravelTime(duration)

			expectedHours := hours
			expectedMinutes := minutes

			// Parse the formatted string
			var parsedHours, parsedMinutes, parsedSeconds int
			n, err := fmt.Sscanf(formatted, "%02d:%02d:%02d", &parsedHours, &parsedMinutes, &parsedSeconds)
			if err != nil || n != 3 {
				return false
			}

			return parsedHours == expectedHours && parsedMinutes == expectedMinutes && parsedSeconds == 0
		},
		gen.IntRange(0, 23),
		gen.IntRange(0, 59),
	))

	// Property: Known destinations should return consistent times
	properties.Property("known destinations consistent", prop.ForAll(
		func(destination string) bool {
			// Call multiple times and ensure consistency
			time1 := service.GetTravelTime(destination, "regular")
			time2 := service.GetTravelTime(destination, "regular")
			time3 := service.GetTravelTime(destination, "airstrip")
			time4 := service.GetTravelTime(destination, "airstrip")

			return time1 == time2 && time3 == time4
		},
		gen.OneConstOf("Mexico", "United Kingdom", "Switzerland", "Hawaii", "Canada"),
	))

	// Property: Zero and negative durations should format to zero
	properties.Property("zero and negative durations format to zero", prop.ForAll(
		func(negativeMinutes int) bool {
			if negativeMinutes > 0 {
				return true // Skip positive values
			}
			negativeDuration := time.Duration(negativeMinutes) * time.Minute
			formatted := service.FormatTravelTime(negativeDuration)
			return formatted == "00:00:00"
		},
		gen.IntRange(-1000, 0),
	))

	// Property: Large durations should still format correctly (no overflow)
	properties.Property("large durations format correctly", prop.ForAll(
		func(days int) bool {
			if days < 0 || days > 100 {
				return true // Skip unreasonable values
			}

			duration := time.Duration(days*24) * time.Hour
			formatted := service.FormatTravelTime(duration)

			// Should still be properly formatted, even for large values
			if len(formatted) < 8 {
				return false
			}

			// Should have colons in correct positions
			colonCount := 0
			for _, char := range formatted {
				if char == ':' {
					colonCount++
				}
			}

			return colonCount == 2
		},
		gen.IntRange(0, 10), // Test up to 10 days
	))

	properties.TestingRun(t)
}