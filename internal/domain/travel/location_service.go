package travel

import (
	"strings"
)

// LocationService handles location parsing and standardization
type LocationService struct {
	hospitalMappings map[string]string
	locations        []string
}

// NewLocationService creates a new location service with predefined mappings
func NewLocationService() *LocationService {
	return &LocationService{
		hospitalMappings: map[string]string{
			"british":       "United Kingdom",
			"caymanian":     "Cayman Islands",
			"chinese":       "China",
			"mexican":       "Mexico",
			"swiss":         "Switzerland",
			"japanese":      "Japan",
			"canadian":      "Canada",
			"hawaiian":      "Hawaii",
			"emirati":       "UAE",
			"south african": "South Africa",
			"argentinian":   "Argentina",
		},
		locations: []string{
			"Mexico", "Cayman Islands", "Canada", "Hawaii", "United Kingdom",
			"Argentina", "Switzerland", "Japan", "China", "UAE",
			"South Africa",
		},
	}
}

// ParseLocation extracts standardized location from status description
func (ls *LocationService) ParseLocation(description string) string {
	if description == "" {
		return ""
	}

	descLower := strings.ToLower(description)

	// Check for hospital patterns first (handle both "a" and "an")
	if location := ls.parseHospitalLocation(descLower); location != "" {
		return location
	}

	// Check "Traveling to X" pattern
	if location := ls.parseTravelingToLocation(descLower); location != "" {
		return location
	}

	// Check "In X" pattern
	if location := ls.parseInLocation(descLower); location != "" {
		return location
	}

	// Check "Returning to Torn from X" pattern
	if strings.Contains(descLower, "returning to torn from") {
		return "Torn"
	}

	// Default cases
	if strings.Contains(descLower, "okay") || strings.Contains(descLower, "torn") {
		return "Torn"
	}

	// Hospital without location qualifier defaults to Torn
	if location := ls.parseGenericHospital(descLower); location != "" {
		return location
	}

	// Return original description if no pattern matches
	return description
}

// parseHospitalLocation handles hospital location patterns
func (ls *LocationService) parseHospitalLocation(descLower string) string {
	for adjective, location := range ls.hospitalMappings {
		if strings.Contains(descLower, "in a "+adjective+" hospital") ||
			strings.Contains(descLower, "in an "+adjective+" hospital") {
			return location
		}
	}
	return ""
}

// parseTravelingToLocation handles "Traveling to X" patterns
func (ls *LocationService) parseTravelingToLocation(descLower string) string {
	if strings.HasPrefix(descLower, "traveling to ") {
		for _, location := range ls.locations {
			if strings.Contains(descLower, strings.ToLower(location)) {
				return location
			}
		}
	}
	return ""
}

// parseInLocation handles "In X" patterns
func (ls *LocationService) parseInLocation(descLower string) string {
	if strings.HasPrefix(descLower, "in ") && !strings.Contains(descLower, "hospital") {
		for _, location := range ls.locations {
			if strings.Contains(descLower, strings.ToLower(location)) {
				return location
			}
		}
	}
	return ""
}

// parseGenericHospital handles hospital without specific location
func (ls *LocationService) parseGenericHospital(descLower string) string {
	if strings.Contains(descLower, "in hospital for") {
		// Check if it's NOT one of the specific country hospitals
		isCountryHospital := false
		for adjective := range ls.hospitalMappings {
			if strings.Contains(descLower, "in a "+adjective+" hospital") {
				isCountryHospital = true
				break
			}
		}
		if !isCountryHospital {
			return "Torn"
		}
	}
	return ""
}

// GetTravelDestinationForCalculation returns the destination to use for travel time calculations
// For "Returning to Torn from X", returns X (the origin country)
// For other travel, returns the parsed location
func (ls *LocationService) GetTravelDestinationForCalculation(description, parsedLocation string) string {
	if parsedLocation != "Torn" {
		return parsedLocation // Normal travel to foreign country
	}

	// For "Returning to Torn from X" cases, extract X for travel time calculation
	descLower := strings.ToLower(description)
	if strings.Contains(descLower, "returning to torn from") {
		// Extract the country name after "from "
		for _, location := range ls.locations {
			if strings.Contains(descLower, strings.ToLower(location)) {
				return location
			}
		}
	}

	return parsedLocation
}
