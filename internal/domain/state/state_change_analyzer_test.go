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
