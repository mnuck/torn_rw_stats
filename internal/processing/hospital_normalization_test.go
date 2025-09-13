package processing

import (
	"testing"

	"torn_rw_stats/internal/app"
)

// TestNormalizeHospitalDescription tests hospital description normalization for state change detection
func TestNormalizeHospitalDescription(t *testing.T) {
	service := NewStateChangeDetectionService(nil)

	tests := []struct {
		name        string
		description string
		expected    string
	}{
		// Generic hospital patterns
		{
			name:        "Generic hospital with countdown",
			description: "In hospital for 2hrs 15mins",
			expected:    "In hospital",
		},
		{
			name:        "Generic hospital without countdown",
			description: "In hospital",
			expected:    "In hospital",
		},
		// Country-specific hospital patterns (the bug we're fixing)
		{
			name:        "South African hospital with countdown",
			description: "In a South African hospital for 33 mins",
			expected:    "In hospital",
		},
		{
			name:        "British hospital with countdown",
			description: "In a British hospital for 2hrs 30mins",
			expected:    "In hospital",
		},
		{
			name:        "Mexican hospital with countdown",
			description: "In a Mexican hospital for 45mins",
			expected:    "In hospital",
		},
		{
			name:        "Swiss hospital with countdown",
			description: "In a Swiss hospital for 1hr 20mins",
			expected:    "In hospital",
		},
		// Case variations
		{
			name:        "Mixed case country hospital",
			description: "IN A CHINESE HOSPITAL FOR 25MINS",
			expected:    "In hospital",
		},
		{
			name:        "Lower case country hospital",
			description: "in a japanese hospital for 1hr",
			expected:    "In hospital",
		},
		// Non-hospital descriptions should remain unchanged
		{
			name:        "Traveling status",
			description: "Traveling to Mexico",
			expected:    "Traveling to Mexico",
		},
		{
			name:        "Okay status",
			description: "Okay",
			expected:    "Okay",
		},
		{
			name:        "Activity status",
			description: "At the gym",
			expected:    "At the gym",
		},
		// Edge cases
		{
			name:        "Empty description",
			description: "",
			expected:    "",
		},
		{
			name:        "Partial hospital match",
			description: "Was in hospital yesterday",
			expected:    "Was in hospital yesterday",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.NormalizeHospitalDescription(tt.description)
			if result != tt.expected {
				t.Errorf("NormalizeHospitalDescription(%q) = %q, expected %q",
					tt.description, result, tt.expected)
			}
		})
	}
}

// TestHospitalCountdownStateChangeIgnored tests that hospital countdown changes don't trigger state changes
func TestHospitalCountdownStateChangeIgnored(t *testing.T) {
	service := NewStateChangeDetectionService(nil)

	// Create two member states representing the bug scenario
	oldMember := app.FactionMember{
		Name:  "fangYuannn",
		Level: 50,
		Status: app.MemberStatus{
			Description: "In a South African hospital for 33 mins",
			State:       "Hospital",
			Color:       "red",
			Details:     "Mugged by someone",
		},
		LastAction: app.LastAction{
			Status: "Idle",
		},
	}

	newMember := app.FactionMember{
		Name:  "fangYuannn",
		Level: 50,
		Status: app.MemberStatus{
			Description: "In a South African hospital for 28 mins", // Countdown decreased
			State:       "Hospital",
			Color:       "red",
			Details:     "Mugged by someone",
		},
		LastAction: app.LastAction{
			Status: "Idle",
		},
	}

	// This should NOT be considered a status change
	hasChanged := service.HasStatusChanged(oldMember, newMember)
	if hasChanged {
		t.Error("Expected no status change for hospital countdown progression, but got change detected")
	}
}

// TestActualHospitalStateChangesDetected tests that real hospital state changes are still detected
func TestActualHospitalStateChangesDetected(t *testing.T) {
	service := NewStateChangeDetectionService(nil)

	tests := []struct {
		name         string
		oldMember    app.FactionMember
		newMember    app.FactionMember
		expectChange bool
	}{
		{
			name: "Hospital state to okay state",
			oldMember: app.FactionMember{
				Status: app.MemberStatus{
					Description: "In a South African hospital for 5 mins",
					State:       "Hospital",
					Color:       "red",
				},
			},
			newMember: app.FactionMember{
				Status: app.MemberStatus{
					Description: "Okay",
					State:       "Okay",
					Color:       "green",
				},
			},
			expectChange: true,
		},
		{
			name: "Hospital color change",
			oldMember: app.FactionMember{
				Status: app.MemberStatus{
					Description: "In a British hospital for 30 mins",
					State:       "Hospital",
					Color:       "red",
				},
			},
			newMember: app.FactionMember{
				Status: app.MemberStatus{
					Description: "In a British hospital for 25 mins",
					State:       "Hospital",
					Color:       "orange", // Color changed
				},
			},
			expectChange: true,
		},
		{
			name: "Last action status change",
			oldMember: app.FactionMember{
				Status: app.MemberStatus{
					Description: "In hospital for 15 mins",
					State:       "Hospital",
				},
				LastAction: app.LastAction{
					Status: "Idle",
				},
			},
			newMember: app.FactionMember{
				Status: app.MemberStatus{
					Description: "In hospital for 10 mins",
					State:       "Hospital",
				},
				LastAction: app.LastAction{
					Status: "Online", // Last action changed
				},
			},
			expectChange: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hasChanged := service.HasStatusChanged(tt.oldMember, tt.newMember)
			if hasChanged != tt.expectChange {
				t.Errorf("Expected HasStatusChanged = %v, got %v", tt.expectChange, hasChanged)
			}
		})
	}
}
