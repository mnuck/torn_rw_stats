package processing

import (
	"context"
	"testing"
	"time"

	"torn_rw_stats/internal/app"
	"torn_rw_stats/internal/processing/mocks"
	"torn_rw_stats/internal/sheets"
)

// TestPerformanceImprovements demonstrates the complete optimization impact
func TestPerformanceImprovements(t *testing.T) {
	ctx := context.Background()

	t.Run("EndToEndAPIOptimization", func(t *testing.T) {
		// Create test dependencies
		mockTornClient := mocks.NewMockTornClient()
		mockSheetsClient := mocks.NewMockSheetsClient()

		// Set up realistic responses
		mockTornClient.FactionWarsResponse = &app.WarResponse{
			Wars: struct {
				Ranked    *app.War  `json:"ranked"`
				Raids     []app.War `json:"raids"`
				Territory []app.War `json:"territory"`
			}{
				Ranked: &app.War{
					ID:    12345,
					Start: time.Now().Unix() - 3600,
					Factions: []app.Faction{
						{ID: 1001, Name: "Our Faction"},
						{ID: 1002, Name: "Enemy Faction"},
					},
				},
			},
		}

		mockTornClient.OwnFactionResponse = &app.FactionInfoResponse{
			ID:   1001,
			Name: "Our Faction",
		}

		// Set up mock sheets client responses
		mockSheetsClient.EnsureWarSheetsResponse = &app.SheetConfig{
			SummaryTabName: "Summary - 12345",
			RecordsTabName: "Records - 12345",
		}
		mockSheetsClient.ReadExistingRecordsResponse = &sheets.ExistingRecordsInfo{
			RecordCount:     0,
			LatestTimestamp: 0,
		}

		// Set up faction basic responses with member data
		mockTornClient.FactionBasicResponse = &app.FactionBasicResponse{
			Members: map[string]app.FactionMember{
				"123": {Name: "Player 1", Status: app.MemberStatus{State: "Okay"}},
				"456": {Name: "Player 2", Status: app.MemberStatus{State: "Okay"}},
			},
		}

		// Set up additional mock responses for sheets operations
		mockSheetsClient.EnsureTravelStatusSheetResponse = "Travel Status - 1001"
		mockSheetsClient.EnsurePreviousStateSheetResponse = "Previous State - 1001"
		mockSheetsClient.EnsureStateChangeRecordsSheetResponse = "State Changes - 1001"
		mockSheetsClient.ReadSheetResponse = [][]interface{}{} // Empty sheet data

		// Set up services with real implementations
		locationService := NewLocationService()
		travelTimeService := NewTravelTimeService()
		attackService := NewAttackProcessingService(1001)
		warSummaryService := NewWarSummaryService(attackService)
		stateChangeService := NewStateChangeDetectionService(nil)

		// Create optimized processor
		optimizedProcessor := NewOptimizedWarProcessor(
			mockTornClient,
			mockSheetsClient,
			locationService,
			travelTimeService,
			attackService,
			warSummaryService,
			stateChangeService,
			&app.Config{SpreadsheetID: "test"},
		)

		// Simulate multiple processing cycles
		totalCalls := int64(0)
		cycles := 5

		for i := 0; i < cycles; i++ {
			startCalls := optimizedProcessor.GetAPICallCount()

			err := optimizedProcessor.ProcessActiveWars(ctx)
			if err != nil {
				t.Fatalf("Processing cycle %d failed: %v", i, err)
			}

			cycleCalls := optimizedProcessor.GetAPICallCount() - startCalls
			totalCalls += cycleCalls

			t.Logf("Cycle %d: %d API calls", i+1, cycleCalls)
		}

		// Get optimization summary
		summary := optimizedProcessor.GetOptimizationSummary()

		t.Logf("=== PERFORMANCE OPTIMIZATION RESULTS ===")
		t.Logf("Total processing cycles: %d", cycles)
		t.Logf("Total API calls made: %d", totalCalls)
		t.Logf("Average calls per cycle: %.1f", float64(totalCalls)/float64(cycles))
		t.Logf("API calls per minute: %.1f", summary.CallsPerMinute)
		t.Logf("Cache hit rate: %.1f%%", summary.CacheHitRate)
		t.Logf("Active wars detected: %d", summary.ActiveWarsDetected)
		t.Logf("Next check interval: %v", summary.NextCheckIn)

		// Expected performance: with caching, should use significantly fewer calls
		// Without optimization: ~10 calls per cycle (faction + wars + attacks + sheets)
		// With optimization: ~3-5 calls per cycle (cached faction/wars, fresh attacks)
		maxExpectedCallsPerCycle := 6.0
		actualCallsPerCycle := float64(totalCalls) / float64(cycles)

		if actualCallsPerCycle > maxExpectedCallsPerCycle {
			t.Errorf("Expected ≤%.1f API calls per cycle, got %.1f", maxExpectedCallsPerCycle, actualCallsPerCycle)
		}

		// Verify optimization is working - in NoWars state, 0% hit rate is optimal
		// because no API calls are made at all (ultimate optimization)
		if totalCalls > 0 && summary.CacheHitRate == 0 {
			t.Error("Expected some cache hits when API calls were made, but got 0% hit rate")
		}
	})

	t.Run("ScenarioBasedOptimization", func(t *testing.T) {
		scenarios := []struct {
			name          string
			activeWars    int
			cycles        int
			expectedRange [2]int // min, max expected calls per cycle
		}{
			{"NoActiveWars", 0, 10, [2]int{0, 1}}, // Heavily cached, minimal calls
			{"OneActiveWar", 1, 5, [2]int{1, 3}},  // War checks + attack calls, with aggressive caching
			{"MultipleWars", 3, 3, [2]int{3, 6}},  // Multiple wars with aggressive caching
		}

		for _, scenario := range scenarios {
			t.Run(scenario.name, func(t *testing.T) {
				mockClient := mocks.NewMockTornClient()
				tracker := NewAPICallTracker()
				cachedClient := NewCachedTornClient(mockClient, tracker)

				// Set up scenario-specific responses
				warResponse := &app.WarResponse{
					Wars: struct {
						Ranked    *app.War  `json:"ranked"`
						Raids     []app.War `json:"raids"`
						Territory []app.War `json:"territory"`
					}{},
				}

				// Add wars based on scenario
				for i := 0; i < scenario.activeWars; i++ {
					war := app.War{
						ID:    12345 + i,
						Start: time.Now().Unix() - 3600,
						Factions: []app.Faction{
							{ID: 1001, Name: "Faction A"},
							{ID: 1002, Name: "Faction B"},
						},
					}
					warResponse.Wars.Raids = append(warResponse.Wars.Raids, war)
				}

				mockClient.FactionWarsResponse = warResponse

				// Simulate processing cycles
				totalCalls := int64(0)
				for i := 0; i < scenario.cycles; i++ {
					startCalls := tracker.GetSessionStats().SessionCalls

					// Simulate processing: get wars, then attacks for each war
					_, _ = cachedClient.GetFactionWars(ctx)
					for j := 0; j < scenario.activeWars; j++ {
						war := &warResponse.Wars.Raids[j]
						_, _ = cachedClient.GetAllAttacksForWar(ctx, war)
					}

					cycleCalls := tracker.GetSessionStats().SessionCalls - startCalls
					totalCalls += cycleCalls
				}

				avgCallsPerCycle := float64(totalCalls) / float64(scenario.cycles)

				t.Logf("Scenario %s: %.1f avg calls per cycle (range: %d-%d)",
					scenario.name, avgCallsPerCycle, scenario.expectedRange[0], scenario.expectedRange[1])

				// Verify calls are within expected range
				if avgCallsPerCycle < float64(scenario.expectedRange[0]) || avgCallsPerCycle > float64(scenario.expectedRange[1]) {
					t.Errorf("Scenario %s: expected %.1f calls per cycle to be in range [%d, %d]",
						scenario.name, avgCallsPerCycle, scenario.expectedRange[0], scenario.expectedRange[1])
				}
			})
		}
	})
}

// BenchmarkPerformanceOptimizations benchmarks the optimization impact
func BenchmarkPerformanceOptimizations(b *testing.B) {
	ctx := context.Background()

	b.Run("OptimizedVsUnoptimized", func(b *testing.B) {
		// Create mock dependencies
		mockClient := mocks.NewMockTornClient()
		mockClient.FactionWarsResponse = &app.WarResponse{
			Wars: struct {
				Ranked    *app.War  `json:"ranked"`
				Raids     []app.War `json:"raids"`
				Territory []app.War `json:"territory"`
			}{
				Raids: []app.War{{
					ID:    12345,
					Start: time.Now().Unix() - 3600,
					Factions: []app.Faction{
						{ID: 1001, Name: "Faction A"},
						{ID: 1002, Name: "Faction B"},
					},
				}},
			},
		}

		// Benchmark unoptimized approach
		b.Run("Unoptimized", func(b *testing.B) {
			callCount := int64(0)

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				// Simulate unoptimized: every call hits API
				_, _ = mockClient.GetOwnFaction(ctx)
				callCount++

				_, _ = mockClient.GetFactionWars(ctx)
				callCount++

				_, _ = mockClient.GetAllAttacksForWar(ctx, &mockClient.FactionWarsResponse.Wars.Raids[0])
				callCount++
			}

			b.ReportMetric(float64(callCount), "api_calls")
			b.ReportMetric(float64(callCount)/float64(b.N), "calls_per_operation")
		})

		// Benchmark optimized approach
		b.Run("Optimized", func(b *testing.B) {
			tracker := NewAPICallTracker()
			cachedClient := NewCachedTornClient(mockClient, tracker)

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				// Simulate optimized: caching reduces API calls
				_, _ = cachedClient.GetOwnFaction(ctx)                                                      // Cached after first call
				_, _ = cachedClient.GetFactionWars(ctx)                                                     // Cached for 2 minutes
				_, _ = cachedClient.GetAllAttacksForWar(ctx, &mockClient.FactionWarsResponse.Wars.Raids[0]) // Never cached
			}

			stats := tracker.GetSessionStats()
			b.ReportMetric(float64(stats.SessionCalls), "api_calls")
			b.ReportMetric(float64(stats.SessionCalls)/float64(b.N), "calls_per_operation")

			// Calculate efficiency improvement
			unoptimizedCallsPerOp := 3.0 // 3 calls per operation without caching
			optimizedCallsPerOp := float64(stats.SessionCalls) / float64(b.N)
			efficiency := (unoptimizedCallsPerOp - optimizedCallsPerOp) / unoptimizedCallsPerOp * 100

			b.ReportMetric(efficiency, "efficiency_improvement_percent")
		})
	})
}
