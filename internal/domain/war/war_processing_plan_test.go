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
