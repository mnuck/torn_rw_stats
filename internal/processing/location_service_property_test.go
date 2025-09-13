package processing

import (
	"strings"
	"testing"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
)

// TestLocationServiceProperties uses property-based testing for location parsing logic
func TestLocationServiceProperties(t *testing.T) {
	service := NewLocationService()

	properties := gopter.NewProperties(nil)

	// Property: ParseLocation should be idempotent
	properties.Property("parse location idempotent", prop.ForAll(
		func(description string) bool {
			location1 := service.ParseLocation(description)
			location2 := service.ParseLocation(location1)
			return location1 == location2
		},
		gen.AlphaString(),
	))

	// Property: "Okay" status should always return "Torn"
	properties.Property("okay status returns torn", prop.ForAll(
		func() bool {
			location := service.ParseLocation("Okay")
			return location == "Torn"
		},
	))

	// Property: Traveling patterns should extract destination correctly
	properties.Property("traveling patterns extract destination", prop.ForAll(
		func(destination string) bool {
			patterns := []string{
				"Traveling to " + destination,
				"TRAVELING TO " + strings.ToUpper(destination),
				"traveling to " + strings.ToLower(destination),
			}

			for _, pattern := range patterns {
				location := service.ParseLocation(pattern)
				if location != destination {
					return false
				}
			}
			return true
		},
		gen.OneConstOf("Mexico", "United Kingdom", "Switzerland", "Hawaii", "Canada"),
	))

	// Property: "In [Location]" patterns should extract location correctly
	properties.Property("in location patterns extract location", prop.ForAll(
		func(location string) bool {
			patterns := []string{
				"In " + location,
				"IN " + strings.ToUpper(location),
				"in " + strings.ToLower(location),
			}

			for _, pattern := range patterns {
				result := service.ParseLocation(pattern)
				if result != location {
					return false
				}
			}
			return true
		},
		gen.OneConstOf("Mexico", "United Kingdom", "Switzerland", "Hawaii", "Canada"),
	))

	// Property: Return travel should always return "Torn"
	properties.Property("return travel returns torn", prop.ForAll(
		func(origin string) bool {
			patterns := []string{
				"Returning to Torn from " + origin,
				"RETURNING TO TORN FROM " + strings.ToUpper(origin),
				"returning to torn from " + strings.ToLower(origin),
			}

			for _, pattern := range patterns {
				location := service.ParseLocation(pattern)
				if location != "Torn" {
					return false
				}
			}
			return true
		},
		gen.OneConstOf("Mexico", "United Kingdom", "Switzerland", "Hawaii", "Canada"),
	))

	// Property: Hospital patterns should map to correct countries
	properties.Property("hospital patterns map to countries", prop.ForAll(
		func(pair [2]string) bool {
			adjective := pair[0]
			expectedCountry := pair[1]
			patterns := []string{
				"In a " + adjective + " hospital for 30mins",
				"IN A " + strings.ToUpper(adjective) + " HOSPITAL FOR 2HRS",
				"in a " + strings.ToLower(adjective) + " hospital for 45mins",
			}

			for _, pattern := range patterns {
				location := service.ParseLocation(pattern)
				if location != expectedCountry {
					return false
				}
			}
			return true
		},
		gen.OneConstOf(
			[2]string{"British", "United Kingdom"},
			[2]string{"Mexican", "Mexico"},
			[2]string{"Swiss", "Switzerland"},
		),
	))

	// Property: Descriptions containing "torn" should return "Torn"
	properties.Property("descriptions with torn return torn", prop.ForAll(
		func(prefix, suffix string) bool {
			// Create descriptions that contain "torn" (case insensitive)
			patterns := []string{
				prefix + "torn" + suffix,
				prefix + "Torn" + suffix,
				prefix + "TORN" + suffix,
			}

			for _, pattern := range patterns {
				location := service.ParseLocation(pattern)
				if location != "Torn" {
					return false
				}
			}
			return true
		},
		gen.OneConstOf("In ", "At ", "Chilling in ", ""),
		gen.OneConstOf(" city", " location", ""),
	))

	// Property: GetTravelDestinationForCalculation should handle return journeys
	properties.Property("travel destination calculation for returns", prop.ForAll(
		func(origin string) bool {
			description := "Returning to Torn from " + origin
			parsedLocation := "Torn" // Should be parsed as Torn

			destination := service.GetTravelDestinationForCalculation(description, parsedLocation)
			return destination == origin
		},
		gen.OneConstOf("Mexico", "United Kingdom", "Switzerland", "Hawaii", "Canada"),
	))

	// Property: GetTravelDestinationForCalculation should return parsed location for non-returns
	properties.Property("travel destination calculation for non-returns", prop.ForAll(
		func(destination string) bool {
			description := "Traveling to " + destination
			parsedLocation := destination

			result := service.GetTravelDestinationForCalculation(description, parsedLocation)
			return result == destination
		},
		gen.OneConstOf("Mexico", "United Kingdom", "Switzerland", "Hawaii", "Canada"),
	))

	// Property: Empty descriptions should return empty strings
	properties.Property("empty descriptions return empty", prop.ForAll(
		func() bool {
			location := service.ParseLocation("")
			return location == ""
		},
	))

	// Property: Unknown descriptions should return themselves
	properties.Property("unknown descriptions return themselves", prop.ForAll(
		func(description string) bool {
			// Only test strings that don't match known patterns
			lowerDesc := strings.ToLower(description)
			if strings.Contains(lowerDesc, "traveling") ||
			   strings.Contains(lowerDesc, "hospital") ||
			   strings.Contains(lowerDesc, "torn") ||
			   strings.Contains(lowerDesc, "okay") ||
			   strings.HasPrefix(lowerDesc, "in ") ||
			   strings.Contains(lowerDesc, "returning") {
				return true // Skip known patterns
			}

			location := service.ParseLocation(description)
			return location == description
		},
		gen.OneConstOf("At the beach", "Doing crimes", "Random activity", "Custom status"),
	))

	properties.TestingRun(t)
}