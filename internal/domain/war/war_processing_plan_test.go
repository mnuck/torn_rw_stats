package war

import (
	"testing"
	"time"

	"torn_rw_stats/internal/app"
)

func TestDetermineProcessingPlan(t *testing.T) {
	now := time.Now()
	warStart := now.Add(-2 * time.Hour)
	lastProcessed := now.Add(-30 * time.Minute)

	tests := []struct {
		name              string
		war               *app.War
		fullMode          bool
		lastProcessed     time.Time
		expectedFetchMode FetchMode
		checkTimeRange    bool
		expectedStart     time.Time
	}{
		{
			name: "full mode fetches all attacks",
			war: &app.War{
				ID:    12345,
				Start: warStart.Unix(),
			},
			fullMode:          true,
			lastProcessed:     lastProcessed,
			expectedFetchMode: FetchModeAll,
			checkTimeRange:    true,
			expectedStart:     warStart,
		},
		{
			name: "incremental mode fetches since last processed",
			war: &app.War{
				ID:    12345,
				Start: warStart.Unix(),
			},
			fullMode:          false,
			lastProcessed:     lastProcessed,
			expectedFetchMode: FetchModeIncremental,
			checkTimeRange:    true,
			expectedStart:     lastProcessed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			plan := DetermineProcessingPlan(tt.war, tt.fullMode, tt.lastProcessed)

			if plan.FetchMode != tt.expectedFetchMode {
				t.Errorf("expected fetch mode %s, got %s", tt.expectedFetchMode, plan.FetchMode)
			}

			if plan.WarID != tt.war.ID {
				t.Errorf("expected war ID %d, got %d", tt.war.ID, plan.WarID)
			}

			if !plan.RequiresSheets {
				t.Error("expected RequiresSheets to be true")
			}

			if len(plan.SheetNames) != 3 {
				t.Errorf("expected 3 sheet names, got %d", len(plan.SheetNames))
			}

			if tt.checkTimeRange {
				// Allow small time differences due to test execution time
				timeDiff := tt.expectedStart.Sub(plan.AttackTimeRange.Start)
				if timeDiff < 0 {
					timeDiff = -timeDiff
				}
				if timeDiff > time.Second {
					t.Errorf("expected start time %v, got %v (diff: %v)",
						tt.expectedStart, plan.AttackTimeRange.Start, timeDiff)
				}
			}
		})
	}
}

func TestShouldProcessWar(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name     string
		war      *app.War
		expected bool
	}{
		{
			name: "war in progress should be processed",
			war: &app.War{
				Start: now.Add(-1 * time.Hour).Unix(),
				End:   nil, // War still ongoing
			},
			expected: true,
		},
		{
			name: "war not yet started should not be processed",
			war: &app.War{
				Start: now.Add(1 * time.Hour).Unix(),
				End:   nil,
			},
			expected: false,
		},
		{
			name: "war ended more than 1 hour ago should not be processed",
			war: &app.War{
				Start: now.Add(-5 * time.Hour).Unix(),
				End:   ptrInt64(now.Add(-2 * time.Hour).Unix()),
			},
			expected: false,
		},
		{
			name: "war just ended (within 1 hour) should be processed",
			war: &app.War{
				Start: now.Add(-3 * time.Hour).Unix(),
				End:   ptrInt64(now.Add(-30 * time.Minute).Unix()),
			},
			expected: true,
		},
		{
			name: "war ended exactly 1 hour ago should not be processed",
			war: &app.War{
				Start: now.Add(-3 * time.Hour).Unix(),
				End:   ptrInt64(now.Add(-61 * time.Minute).Unix()), // Just over 1 hour
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ShouldProcessWar(tt.war, now)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

// ptrInt64 is a helper to create int64 pointers
func ptrInt64(i int64) *int64 {
	return &i
}

// TestDetermineProcessingPlanEdgeCases tests boundary conditions and edge cases
func TestDetermineProcessingPlanEdgeCases(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name              string
		war               *app.War
		fullMode          bool
		lastProcessed     time.Time
		expectedFetchMode FetchMode
		description       string
	}{
		{
			name: "war started in future (edge case)",
			war: &app.War{
				ID:    99999,
				Start: now.Add(24 * time.Hour).Unix(),
			},
			fullMode:          true,
			lastProcessed:     now,
			expectedFetchMode: FetchModeAll,
			description:       "future war should still generate valid plan",
		},
		{
			name: "war started very long ago (30 days)",
			war: &app.War{
				ID:    88888,
				Start: now.Add(-30 * 24 * time.Hour).Unix(),
			},
			fullMode:          true,
			lastProcessed:     now.Add(-29 * 24 * time.Hour),
			expectedFetchMode: FetchModeAll,
			description:       "very old war should work with full mode",
		},
		{
			name: "incremental with last processed equal to now",
			war: &app.War{
				ID:    77777,
				Start: now.Add(-2 * time.Hour).Unix(),
			},
			fullMode:          false,
			lastProcessed:     now,
			expectedFetchMode: FetchModeIncremental,
			description:       "zero-duration incremental update",
		},
		{
			name: "incremental with last processed in future",
			war: &app.War{
				ID:    66666,
				Start: now.Add(-2 * time.Hour).Unix(),
			},
			fullMode:          false,
			lastProcessed:     now.Add(1 * time.Hour),
			expectedFetchMode: FetchModeIncremental,
			description:       "last processed in future (clock skew scenario)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			plan := DetermineProcessingPlan(tt.war, tt.fullMode, tt.lastProcessed)

			if plan.FetchMode != tt.expectedFetchMode {
				t.Errorf("%s: expected fetch mode %s, got %s", tt.description, tt.expectedFetchMode, plan.FetchMode)
			}

			if plan.WarID != tt.war.ID {
				t.Errorf("expected war ID %d, got %d", tt.war.ID, plan.WarID)
			}

			if !plan.RequiresSheets {
				t.Error("expected RequiresSheets to be true")
			}

			// Verify sheet names format
			if len(plan.SheetNames) != 3 {
				t.Errorf("expected 3 sheet names, got %d", len(plan.SheetNames))
			}

			// Verify time range is set
			if plan.AttackTimeRange.Start.IsZero() {
				t.Error("expected non-zero start time")
			}

			if plan.AttackTimeRange.End.IsZero() {
				t.Error("expected non-zero end time")
			}
		})
	}
}

// TestShouldProcessWarEdgeCases tests edge cases for war processing decision
func TestShouldProcessWarEdgeCases(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name        string
		war         *app.War
		currentTime time.Time
		expected    bool
		description string
	}{
		{
			name: "war started exactly now",
			war: &app.War{
				Start: now.Unix(),
				End:   nil,
			},
			currentTime: now,
			expected:    true,
			description: "war starting exactly at current time should be processed",
		},
		{
			name: "war ended exactly 1 hour ago",
			war: &app.War{
				Start: now.Add(-3 * time.Hour).Unix(),
				End:   ptrInt64(now.Add(-1 * time.Hour).Unix()),
			},
			currentTime: now,
			expected:    false,
			description: "war ended exactly 1 hour ago should not be processed",
		},
		{
			name: "war ended 59 minutes 59 seconds ago",
			war: &app.War{
				Start: now.Add(-3 * time.Hour).Unix(),
				End:   ptrInt64(now.Add(-59*time.Minute - 59*time.Second).Unix()),
			},
			currentTime: now,
			expected:    true,
			description: "war ended just under 1 hour ago should be processed",
		},
		{
			name: "war starts 1 second in future",
			war: &app.War{
				Start: now.Add(1 * time.Second).Unix(),
				End:   nil,
			},
			currentTime: now,
			expected:    false,
			description: "war starting in near future should not be processed",
		},
		{
			name: "ongoing war without end time",
			war: &app.War{
				Start: now.Add(-48 * time.Hour).Unix(),
				End:   nil,
			},
			currentTime: now,
			expected:    true,
			description: "long-running war without end should be processed",
		},
		{
			name: "war ended long ago (1 week)",
			war: &app.War{
				Start: now.Add(-8 * 24 * time.Hour).Unix(),
				End:   ptrInt64(now.Add(-7 * 24 * time.Hour).Unix()),
			},
			currentTime: now,
			expected:    false,
			description: "war ended long ago should not be processed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ShouldProcessWar(tt.war, tt.currentTime)
			if result != tt.expected {
				t.Errorf("%s: expected %v, got %v", tt.description, tt.expected, result)
			}
		})
	}
}

// TestProcessingPlanSheetNames verifies correct sheet name generation
func TestProcessingPlanSheetNames(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name           string
		warID          int
		expectedSuffix string
	}{
		{name: "small war ID", warID: 123, expectedSuffix: "123"},
		{name: "large war ID", warID: 999999, expectedSuffix: "999999"},
		{name: "zero war ID", warID: 0, expectedSuffix: "0"},
		{name: "negative war ID", warID: -1, expectedSuffix: "-1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			war := &app.War{
				ID:    tt.warID,
				Start: now.Unix(),
			}

			plan := DetermineProcessingPlan(war, true, now)

			// Verify all three sheet names are generated
			if len(plan.SheetNames) != 3 {
				t.Fatalf("expected 3 sheet names, got %d", len(plan.SheetNames))
			}

			// Verify each sheet name contains the war ID
			expectedNames := []string{
				"Summary - " + tt.expectedSuffix,
				"Records - " + tt.expectedSuffix,
				"Status - " + tt.expectedSuffix,
			}

			for i, expected := range expectedNames {
				if plan.SheetNames[i] != expected {
					t.Errorf("sheet %d: expected %q, got %q", i, expected, plan.SheetNames[i])
				}
			}
		})
	}
}

// TestFetchModeConsistency verifies fetch mode behavior is consistent
func TestFetchModeConsistency(t *testing.T) {
	now := time.Now()
	war := &app.War{
		ID:    12345,
		Start: now.Add(-10 * time.Hour).Unix(),
	}

	// Test multiple calls with same parameters return same result
	lastProcessed := now.Add(-1 * time.Hour)

	plan1 := DetermineProcessingPlan(war, false, lastProcessed)
	plan2 := DetermineProcessingPlan(war, false, lastProcessed)

	if plan1.FetchMode != plan2.FetchMode {
		t.Error("multiple calls with same parameters should return same fetch mode")
	}

	// Test that full mode always returns FetchModeAll
	for i := 0; i < 5; i++ {
		plan := DetermineProcessingPlan(war, true, now.Add(-time.Duration(i)*time.Hour))
		if plan.FetchMode != FetchModeAll {
			t.Errorf("full mode should always return FetchModeAll, got %s", plan.FetchMode)
		}
	}

	// Test that incremental mode always returns FetchModeIncremental
	for i := 0; i < 5; i++ {
		plan := DetermineProcessingPlan(war, false, now.Add(-time.Duration(i)*time.Hour))
		if plan.FetchMode != FetchModeIncremental {
			t.Errorf("incremental mode should always return FetchModeIncremental, got %s", plan.FetchMode)
		}
	}
}
