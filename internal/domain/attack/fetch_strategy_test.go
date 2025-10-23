package attack

import (
	"testing"
	"time"
)

func TestDetermineFetchStrategy(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name           string
		startTime      time.Time
		endTime        time.Time
		expectedMethod FetchMethod
	}{
		{
			name:           "30 minute range uses simple fetch",
			startTime:      now.Add(-30 * time.Minute),
			endTime:        now,
			expectedMethod: FetchMethodSimple,
		},
		{
			name:           "1 hour range uses simple fetch",
			startTime:      now.Add(-1 * time.Hour),
			endTime:        now,
			expectedMethod: FetchMethodSimple,
		},
		{
			name:           "12 hour range uses simple fetch",
			startTime:      now.Add(-12 * time.Hour),
			endTime:        now,
			expectedMethod: FetchMethodSimple,
		},
		{
			name:           "24 hour range uses simple fetch",
			startTime:      now.Add(-24 * time.Hour),
			endTime:        now,
			expectedMethod: FetchMethodSimple,
		},
		{
			name:           "25 hour range uses paginated fetch",
			startTime:      now.Add(-25 * time.Hour),
			endTime:        now,
			expectedMethod: FetchMethodPaginated,
		},
		{
			name:           "48 hour range uses paginated fetch",
			startTime:      now.Add(-48 * time.Hour),
			endTime:        now,
			expectedMethod: FetchMethodPaginated,
		},
		{
			name:           "1 week range uses paginated fetch",
			startTime:      now.Add(-7 * 24 * time.Hour),
			endTime:        now,
			expectedMethod: FetchMethodPaginated,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			strategy := DetermineFetchStrategy(tt.startTime, tt.endTime)

			if strategy.Method != tt.expectedMethod {
				t.Errorf("expected method %s, got %s", tt.expectedMethod, strategy.Method)
			}

			// Verify pagination config is set correctly
			if strategy.Method == FetchMethodSimple && strategy.Pagination.Enabled {
				t.Error("simple method should not have pagination enabled")
			}

			if strategy.Method == FetchMethodPaginated && !strategy.Pagination.Enabled {
				t.Error("paginated method should have pagination enabled")
			}

			// Verify time range is set
			if !strategy.TimeRange.Start.Equal(tt.startTime) {
				t.Errorf("expected start time %v, got %v", tt.startTime, strategy.TimeRange.Start)
			}

			if !strategy.TimeRange.End.Equal(tt.endTime) {
				t.Errorf("expected end time %v, got %v", tt.endTime, strategy.TimeRange.End)
			}
		})
	}
}

func TestShouldUseSimpleApproach(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name      string
		startTime time.Time
		endTime   time.Time
		expected  bool
	}{
		{
			name:      "5 minute range should use simple",
			startTime: now.Add(-5 * time.Minute),
			endTime:   now,
			expected:  true,
		},
		{
			name:      "1 hour range should use simple",
			startTime: now.Add(-1 * time.Hour),
			endTime:   now,
			expected:  true,
		},
		{
			name:      "24 hour range should use simple",
			startTime: now.Add(-24 * time.Hour),
			endTime:   now,
			expected:  true,
		},
		{
			name:      "exactly 24 hour range should use simple",
			startTime: now.Add(-24 * time.Hour),
			endTime:   now,
			expected:  true,
		},
		{
			name:      "25 hour range should not use simple",
			startTime: now.Add(-25 * time.Hour),
			endTime:   now,
			expected:  false,
		},
		{
			name:      "48 hour range should not use simple",
			startTime: now.Add(-48 * time.Hour),
			endTime:   now,
			expected:  false,
		},
		{
			name:      "zero duration should use simple",
			startTime: now,
			endTime:   now,
			expected:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ShouldUseSimpleApproach(tt.startTime, tt.endTime)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestEstimateAPICallsRequired(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name             string
		strategy         FetchStrategy
		expectedMinCalls int
		expectedMaxCalls int
	}{
		{
			name: "simple fetch requires 1 call",
			strategy: FetchStrategy{
				Method: FetchMethodSimple,
			},
			expectedMinCalls: 1,
			expectedMaxCalls: 1,
		},
		{
			name: "1 hour paginated fetch estimates low calls",
			strategy: FetchStrategy{
				Method: FetchMethodPaginated,
				TimeRange: TimeRange{
					Start: now.Add(-1 * time.Hour),
					End:   now,
				},
			},
			expectedMinCalls: 1,
			expectedMaxCalls: 1,
		},
		{
			name: "10 hour paginated fetch estimates 1 call",
			strategy: FetchStrategy{
				Method: FetchMethodPaginated,
				TimeRange: TimeRange{
					Start: now.Add(-10 * time.Hour),
					End:   now,
				},
			},
			expectedMinCalls: 1,
			expectedMaxCalls: 1,
		},
		{
			name: "24 hour paginated fetch estimates 2-3 calls",
			strategy: FetchStrategy{
				Method: FetchMethodPaginated,
				TimeRange: TimeRange{
					Start: now.Add(-24 * time.Hour),
					End:   now,
				},
			},
			expectedMinCalls: 2,
			expectedMaxCalls: 3,
		},
		{
			name: "100 hour paginated fetch estimates 10 calls",
			strategy: FetchStrategy{
				Method: FetchMethodPaginated,
				TimeRange: TimeRange{
					Start: now.Add(-100 * time.Hour),
					End:   now,
				},
			},
			expectedMinCalls: 10,
			expectedMaxCalls: 10,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			calls := EstimateAPICallsRequired(tt.strategy)

			if calls < tt.expectedMinCalls || calls > tt.expectedMaxCalls {
				t.Errorf("expected %d-%d calls, got %d", tt.expectedMinCalls, tt.expectedMaxCalls, calls)
			}
		})
	}
}
