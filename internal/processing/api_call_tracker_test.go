package processing

import (
	"context"
	"testing"
)

func TestAPICallTracker_ResetSession(t *testing.T) {
	tracker := NewAPICallTracker()

	// Add some calls
	tracker.RecordCall("GetFactionWars")
	tracker.RecordCall("GetOwnFaction")
	tracker.RecordCall("GetFactionWars")

	// Verify calls were recorded
	stats := tracker.GetSessionStats()
	if stats.TotalCalls != 3 {
		t.Errorf("Expected 3 total calls before reset, got %d", stats.TotalCalls)
	}

	// Reset session
	tracker.ResetSession()

	// Verify session calls were reset (but total calls remain for history)
	stats = tracker.GetSessionStats()
	if stats.SessionCalls != 0 {
		t.Errorf("Expected 0 session calls after reset, got %d", stats.SessionCalls)
	}

	// Total calls and endpoint data should remain for historical tracking
	if stats.TotalCalls == 0 {
		t.Error("Expected total calls to be preserved after session reset")
	}
}

func TestAPICallTracker_LogSessionSummary(t *testing.T) {
	tracker := NewAPICallTracker()

	// Add some calls
	tracker.RecordCall("GetFactionWars")
	tracker.RecordCall("GetOwnFaction")
	tracker.RecordCall("GetFactionWars")
	tracker.RecordCall("GetAttacksForTimeRange")

	// This should not panic and should log the summary
	// We can't easily test the log output, but we can ensure it doesn't crash
	tracker.LogSessionSummary(context.Background())

	// Verify the tracker still works after logging
	stats := tracker.GetSessionStats()
	if stats.TotalCalls != 4 {
		t.Errorf("Expected 4 total calls after logging, got %d", stats.TotalCalls)
	}
}

func TestAPICallTracker_PredictCallsForNextCycle(t *testing.T) {
	tracker := NewAPICallTracker()

	// Add some calls to establish a pattern
	tracker.RecordCall("GetFactionWars")
	tracker.RecordCall("GetOwnFaction")
	tracker.RecordCall("GetAttacksForTimeRange")
	tracker.RecordCall("GetAttacksForTimeRange")

	activeWars := 2
	prediction := tracker.PredictCallsForNextCycle(activeWars)

	// Should return some prediction (exact value may vary based on implementation)
	if prediction < 0 {
		t.Errorf("Expected non-negative prediction, got %d", prediction)
	}

	// For a short cycle with existing calls, should predict at least current pattern
	if prediction == 0 && tracker.GetSessionStats().TotalCalls > 0 {
		t.Error("Expected non-zero prediction when there are existing calls")
	}
}