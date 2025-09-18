package travel

import (
	"context"
	"testing"
	"time"
)

func TestTravelTimeServiceGetTravelTime(t *testing.T) {
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
			name:        "United Kingdom airstrip",
			destination: "United Kingdom",
			travelType:  "airstrip",
			expected:    111 * time.Minute,
		},
		{
			name:        "Mexico business class",
			destination: "Mexico",
			travelType:  "business",
			expected:    8 * time.Minute,
		},
		{
			name:        "Switzerland business class",
			destination: "Switzerland",
			travelType:  "business",
			expected:    53 * time.Minute,
		},
		{
			name:        "Unknown destination",
			destination: "Unknown",
			travelType:  "regular",
			expected:    30 * time.Minute, // Default fallback
		},
		{
			name:        "Switzerland airstrip",
			destination: "Switzerland",
			travelType:  "airstrip",
			expected:    123 * time.Minute,
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

func TestTravelTimeServiceFormatTravelTime(t *testing.T) {
	tts := NewTravelTimeService()

	tests := []struct {
		name     string
		duration time.Duration
		expected string
	}{
		{
			name:     "Zero duration",
			duration: 0,
			expected: "'00:00:00",
		},
		{
			name:     "Negative duration",
			duration: -5 * time.Minute,
			expected: "'00:00:00",
		},
		{
			name:     "Minutes only",
			duration: 45 * time.Minute,
			expected: "'00:45:00",
		},
		{
			name:     "Hours and minutes",
			duration: 2*time.Hour + 30*time.Minute,
			expected: "'02:30:00",
		},
		{
			name:     "Hours, minutes, seconds",
			duration: 3*time.Hour + 15*time.Minute + 42*time.Second,
			expected: "'03:15:42",
		},
		{
			name:     "Large duration",
			duration: 10*time.Hour + 5*time.Minute + 30*time.Second,
			expected: "'10:05:30",
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

func TestTravelTimeServiceCalculateTravelTimes(t *testing.T) {
	tts := NewTravelTimeService()
	ctx := context.Background()
	currentTime := time.Date(2022, 1, 1, 12, 0, 0, 0, time.UTC)
	updateInterval := 5 * time.Minute

	tests := []struct {
		name             string
		userID           int
		destination      string
		travelType       string
		expectedDuration time.Duration
	}{
		{
			name:             "Mexico regular travel",
			userID:           123,
			destination:      "Mexico",
			travelType:       "regular",
			expectedDuration: 26 * time.Minute,
		},
		{
			name:             "UK airstrip travel",
			userID:           456,
			destination:      "United Kingdom",
			travelType:       "airstrip",
			expectedDuration: 111 * time.Minute,
		},
		{
			name:             "Japan business class travel",
			userID:           789,
			destination:      "Japan",
			travelType:       "business",
			expectedDuration: 68 * time.Minute,
		},
		{
			name:             "Switzerland standard travel (with business arrival)",
			userID:           999,
			destination:      "Switzerland",
			travelType:       "standard",
			expectedDuration: 175 * time.Minute, // Regular duration
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tts.CalculateTravelTimes(ctx, tt.userID, tt.destination, tt.travelType, currentTime, updateInterval)

			if result == nil {
				t.Fatal("CalculateTravelTimes returned nil")
			}

			// Parse the departure and arrival times
			departureTime, err := time.Parse("2006-01-02 15:04:05", result.Departure)
			if err != nil {
				t.Fatalf("Failed to parse departure time: %v", err)
			}

			arrivalTime, err := time.Parse("2006-01-02 15:04:05", result.Arrival)
			if err != nil {
				t.Fatalf("Failed to parse arrival time: %v", err)
			}

			// Check that travel duration is correct
			actualDuration := arrivalTime.Sub(departureTime)
			if actualDuration != tt.expectedDuration {
				t.Errorf("Travel duration = %v, expected %v", actualDuration, tt.expectedDuration)
			}

			// Check that departure is approximately 2.5 minutes before current time
			expectedDeparture := currentTime.Add(-updateInterval / 2)
			if departureTime.Sub(expectedDeparture).Abs() > time.Second {
				t.Errorf("Departure time = %v, expected around %v", departureTime, expectedDeparture)
			}

			// Check countdown format
			if result.Countdown == "" {
				t.Error("Countdown should not be empty")
			}

			// For "standard" travel type, check BusinessArrival is calculated
			if tt.travelType == "standard" {
				if result.BusinessArrival == "" {
					t.Error("BusinessArrival should not be empty for standard travel")
				}

				// Parse business arrival time and verify it's before regular arrival
				businessArrival, err := time.Parse("2006-01-02 15:04:05", result.BusinessArrival)
				if err != nil {
					t.Fatalf("Failed to parse business arrival time: %v", err)
				}

				if !businessArrival.Before(arrivalTime) {
					t.Error("Business arrival should be before regular arrival for standard travel")
				}
			} else {
				// Non-standard travel should not have BusinessArrival
				if result.BusinessArrival != "" {
					t.Error("BusinessArrival should be empty for non-standard travel")
				}
			}
		})
	}
}

func TestTravelTimeServiceCalculateTravelTimesFromDeparture(t *testing.T) {
	tts := NewTravelTimeService()
	ls := NewLocationService()
	ctx := context.Background()
	currentTime := time.Date(2022, 1, 1, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name               string
		userID             int
		destination        string
		departureStr       string
		existingArrivalStr string
		travelType         string
		expectNil          bool
	}{
		{
			name:               "Valid departure with existing arrival",
			userID:             123,
			destination:        "Mexico",
			departureStr:       "2022-01-01 11:30:00",
			existingArrivalStr: "2022-01-01 11:56:00",
			travelType:         "regular",
			expectNil:          false,
		},
		{
			name:               "Valid departure without existing arrival",
			userID:             456,
			destination:        "United Kingdom",
			departureStr:       "2022-01-01 10:00:00",
			existingArrivalStr: "",
			travelType:         "airstrip",
			expectNil:          false,
		},
		{
			name:               "Invalid departure time",
			userID:             789,
			destination:        "Mexico",
			departureStr:       "invalid-time",
			existingArrivalStr: "",
			travelType:         "regular",
			expectNil:          true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tts.CalculateTravelTimesFromDeparture(
				ctx, tt.userID, tt.destination, tt.departureStr,
				tt.existingArrivalStr, tt.travelType, currentTime, ls, "",
			)

			if tt.expectNil {
				if result != nil {
					t.Error("Expected nil result for invalid input")
				}
				return
			}

			if result == nil {
				t.Fatal("CalculateTravelTimesFromDeparture returned nil unexpectedly")
			}

			// Check that departure time is preserved
			if result.Departure != tt.departureStr {
				t.Errorf("Departure = %q, expected %q", result.Departure, tt.departureStr)
			}

			// Check countdown format
			if result.Countdown == "" {
				t.Error("Countdown should not be empty")
			}

			// If existing arrival was provided, it should be preserved
			if tt.existingArrivalStr != "" && result.Arrival != tt.existingArrivalStr {
				t.Errorf("Arrival = %q, expected %q", result.Arrival, tt.existingArrivalStr)
			}
		})
	}
}

func TestTravelTimeServiceEdgeCases(t *testing.T) {
	tts := NewTravelTimeService()

	// Test unknown destination fallback
	duration := tts.GetTravelTime("NonExistentPlace", "regular")
	expected := 30 * time.Minute
	if duration != expected {
		t.Errorf("Unknown destination should return %v, got %v", expected, duration)
	}

	// Test airstrip vs regular times are different
	mexicoRegular := tts.GetTravelTime("Mexico", "regular")
	mexicoAirstrip := tts.GetTravelTime("Mexico", "airstrip")
	mexicoBusiness := tts.GetTravelTime("Mexico", "business")
	if mexicoRegular <= mexicoAirstrip {
		t.Error("Regular travel should be slower than airstrip travel")
	}
	if mexicoAirstrip <= mexicoBusiness {
		t.Error("Airstrip travel should be slower than business class travel")
	}
	if mexicoRegular <= mexicoBusiness {
		t.Error("Regular travel should be slower than business class travel")
	}

	// Test all destinations have regular, airstrip, and business class times
	destinations := []string{
		"Mexico", "Cayman Islands", "Canada", "Hawaii", "United Kingdom",
		"Argentina", "Switzerland", "Japan", "China", "UAE", "South Africa",
	}

	for _, dest := range destinations {
		regular := tts.GetTravelTime(dest, "regular")
		airstrip := tts.GetTravelTime(dest, "airstrip")
		business := tts.GetTravelTime(dest, "business")

		if regular == 30*time.Minute || airstrip == 30*time.Minute || business == 30*time.Minute {
			t.Errorf("Destination %q should have predefined times, not fallback", dest)
		}

		if regular <= airstrip {
			t.Errorf("Regular travel to %q should be slower than airstrip", dest)
		}
		if airstrip <= business {
			t.Errorf("Airstrip travel to %q should be slower than business class", dest)
		}
		if regular <= business {
			t.Errorf("Regular travel to %q should be slower than business class", dest)
		}
	}
}

// Benchmark the travel time calculation
func BenchmarkTravelTimeServiceGetTravelTime(b *testing.B) {
	tts := NewTravelTimeService()
	destinations := []string{"Mexico", "United Kingdom", "Japan", "Unknown"}
	travelTypes := []string{"regular", "airstrip", "business"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		dest := destinations[i%len(destinations)]
		travelType := travelTypes[i%len(travelTypes)]
		tts.GetTravelTime(dest, travelType)
	}
}

func BenchmarkTravelTimeServiceFormatTravelTime(b *testing.B) {
	tts := NewTravelTimeService()
	durations := []time.Duration{
		30 * time.Minute,
		2*time.Hour + 30*time.Minute,
		5 * time.Hour,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		duration := durations[i%len(durations)]
		tts.FormatTravelTime(duration)
	}
}
