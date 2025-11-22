package state

import (
	"testing"

	"torn_rw_stats/internal/app"
)

func TestDetermineFactionsToTrack(t *testing.T) {
	tests := []struct {
		name             string
		changes          []app.StateChangeRecord
		currentStates    map[int]app.StateRecord
		expectedFactions []int
		expectedReasons  map[int]string
	}{
		{
			name: "tracks hospital admissions",
			changes: []app.StateChangeRecord{
				{
					FactionID:    100,
					MemberID:     1,
					CurrentState: "Hospital",
					StatusState:  "Hospital",
				},
			},
			currentStates:    make(map[int]app.StateRecord),
			expectedFactions: []int{100},
			expectedReasons:  map[int]string{100: "Hospital"},
		},
		{
			name: "tracks travel departures",
			changes: []app.StateChangeRecord{
				{
					FactionID:    100,
					MemberID:     1,
					CurrentState: "Traveling",
					StatusState:  "Traveling",
				},
			},
			currentStates:    make(map[int]app.StateRecord),
			expectedFactions: []int{100},
			expectedReasons:  map[int]string{100: "Traveling"},
		},
		{
			name: "tracks federal jail",
			changes: []app.StateChangeRecord{
				{
					FactionID:    100,
					MemberID:     1,
					CurrentState: "Federal",
					StatusState:  "Federal",
				},
			},
			currentStates:    make(map[int]app.StateRecord),
			expectedFactions: []int{100},
			expectedReasons:  map[int]string{100: "Federal"},
		},
		{
			name: "deduplicates factions",
			changes: []app.StateChangeRecord{
				{FactionID: 100, CurrentState: "Hospital", StatusState: "Hospital"},
				{FactionID: 100, CurrentState: "Traveling", StatusState: "Traveling"},
			},
			currentStates:    make(map[int]app.StateRecord),
			expectedFactions: []int{100}, // Should only appear once
		},
		{
			name: "ignores insignificant changes",
			changes: []app.StateChangeRecord{
				{
					FactionID:    100,
					MemberID:     1,
					CurrentState: "Okay",
					StatusState:  "Okay",
				},
			},
			currentStates:    make(map[int]app.StateRecord),
			expectedFactions: []int{}, // Should not track
		},
		{
			name: "tracks multiple different factions",
			changes: []app.StateChangeRecord{
				{FactionID: 100, CurrentState: "Hospital", StatusState: "Hospital"},
				{FactionID: 200, CurrentState: "Traveling", StatusState: "Traveling"},
				{FactionID: 300, CurrentState: "Federal", StatusState: "Federal"},
			},
			currentStates:    make(map[int]app.StateRecord),
			expectedFactions: []int{100, 200, 300},
		},
		{
			name:             "handles empty changes",
			changes:          []app.StateChangeRecord{},
			currentStates:    make(map[int]app.StateRecord),
			expectedFactions: []int{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			plan := DetermineFactionsToTrack(tt.changes, tt.currentStates)

			if len(plan.FactionsToTrack) != len(tt.expectedFactions) {
				t.Errorf("expected %d factions, got %d", len(tt.expectedFactions), len(plan.FactionsToTrack))
			}

			// Verify specific factions
			for _, expectedID := range tt.expectedFactions {
				found := false
				for _, actualID := range plan.FactionsToTrack {
					if actualID == expectedID {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("expected faction %d not found in tracking plan", expectedID)
				}
			}

			// Verify reasons if provided
			if tt.expectedReasons != nil {
				for factionID, expectedReason := range tt.expectedReasons {
					actualReason, exists := plan.Reason[factionID]
					if !exists {
						t.Errorf("expected reason for faction %d not found", factionID)
					} else if actualReason != expectedReason {
						t.Errorf("faction %d: expected reason %s, got %s", factionID, expectedReason, actualReason)
					}
				}
			}
		})
	}
}

func TestIsSignificantChange(t *testing.T) {
	tests := []struct {
		name     string
		change   app.StateChangeRecord
		expected bool
	}{
		{
			name: "hospital in StatusState is significant",
			change: app.StateChangeRecord{
				StatusState:  "Hospital",
				CurrentState: "Okay",
			},
			expected: true,
		},
		{
			name: "hospital in CurrentState is significant",
			change: app.StateChangeRecord{
				StatusState:  "Okay",
				CurrentState: "Hospital",
			},
			expected: true,
		},
		{
			name: "traveling in StatusState is significant",
			change: app.StateChangeRecord{
				StatusState:  "Traveling",
				CurrentState: "Okay",
			},
			expected: true,
		},
		{
			name: "traveling in CurrentState is significant",
			change: app.StateChangeRecord{
				StatusState:  "Okay",
				CurrentState: "Traveling",
			},
			expected: true,
		},
		{
			name: "federal in StatusState is significant",
			change: app.StateChangeRecord{
				StatusState:  "Federal",
				CurrentState: "Okay",
			},
			expected: true,
		},
		{
			name: "federal in CurrentState is significant",
			change: app.StateChangeRecord{
				StatusState:  "Okay",
				CurrentState: "Federal",
			},
			expected: true,
		},
		{
			name: "okay status is not significant",
			change: app.StateChangeRecord{
				StatusState:  "Okay",
				CurrentState: "Okay",
			},
			expected: false,
		},
		{
			name: "idle status is not significant",
			change: app.StateChangeRecord{
				StatusState:  "Idle",
				CurrentState: "Idle",
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isSignificantChange(tt.change)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

// TestDetermineFactionsToTrackEdgeCases tests edge cases and boundary conditions
func TestDetermineFactionsToTrackEdgeCases(t *testing.T) {
	tests := []struct {
		name             string
		changes          []app.StateChangeRecord
		currentStates    map[int]app.StateRecord
		expectedFactions []int
		description      string
	}{
		{
			name:             "handles nil changes slice",
			changes:          nil,
			currentStates:    make(map[int]app.StateRecord),
			expectedFactions: []int{},
			description:      "should handle nil changes without panicking",
		},
		{
			name:             "handles nil current states map",
			changes:          []app.StateChangeRecord{{FactionID: 100, StatusState: "Hospital"}},
			currentStates:    nil,
			expectedFactions: []int{100},
			description:      "should handle nil current states",
		},
		{
			name: "handles very large faction IDs",
			changes: []app.StateChangeRecord{
				{FactionID: 999999999, StatusState: "Hospital"},
			},
			currentStates:    make(map[int]app.StateRecord),
			expectedFactions: []int{999999999},
			description:      "should handle very large faction IDs",
		},
		{
			name: "handles negative faction IDs",
			changes: []app.StateChangeRecord{
				{FactionID: -1, StatusState: "Hospital"},
			},
			currentStates:    make(map[int]app.StateRecord),
			expectedFactions: []int{-1},
			description:      "should handle negative faction IDs (edge case)",
		},
		{
			name: "handles zero faction ID",
			changes: []app.StateChangeRecord{
				{FactionID: 0, StatusState: "Hospital"},
			},
			currentStates:    make(map[int]app.StateRecord),
			expectedFactions: []int{0},
			description:      "should handle zero faction ID",
		},
		{
			name: "deduplicates many changes for same faction",
			changes: func() []app.StateChangeRecord {
				changes := make([]app.StateChangeRecord, 100)
				for i := 0; i < 100; i++ {
					changes[i] = app.StateChangeRecord{
						FactionID:    100,
						StatusState:  "Hospital",
						CurrentState: "Hospital",
					}
				}
				return changes
			}(),
			currentStates:    make(map[int]app.StateRecord),
			expectedFactions: []int{100},
			description:      "should deduplicate many changes for same faction",
		},
		{
			name: "tracks many different factions",
			changes: func() []app.StateChangeRecord {
				changes := make([]app.StateChangeRecord, 1000)
				for i := 0; i < 1000; i++ {
					changes[i] = app.StateChangeRecord{
						FactionID:    i + 1,
						StatusState:  "Hospital",
						CurrentState: "Hospital",
					}
				}
				return changes
			}(),
			currentStates: make(map[int]app.StateRecord),
			expectedFactions: func() []int {
				factions := make([]int, 1000)
				for i := 0; i < 1000; i++ {
					factions[i] = i + 1
				}
				return factions
			}(),
			description: "should handle tracking many different factions",
		},
		{
			name: "mixed significant and insignificant changes",
			changes: []app.StateChangeRecord{
				{FactionID: 100, StatusState: "Okay"},      // Not significant
				{FactionID: 101, StatusState: "Hospital"},  // Significant
				{FactionID: 102, StatusState: "Idle"},      // Not significant
				{FactionID: 103, StatusState: "Traveling"}, // Significant
				{FactionID: 104, StatusState: "Online"},    // Not significant
				{FactionID: 105, StatusState: "Federal"},   // Significant
			},
			currentStates:    make(map[int]app.StateRecord),
			expectedFactions: []int{101, 103, 105},
			description:      "should filter out insignificant changes correctly",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			plan := DetermineFactionsToTrack(tt.changes, tt.currentStates)

			if len(plan.FactionsToTrack) != len(tt.expectedFactions) {
				t.Errorf("%s: expected %d factions, got %d", tt.description, len(tt.expectedFactions), len(plan.FactionsToTrack))
			}

			// Verify all expected factions are present
			for _, expectedID := range tt.expectedFactions {
				found := false
				for _, actualID := range plan.FactionsToTrack {
					if actualID == expectedID {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("%s: expected faction %d not found in tracking plan", tt.description, expectedID)
				}
			}
		})
	}
}

// TestIsSignificantChangeEdgeCases tests edge cases for significance detection
func TestIsSignificantChangeEdgeCases(t *testing.T) {
	tests := []struct {
		name        string
		change      app.StateChangeRecord
		expected    bool
		description string
	}{
		{
			name: "empty strings are not significant",
			change: app.StateChangeRecord{
				StatusState:  "",
				CurrentState: "",
			},
			expected:    false,
			description: "empty state strings should not be significant",
		},
		{
			name: "both fields have significant status",
			change: app.StateChangeRecord{
				StatusState:  "Hospital",
				CurrentState: "Traveling",
			},
			expected:    true,
			description: "should be significant if either field has significant status",
		},
		{
			name: "case sensitivity check - lowercase hospital",
			change: app.StateChangeRecord{
				StatusState:  "hospital",
				CurrentState: "Okay",
			},
			expected:    false,
			description: "lowercase 'hospital' should not match (case sensitive)",
		},
		{
			name: "case sensitivity check - UPPERCASE HOSPITAL",
			change: app.StateChangeRecord{
				StatusState:  "HOSPITAL",
				CurrentState: "Okay",
			},
			expected:    false,
			description: "uppercase 'HOSPITAL' should not match (case sensitive)",
		},
		{
			name: "whitespace in status",
			change: app.StateChangeRecord{
				StatusState:  " Hospital ",
				CurrentState: "Okay",
			},
			expected:    false,
			description: "status with whitespace should not match",
		},
		{
			name: "partial match should not be significant",
			change: app.StateChangeRecord{
				StatusState:  "Hospitalized",
				CurrentState: "Okay",
			},
			expected:    false,
			description: "partial match of 'Hospital' should not be significant",
		},
		{
			name: "jail vs federal distinction",
			change: app.StateChangeRecord{
				StatusState:  "Jail",
				CurrentState: "Okay",
			},
			expected:    false,
			description: "regular 'Jail' should not be significant (only 'Federal')",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isSignificantChange(tt.change)
			if result != tt.expected {
				t.Errorf("%s: expected %v, got %v", tt.description, tt.expected, result)
			}
		})
	}
}

// TestTrackingPlanStructure tests the structure and integrity of tracking plans
func TestTrackingPlanStructure(t *testing.T) {
	tests := []struct {
		name        string
		changes     []app.StateChangeRecord
		description string
	}{
		{
			name: "reason map contains all tracked factions",
			changes: []app.StateChangeRecord{
				{FactionID: 100, CurrentState: "Hospital", StatusState: "Hospital"},
				{FactionID: 200, CurrentState: "Traveling", StatusState: "Traveling"},
				{FactionID: 300, CurrentState: "Federal", StatusState: "Federal"},
			},
			description: "reason map should have entries for all tracked factions",
		},
		{
			name: "reason map does not contain untracked factions",
			changes: []app.StateChangeRecord{
				{FactionID: 100, CurrentState: "Hospital", StatusState: "Hospital"},
				{FactionID: 200, CurrentState: "Okay", StatusState: "Okay"}, // Not tracked
			},
			description: "reason map should not have entries for untracked factions",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			plan := DetermineFactionsToTrack(tt.changes, make(map[int]app.StateRecord))

			// Verify all tracked factions have reasons
			for _, factionID := range plan.FactionsToTrack {
				if _, exists := plan.Reason[factionID]; !exists {
					t.Errorf("%s: tracked faction %d missing from reason map", tt.description, factionID)
				}
			}

			// Verify no extra reasons exist
			if len(plan.Reason) != len(plan.FactionsToTrack) {
				t.Errorf("%s: reason map size (%d) doesn't match tracked factions count (%d)",
					tt.description, len(plan.Reason), len(plan.FactionsToTrack))
			}

			// Verify reasons are not empty strings
			for factionID, reason := range plan.Reason {
				if reason == "" {
					t.Errorf("%s: faction %d has empty reason", tt.description, factionID)
				}
			}
		})
	}
}

// TestTrackingPlanConsistency verifies consistent behavior across multiple calls
func TestTrackingPlanConsistency(t *testing.T) {
	changes := []app.StateChangeRecord{
		{FactionID: 100, StatusState: "Hospital"},
		{FactionID: 200, StatusState: "Traveling"},
		{FactionID: 300, StatusState: "Okay"}, // Not tracked
	}

	// Run same input multiple times
	plan1 := DetermineFactionsToTrack(changes, make(map[int]app.StateRecord))
	plan2 := DetermineFactionsToTrack(changes, make(map[int]app.StateRecord))
	plan3 := DetermineFactionsToTrack(changes, make(map[int]app.StateRecord))

	// All should produce same number of tracked factions
	if len(plan1.FactionsToTrack) != len(plan2.FactionsToTrack) ||
		len(plan2.FactionsToTrack) != len(plan3.FactionsToTrack) {
		t.Error("multiple calls with same input should produce same number of tracked factions")
	}

	// All should have same reasons
	for factionID, reason1 := range plan1.Reason {
		if reason2, exists := plan2.Reason[factionID]; !exists || reason1 != reason2 {
			t.Errorf("faction %d has inconsistent reasons between calls", factionID)
		}
		if reason3, exists := plan3.Reason[factionID]; !exists || reason1 != reason3 {
			t.Errorf("faction %d has inconsistent reasons between calls", factionID)
		}
	}
}
