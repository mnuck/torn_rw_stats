package services

import (
	"testing"
	"time"

	"torn_rw_stats/internal/app"
	"torn_rw_stats/internal/domain/travel"
)

func TestDepartureTimeDetectionWithParsedLocations(t *testing.T) {
	// Setup
	locationService := travel.NewLocationService()
	service := &StatusV2Service{
		locationService: locationService,
	}

	// King_Taurus scenario: travels to Switzerland, then returns to Torn
	memberID := "3330609"

	// Create state records representing King_Taurus's journey
	baseTime := time.Date(2025, 9, 18, 0, 13, 8, 0, time.UTC)

	allRecords := []app.StateRecord{
		// Initial departure to Switzerland at 0:13:08
		{
			Timestamp:         baseTime,
			MemberID:         memberID,
			StatusState:      "Traveling",
			StatusDescription: "Traveling to Switzerland",
		},
		// Multiple intermediate records (status changes while traveling)
		{
			Timestamp:         baseTime.Add(1 * time.Minute),
			MemberID:         memberID,
			StatusState:      "Traveling",
			StatusDescription: "Traveling to Switzerland",
		},
		// Direction change at 2:17:06 - starts returning
		{
			Timestamp:         baseTime.Add(2*time.Hour + 4*time.Minute + 6*time.Second),
			MemberID:         memberID,
			StatusState:      "Traveling",
			StatusDescription: "Returning to Torn from Switzerland",
		},
		// Continuing return journey
		{
			Timestamp:         baseTime.Add(2*time.Hour + 10*time.Minute),
			MemberID:         memberID,
			StatusState:      "Traveling",
			StatusDescription: "Returning to Torn from Switzerland",
		},
	}

	tests := []struct {
		name                string
		currentDestination  string
		expectedDeparture   time.Time
		description         string
	}{
		{
			name:               "Find departure for Switzerland trip",
			currentDestination: "Switzerland",
			expectedDeparture:  baseTime, // Should find initial departure at 0:13:08
			description:        "Should find when King_Taurus first started traveling to Switzerland",
		},
		{
			name:               "Find departure for return to Torn",
			currentDestination: "Torn",
			expectedDeparture:  baseTime.Add(2*time.Hour + 4*time.Minute + 6*time.Second), // Should find return departure at 2:17:06
			description:        "Should find when King_Taurus started returning to Torn",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.findMostRecentTravelingTransition(allRecords, memberID, tt.currentDestination)

			if !result.Equal(tt.expectedDeparture) {
				t.Errorf("findMostRecentTravelingTransition() = %v, expected %v",
					result.Format("2006-01-02 15:04:05"),
					tt.expectedDeparture.Format("2006-01-02 15:04:05"))
				t.Errorf("Description: %s", tt.description)
			}
		})
	}
}

func TestLocationParsingConsistency(t *testing.T) {
	locationService := travel.NewLocationService()

	tests := []struct {
		statusDescription string
		expectedLocation  string
	}{
		{"Traveling to Switzerland", "Switzerland"},
		{"Returning to Torn from Switzerland", "Torn"},
		{"In Switzerland", "Switzerland"},
		{"Okay", "Torn"},
	}

	for _, tt := range tests {
		t.Run(tt.statusDescription, func(t *testing.T) {
			result := locationService.ParseLocation(tt.statusDescription)
			if result != tt.expectedLocation {
				t.Errorf("ParseLocation(%q) = %q, expected %q",
					tt.statusDescription, result, tt.expectedLocation)
			}
		})
	}
}