package processing

import (
	"testing"
)

func TestStateRecordComparator_normalizeStatusDescription(t *testing.T) {
	comparator := NewStateRecordComparator()

	tests := []struct {
		name        string
		description string
		expected    string
	}{
		{
			name:        "hospital with countdown",
			description: "In hospital for 3 hrs 5 mins ",
			expected:    "In hospital",
		},
		{
			name:        "hospital different format",
			description: "In a private hospital for 2 hours 30 minutes",
			expected:    "In hospital",
		},
		{
			name:        "jail with hrs format",
			description: "In jail for 4 hrs 14 mins ",
			expected:    "In jail",
		},
		{
			name:        "jail with hours format",
			description: "In jail for 4 hours 12 mins ",
			expected:    "In jail",
		},
		{
			name:        "jail with different countdown",
			description: "In jail for 1 hour 5 minutes",
			expected:    "In jail",
		},
		{
			name:        "regular status unchanged",
			description: "Okay",
			expected:    "Okay",
		},
		{
			name:        "traveling status unchanged",
			description: "Traveling to Japan",
			expected:    "Traveling to Japan",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := comparator.normalizeStatusDescription(tt.description)
			if result != tt.expected {
				t.Errorf("normalizeStatusDescription(%q) = %q, want %q", tt.description, result, tt.expected)
			}
		})
	}
}
