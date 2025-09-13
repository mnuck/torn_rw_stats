package processing

import (
	"testing"
	"time"

	"torn_rw_stats/internal/app"
)

// Test the parseLocation function - this is critical business logic that needs comprehensive testing
func TestParseLocation(t *testing.T) {
	config := &app.Config{UpdateInterval: 5 * time.Minute}
	wp := &WarProcessor{
		tornClient:        nil, // Not needed for parseLocation tests
		sheetsClient:      nil, // Not needed for parseLocation tests
		config:            config,
		ourFactionID:      12345,
		locationService:   NewLocationService(),
		travelTimeService: NewTravelTimeService(),
	}

	tests := []struct {
		name        string
		description string
		expected    string
	}{
		// Hospital location mappings
		{
			name:        "British hospital",
			description: "In a British hospital for 2hrs 15mins",
			expected:    "United Kingdom",
		},
		{
			name:        "Mexican hospital with 'an'",
			description: "In a Mexican hospital for 1hrs 30mins",
			expected:    "Mexico",
		},
		{
			name:        "Swiss hospital",
			description: "In a Swiss hospital for 45mins",
			expected:    "Switzerland",
		},
		// Direct location patterns
		{
			name:        "Traveling to Mexico",
			description: "Traveling to Mexico",
			expected:    "Mexico",
		},
		{
			name:        "Traveling to United Kingdom",
			description: "Traveling to United Kingdom",
			expected:    "United Kingdom",
		},
		{
			name:        "In Canada",
			description: "In Canada",
			expected:    "Canada",
		},
		{
			name:        "In Hawaii",
			description: "In Hawaii",
			expected:    "Hawaii",
		},
		// Return travel
		{
			name:        "Returning to Torn from Mexico",
			description: "Returning to Torn from Mexico",
			expected:    "Torn",
		},
		{
			name:        "Returning to Torn from Switzerland",
			description: "Returning to Torn from Switzerland",
			expected:    "Torn",
		},
		// Default cases
		{
			name:        "Okay status",
			description: "Okay",
			expected:    "Torn",
		},
		{
			name:        "Contains torn",
			description: "Chilling in Torn",
			expected:    "Torn",
		},
		// Hospital without location defaults to Torn
		{
			name:        "Generic hospital",
			description: "In hospital for 2hrs",
			expected:    "Torn",
		},
		// Edge cases
		{
			name:        "Empty description",
			description: "",
			expected:    "",
		},
		{
			name:        "Unknown location",
			description: "Some random status",
			expected:    "Some random status",
		},
		// Case insensitive
		{
			name:        "Mixed case traveling",
			description: "TRAVELING TO mexico",
			expected:    "Mexico",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := wp.locationService.ParseLocation(tt.description)
			if result != tt.expected {
				t.Errorf("parseLocation(%q) = %q, expected %q", tt.description, result, tt.expected)
			}
		})
	}
}

// Test hospital countdown parsing
func TestParseHospitalCountdown(t *testing.T) {
	config := &app.Config{UpdateInterval: 5 * time.Minute}
	wp := &WarProcessor{
		tornClient:        nil,
		sheetsClient:      nil,
		config:            config,
		ourFactionID:      12345,
		locationService:   NewLocationService(),
		travelTimeService: NewTravelTimeService(),
	}

	tests := []struct {
		name        string
		description string
		expected    string
	}{
		{
			name:        "Hours and minutes",
			description: "In hospital for 2hrs 15mins",
			expected:    "02:15:00",
		},
		{
			name:        "Only hours",
			description: "In hospital for 3hrs",
			expected:    "03:00:00",
		},
		{
			name:        "Only minutes",
			description: "In hospital for 45mins",
			expected:    "00:45:00",
		},
		{
			name:        "Alternative format",
			description: "In hospital for 1hr 30mins",
			expected:    "01:30:00",
		},
		{
			name:        "No countdown",
			description: "In hospital",
			expected:    "",
		},
		{
			name:        "Empty description",
			description: "",
			expected:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := wp.parseHospitalCountdown(tt.description)
			if result != tt.expected {
				t.Errorf("parseHospitalCountdown(%q) = %q, expected %q", tt.description, result, tt.expected)
			}
		})
	}
}

// Test travel time calculations
func TestGetTravelTime(t *testing.T) {
	tts := NewTravelTimeService()

	tests := []struct {
		name        string
		destination string
		travelType  string
		expected    time.Duration
	}{
		{
			name:        "Mexico regular",
			destination: "Mexico",
			travelType:  "regular",
			expected:    26 * time.Minute,
		},
		{
			name:        "Mexico airstrip",
			destination: "Mexico",
			travelType:  "airstrip",
			expected:    18 * time.Minute,
		},
		{
			name:        "United Kingdom regular",
			destination: "United Kingdom",
			travelType:  "regular",
			expected:    159 * time.Minute,
		},
		{
			name:        "Unknown destination",
			destination: "Unknown",
			travelType:  "regular",
			expected:    30 * time.Minute, // Default fallback
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tts.GetTravelTime(tt.destination, tt.travelType)
			if result != tt.expected {
				t.Errorf("GetTravelTime(%q, %q) = %v, expected %v", tt.destination, tt.travelType, result, tt.expected)
			}
		})
	}
}

// Test travel destination calculation logic
func TestGetTravelDestinationForCalculation(t *testing.T) {
	config := &app.Config{UpdateInterval: 5 * time.Minute}
	wp := &WarProcessor{
		tornClient:        nil,
		sheetsClient:      nil,
		config:            config,
		ourFactionID:      12345,
		locationService:   NewLocationService(),
		travelTimeService: NewTravelTimeService(),
	}

	tests := []struct {
		name           string
		description    string
		parsedLocation string
		expected       string
	}{
		{
			name:           "Normal travel to foreign country",
			description:    "Traveling to Mexico",
			parsedLocation: "Mexico",
			expected:       "Mexico",
		},
		{
			name:           "Return from Mexico",
			description:    "Returning to Torn from Mexico",
			parsedLocation: "Torn",
			expected:       "Mexico", // Should extract origin for calculation
		},
		{
			name:           "Return from United Kingdom",
			description:    "Returning to Torn from United Kingdom",
			parsedLocation: "Torn",
			expected:       "United Kingdom",
		},
		{
			name:           "Just in Torn",
			description:    "Okay",
			parsedLocation: "Torn",
			expected:       "Torn",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := wp.locationService.GetTravelDestinationForCalculation(tt.description, tt.parsedLocation)
			if result != tt.expected {
				t.Errorf("getTravelDestinationForCalculation(%q, %q) = %q, expected %q",
					tt.description, tt.parsedLocation, result, tt.expected)
			}
		})
	}
}

// Test format travel time function
func TestFormatTravelTime(t *testing.T) {
	tts := NewTravelTimeService()

	tests := []struct {
		name     string
		duration time.Duration
		expected string
	}{
		{
			name:     "Zero duration",
			duration: 0,
			expected: "00:00:00",
		},
		{
			name:     "Negative duration",
			duration: -5 * time.Minute,
			expected: "00:00:00",
		},
		{
			name:     "Minutes only",
			duration: 45 * time.Minute,
			expected: "00:45:00",
		},
		{
			name:     "Hours and minutes",
			duration: 2*time.Hour + 30*time.Minute,
			expected: "02:30:00",
		},
		{
			name:     "Hours, minutes, seconds",
			duration: 3*time.Hour + 15*time.Minute + 42*time.Second,
			expected: "03:15:42",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tts.FormatTravelTime(tt.duration)
			if result != tt.expected {
				t.Errorf("FormatTravelTime(%v) = %q, expected %q", tt.duration, result, tt.expected)
			}
		})
	}
}