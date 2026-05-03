package services

import (
	"testing"

	"torn_rw_stats/internal/domain/travel"
)

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
