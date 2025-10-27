package attack

import (
	"testing"
	"torn_rw_stats/internal/app"
)

func TestCalculateTimeRange(t *testing.T) {
	const currentTime = 10000
	const warStart = 5000
	const warEnd = 8000

	tests := []struct {
		name                    string
		war                     *app.War
		latestExistingTimestamp *int64
		currentTime             int64
		expectedFromTime        int64
		expectedToTime          int64
		expectedUpdateMode      string
	}{
		{
			name: "FullModeWithEndedWar",
			war: &app.War{
				Start: warStart,
				End:   ptr(warEnd),
			},
			latestExistingTimestamp: nil,
			currentTime:             currentTime,
			expectedFromTime:        warStart,
			expectedToTime:          warEnd,
			expectedUpdateMode:      UpdateModeFull,
		},
		{
			name: "FullModeWithOngoingWar",
			war: &app.War{
				Start: warStart,
				End:   nil, // Ongoing
			},
			latestExistingTimestamp: nil,
			currentTime:             currentTime,
			expectedFromTime:        warStart,
			expectedToTime:          currentTime, // Uses current time
			expectedUpdateMode:      UpdateModeFull,
		},
		{
			name: "IncrementalModeWithBuffer",
			war: &app.War{
				Start: warStart,
				End:   ptr(warEnd),
			},
			latestExistingTimestamp: ptr(int64(9000)), // Far enough from warStart that buffer won't clamp
			currentTime:             currentTime,
			expectedFromTime:        9000 - 3600, // With 1-hour buffer = 5400
			expectedToTime:          warEnd,
			expectedUpdateMode:      UpdateModeIncremental,
		},
		{
			name: "IncrementalModeBufferClampedToWarStart",
			war: &app.War{
				Start: warStart,
				End:   ptr(warEnd),
			},
			latestExistingTimestamp: ptr(int64(5500)), // Close to war start
			currentTime:             currentTime,
			expectedFromTime:        warStart, // Clamped to war start (5500-3600=1900 < warStart)
			expectedToTime:          warEnd,
			expectedUpdateMode:      UpdateModeIncremental,
		},
		{
			name: "IncrementalModeOngoingWar",
			war: &app.War{
				Start: warStart,
				End:   nil,
			},
			latestExistingTimestamp: ptr(int64(9000)),
			currentTime:             currentTime,
			expectedFromTime:        9000 - 3600, // With buffer
			expectedToTime:          currentTime,
			expectedUpdateMode:      UpdateModeIncremental,
		},
		{
			name: "IncrementalModeZeroTimestamp",
			war: &app.War{
				Start: warStart,
				End:   ptr(warEnd),
			},
			latestExistingTimestamp: ptr(int64(0)), // Zero treated as no existing timestamp
			currentTime:             currentTime,
			expectedFromTime:        warStart,
			expectedToTime:          warEnd,
			expectedUpdateMode:      UpdateModeFull, // Falls back to full mode
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CalculateTimeRange(tt.war, tt.latestExistingTimestamp, tt.currentTime)

			if result.FromTime != tt.expectedFromTime {
				t.Errorf("FromTime: expected %d, got %d", tt.expectedFromTime, result.FromTime)
			}

			if result.ToTime != tt.expectedToTime {
				t.Errorf("ToTime: expected %d, got %d", tt.expectedToTime, result.ToTime)
			}

			if result.UpdateMode != tt.expectedUpdateMode {
				t.Errorf("UpdateMode: expected %q, got %q", tt.expectedUpdateMode, result.UpdateMode)
			}
		})
	}
}

// Helper function to create int64 pointer
func ptr(i int64) *int64 {
	return &i
}
