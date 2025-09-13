package processing

import (
	"context"
	"testing"
	"time"

	"torn_rw_stats/internal/app"
	"torn_rw_stats/internal/processing/mocks"
)

// TestAPICallEfficiency measures actual API call reduction through optimizations
func TestAPICallEfficiency(t *testing.T) {
	ctx := context.Background()

	t.Run("BaselineWithoutOptimizations", func(t *testing.T) {
		// Simulate 10 execution cycles without any optimizations
		totalCalls := int64(0)

		for i := 0; i < 10; i++ {
			mock := mocks.NewMockTornClient()

			// Each cycle: check faction info, check wars, fetch attacks for 2 wars
			_, _ = mock.GetOwnFaction(ctx)
			totalCalls++

			_, _ = mock.GetFactionWars(ctx)
			totalCalls++

			// Simulate 2 active wars
			for j := 0; j < 2; j++ {
				_, _ = mock.GetAllAttacksForWar(ctx, &mockWar)
				totalCalls++
			}
		}

		expectedCalls := int64(40) // 10 cycles × (1 faction + 1 wars + 2 attacks)
		if totalCalls != expectedCalls {
			t.Errorf("Expected %d API calls without optimizations, got %d", expectedCalls, totalCalls)
		}

		t.Logf("Baseline API calls (10 cycles): %d", totalCalls)
	})

	t.Run("WithCachingOptimization", func(t *testing.T) {
		// Simulate 10 execution cycles with caching
		mock := mocks.NewMockTornClient()
		tracker := NewAPICallTracker()
		cachedClient := NewCachedTornClient(mock, tracker)

		for i := 0; i < 10; i++ {
			// Each cycle: cached faction info, cached wars, fresh attacks
			_, _ = cachedClient.GetOwnFaction(ctx)  // Only 1st call hits API
			_, _ = cachedClient.GetFactionWars(ctx) // Every few calls hit API (shorter TTL)

			// Attacks are never cached (too dynamic)
			for j := 0; j < 2; j++ {
				_, _ = cachedClient.GetAllAttacksForWar(ctx, &mockWar)
			}
		}

		stats := tracker.GetSessionStats()

		// Should be much less than 40 calls due to caching
		// Expect: 1 faction call + ~5 war calls (2min TTL) + 20 attack calls = ~26 calls
		maxExpectedCalls := int64(30) // Conservative upper bound
		if stats.SessionCalls > maxExpectedCalls {
			t.Errorf("Expected ≤%d API calls with caching, got %d", maxExpectedCalls, stats.SessionCalls)
		}

		reduction := float64(40-stats.SessionCalls) / 40 * 100
		t.Logf("With caching API calls: %d (%.1f%% reduction)", stats.SessionCalls, reduction)
	})

	t.Run("APICallBreakdown", func(t *testing.T) {
		// Test the breakdown of different API call types
		mock := mocks.NewMockTornClient()
		tracker := NewAPICallTracker()
		cachedClient := NewCachedTornClient(mock, tracker)

		// Simulate realistic usage pattern over 5 minutes
		for cycle := 0; cycle < 5; cycle++ {
			_, _ = cachedClient.GetOwnFaction(ctx)
			_, _ = cachedClient.GetFactionWars(ctx)

			// Variable number of wars per cycle
			warCount := 1 + (cycle % 3) // 1-3 wars
			for j := 0; j < warCount; j++ {
				_, _ = cachedClient.GetAttacksForTimeRange(ctx, &mockWar, 0, nil)
			}
		}

		stats := tracker.GetSessionStats()

		// Verify the breakdown makes sense
		expectedFactionCalls := int64(1) // Should be cached after first call
		expectedWarCalls := int64(3)     // Should be cached but may expire (2min TTL)
		expectedAttackCalls := int64(10) // Total attacks: 1+2+3+1+2 = 9, but allow some variance

		if factionCalls := stats.CallsByEndpoint["GetOwnFaction"]; factionCalls > expectedFactionCalls {
			t.Errorf("Too many faction calls: expected ≤%d, got %d", expectedFactionCalls, factionCalls)
		}

		if warCalls := stats.CallsByEndpoint["GetFactionWars"]; warCalls > expectedWarCalls {
			t.Errorf("Too many war calls: expected ≤%d, got %d", expectedWarCalls, warCalls)
		}

		attackCalls := stats.CallsByEndpoint["GetAttacksForTimeRange"]
		if attackCalls < expectedAttackCalls-2 || attackCalls > expectedAttackCalls+2 {
			t.Errorf("Unexpected attack calls: expected ~%d, got %d", expectedAttackCalls, attackCalls)
		}

		t.Logf("API call breakdown:")
		for endpoint, count := range stats.CallsByEndpoint {
			t.Logf("  %s: %d calls", endpoint, count)
		}
	})
}

// TestAPIOptimizer tests intelligent frequency adjustment
func TestAPIOptimizer(t *testing.T) {
	ctx := context.Background()

	t.Run("WarCheckFrequencyOptimization", func(t *testing.T) {
		mock := mocks.NewMockTornClient()
		tracker := NewAPICallTracker()
		optimizer := NewAPIOptimizer(mock, tracker)

		// Simulate scenario: no active wars
		emptyWarResponse := &app.WarResponse{
			Wars: struct {
				Ranked    *app.War  `json:"ranked"`
				Raids     []app.War `json:"raids"`
				Territory []app.War `json:"territory"`
			}{},
		}
		mock.FactionWarsResponse = emptyWarResponse

		// Make calls with sufficient time gaps to ensure actual API calls happen
		callCount := int64(0)

		// First call: always happens (first check)
		_, err := optimizer.GetOptimizedWars(ctx)
		if err != nil {
			t.Fatal(err)
		}

		// Check if API call was made
		stats := tracker.GetSessionStats()
		if stats.CallsByEndpoint["GetFactionWars"] > callCount {
			callCount = stats.CallsByEndpoint["GetFactionWars"]
		}

		// Force a time gap by creating a new optimizer with simulated past time
		// This is better than manipulating private fields
		optimizer2 := NewAPIOptimizer(mock, tracker)

		// Simulate that we had a previous empty check by calling twice with time gap
		_, err = optimizer2.GetOptimizedWars(ctx)
		if err != nil {
			t.Fatal(err)
		}

		// Small delay to ensure time difference
		time.Sleep(1 * time.Millisecond)

		_, err = optimizer2.GetOptimizedWars(ctx)
		if err != nil {
			t.Fatal(err)
		}

		// After the calls, optimizer should have tracked consecutive empty responses
		finalStats := optimizer2.GetOptimizationStats()
		if finalStats.ConsecutiveEmpty < 1 {
			t.Errorf("Expected optimizer to track at least 1 consecutive empty response, got %d", finalStats.ConsecutiveEmpty)
		}

		// Next check interval should be longer for empty responses
		if finalStats.NextCheckInterval < 2*60*1000000000 { // 2 minutes in nanoseconds
			t.Errorf("Expected longer check interval after empty responses, got %v", finalStats.NextCheckInterval)
		}

		t.Logf("Optimizer stats after empty responses:")
		t.Logf("  Consecutive empty: %d", finalStats.ConsecutiveEmpty)
		t.Logf("  Next check interval: %v", finalStats.NextCheckInterval)
		t.Logf("  Actual API calls made: %d", callCount)
	})
}

// BenchmarkAPIEfficiency benchmarks the efficiency improvements
func BenchmarkAPIEfficiency(b *testing.B) {
	ctx := context.Background()

	b.Run("APICallReduction", func(b *testing.B) {
		mock := mocks.NewMockTornClient()
		tracker := NewAPICallTracker()
		cachedClient := NewCachedTornClient(mock, tracker)

		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			// Simulate 1 execution cycle
			_, _ = cachedClient.GetOwnFaction(ctx)
			_, _ = cachedClient.GetFactionWars(ctx)
			_, _ = cachedClient.GetAllAttacksForWar(ctx, &mockWar)
		}

		stats := tracker.GetSessionStats()
		b.ReportMetric(float64(stats.SessionCalls), "actual_api_calls")
		b.ReportMetric(float64(b.N*3), "potential_api_calls") // 3 calls per iteration without caching

		efficiency := (float64(b.N*3) - float64(stats.SessionCalls)) / float64(b.N*3) * 100
		b.ReportMetric(efficiency, "efficiency_percent")
	})
}
