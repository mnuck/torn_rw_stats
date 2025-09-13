package processing

import (
	"testing"
)

func TestLocationServiceParseLocation(t *testing.T) {
	ls := NewLocationService()

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
			result := ls.ParseLocation(tt.description)
			if result != tt.expected {
				t.Errorf("ParseLocation(%q) = %q, expected %q", tt.description, result, tt.expected)
			}
		})
	}
}

func TestLocationServiceGetTravelDestinationForCalculation(t *testing.T) {
	ls := NewLocationService()

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
			result := ls.GetTravelDestinationForCalculation(tt.description, tt.parsedLocation)
			if result != tt.expected {
				t.Errorf("GetTravelDestinationForCalculation(%q, %q) = %q, expected %q",
					tt.description, tt.parsedLocation, result, tt.expected)
			}
		})
	}
}

// Test edge cases and error conditions
func TestLocationServiceEdgeCases(t *testing.T) {
	ls := NewLocationService()

	// Test hospital patterns with different articles
	result := ls.ParseLocation("In an Emirati hospital for 1hr")
	if result != "UAE" {
		t.Errorf("Expected 'UAE' for Emirati hospital, got %q", result)
	}

	// Test case sensitivity
	result = ls.ParseLocation("TRAVELING TO CHINA")
	if result != "China" {
		t.Errorf("Expected 'China' for uppercase travel, got %q", result)
	}

	// Test partial matches - this should fall back to original description
	result = ls.ParseLocation("Chilling in Mexico City")
	if result != "Chilling in Mexico City" {
		t.Errorf("Expected 'Chilling in Mexico City' for unmatched pattern, got %q", result)
	}
}

// Benchmark the parsing function to ensure it's efficient
func BenchmarkLocationServiceParseLocation(b *testing.B) {
	ls := NewLocationService()
	descriptions := []string{
		"In a British hospital for 2hrs 15mins",
		"Traveling to Mexico",
		"In Canada",
		"Returning to Torn from Switzerland",
		"Okay",
		"Some random status",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		desc := descriptions[i%len(descriptions)]
		ls.ParseLocation(desc)
	}
}