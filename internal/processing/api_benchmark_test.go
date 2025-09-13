package processing

import (
	"context"
	"testing"
	"time"

	"torn_rw_stats/internal/app"
	"torn_rw_stats/internal/processing/mocks"
)

// BenchmarkAPICallPatterns measures API call efficiency in different scenarios
func BenchmarkAPICallPatterns(b *testing.B) {
	ctx := context.Background()

	b.Run("WithoutCaching", func(b *testing.B) {
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			// Simulate a typical processing cycle without caching
			mock := mocks.NewMockTornClient()
			tracker := NewAPICallTracker()

			// Typical processing: faction info + wars + attacks for 2 wars
			_, _ = mock.GetOwnFaction(ctx)
			tracker.RecordCall("GetOwnFaction")

			_, _ = mock.GetFactionWars(ctx)
			tracker.RecordCall("GetFactionWars")

			// Simulate 2 active wars requiring attack data
			for j := 0; j < 2; j++ {
				_, _ = mock.GetAllAttacksForWar(ctx, &mockWar)
				tracker.RecordCall("GetAllAttacksForWar")
			}

			// Record final call count for this iteration
			stats := tracker.GetSessionStats()
			if stats.SessionCalls != 4 { // 1 + 1 + 2
				b.Errorf("Expected 4 API calls, got %d", stats.SessionCalls)
			}
		}
	})

	b.Run("WithCaching", func(b *testing.B) {
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			// Simulate processing with caching - only first iteration makes API calls
			mock := mocks.NewMockTornClient()
			tracker := NewAPICallTracker()
			cachedClient := NewCachedTornClient(mock, tracker)

			// First call should hit API
			_, _ = cachedClient.GetOwnFaction(ctx)
			_, _ = cachedClient.GetFactionWars(ctx)

			// Subsequent calls within TTL should use cache
			for j := 0; j < 10; j++ {
				_, _ = cachedClient.GetOwnFaction(ctx)
				_, _ = cachedClient.GetFactionWars(ctx)
			}

			// Attack data is never cached (too dynamic)
			for j := 0; j < 2; j++ {
				_, _ = cachedClient.GetAllAttacksForWar(ctx, &mockWar)
			}

			stats := tracker.GetSessionStats()
			// Should be 4 calls: 1 faction + 1 wars + 2 attacks (cached calls don't count)
			if stats.SessionCalls != 4 {
				b.Errorf("Expected 4 API calls with caching, got %d", stats.SessionCalls)
			}
		}
	})

	b.Run("MultipleExecutionCycles", func(b *testing.B) {
		b.ResetTimer()

		// Simulate multiple execution cycles (typical for long-running service)
		mock := mocks.NewMockTornClient()
		tracker := NewAPICallTracker()
		cachedClient := NewCachedTornClient(mock, tracker)

		for i := 0; i < b.N; i++ {
			// Each cycle: check wars, process attacks
			_, _ = cachedClient.GetFactionWars(ctx)

			// Process 1-3 active wars per cycle
			warCount := 1 + (i % 3) // Vary between 1-3 wars
			for j := 0; j < warCount; j++ {
				_, _ = cachedClient.GetAttacksForTimeRange(ctx, &mockWar, 0, nil)
			}

			// Simulate some execution delay
			if i%10 == 0 {
				time.Sleep(time.Millisecond)
			}
		}

		stats := tracker.GetSessionStats()
		b.ReportMetric(float64(stats.SessionCalls), "api_calls")
		b.ReportMetric(stats.CallsPerMinute, "calls_per_minute")
	})
}

// BenchmarkCacheEffectiveness measures cache hit rates
func BenchmarkCacheEffectiveness(b *testing.B) {
	ctx := context.Background()

	b.Run("FactionInfoCaching", func(b *testing.B) {
		mock := mocks.NewMockTornClient()
		tracker := NewAPICallTracker()
		cachedClient := NewCachedTornClient(mock, tracker)

		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			_, _ = cachedClient.GetOwnFaction(ctx)
		}

		stats := tracker.GetSessionStats()
		cacheStats := cachedClient.GetCacheStats()

		// Should only make 1 API call regardless of N
		expectedCalls := int64(1)
		if stats.SessionCalls != expectedCalls {
			b.Errorf("Expected %d API call, got %d", expectedCalls, stats.SessionCalls)
		}

		b.ReportMetric(float64(cacheStats.ValidEntries), "cache_valid_entries")
		b.ReportMetric(float64(stats.SessionCalls), "actual_api_calls")
		b.ReportMetric(float64(b.N), "potential_api_calls")

		// Calculate cache efficiency
		efficiency := (float64(b.N) - float64(stats.SessionCalls)) / float64(b.N) * 100
		b.ReportMetric(efficiency, "cache_efficiency_percent")
	})
}

// BenchmarkAPICallPrediction tests prediction accuracy
func BenchmarkAPICallPrediction(b *testing.B) {
	tracker := NewAPICallTracker()

	scenarios := []struct {
		name       string
		activeWars int
		expected   int64
	}{
		{"NoActiveWars", 0, 1}, // Just GetFactionWars
		{"OneWar", 1, 3},       // GetFactionWars + 2 attack calls avg
		{"ThreeWars", 3, 7},    // GetFactionWars + 6 attack calls avg
		{"TenWars", 10, 21},    // GetFactionWars + 20 attack calls avg
	}

	for _, scenario := range scenarios {
		b.Run(scenario.name, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				predicted := tracker.PredictCallsForNextCycle(scenario.activeWars)
				if predicted != scenario.expected {
					b.Errorf("Prediction mismatch for %d wars: expected %d, got %d",
						scenario.activeWars, scenario.expected, predicted)
				}
			}

			b.ReportMetric(float64(scenario.expected), "predicted_calls")
		})
	}
}

// Mock data for benchmarks
var mockWar = mockWarForBenchmark()

func mockWarForBenchmark() app.War {
	return app.War{
		ID:    12345,
		Start: time.Now().Unix() - 3600, // 1 hour ago
		Factions: []app.Faction{
			{ID: 1001, Name: "Faction A"},
			{ID: 1002, Name: "Faction B"},
		},
	}
}
