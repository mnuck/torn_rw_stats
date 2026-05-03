package services

import (
	"context"
	"log/slog"
	"sync"
	"time"
)

// APICallTracker monitors and optimizes API call usage, tracking session statistics,
// endpoint-specific call counts, and providing predictions for future API usage.
type APICallTracker struct {
	sessionStart    time.Time
	sessionCalls    int64
	totalCalls      int64
	callsByEndpoint map[string]int64
	mutex           sync.RWMutex
}

// NewAPICallTracker creates a new API call tracker
func NewAPICallTracker() *APICallTracker {
	return &APICallTracker{
		sessionStart:    time.Now(),
		callsByEndpoint: make(map[string]int64),
	}
}

// RecordCall records an API call for tracking
func (t *APICallTracker) RecordCall(endpoint string) {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	t.sessionCalls++
	t.totalCalls++
	t.callsByEndpoint[endpoint]++
}

// GetSessionStats returns API call statistics for current session
func (t *APICallTracker) GetSessionStats() APICallStats {
	t.mutex.RLock()
	defer t.mutex.RUnlock()

	duration := time.Since(t.sessionStart)

	endpointCopy := make(map[string]int64)
	for k, v := range t.callsByEndpoint {
		endpointCopy[k] = v
	}

	callsPerMinute := 0.0
	if duration > 0 {
		callsPerMinute = float64(t.sessionCalls) / duration.Minutes()
	}

	return APICallStats{
		SessionCalls:    t.sessionCalls,
		TotalCalls:      t.totalCalls,
		SessionDuration: duration,
		CallsByEndpoint: endpointCopy,
		CallsPerMinute:  callsPerMinute,
	}
}

// ResetSession resets session-specific counters
func (t *APICallTracker) ResetSession() {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	t.sessionStart = time.Now()
	t.sessionCalls = 0
	// Keep total calls and endpoint breakdown for historical tracking
}

// LogSessionSummary logs a summary of API usage for the session
func (t *APICallTracker) LogSessionSummary(ctx context.Context) {
	stats := t.GetSessionStats()
	args := []any{
		"session_calls", stats.SessionCalls,
		"total_calls", stats.TotalCalls,
		"calls_per_minute", stats.CallsPerMinute,
		"session_duration", stats.SessionDuration,
	}
	for endpoint, count := range stats.CallsByEndpoint {
		args = append(args, endpoint+"_calls", count)
	}
	slog.Info("API call session summary", args...)
}

// APICallStats represents API call statistics for a session, including call counts,
// duration, breakdown by endpoint, and calls per minute rate.
type APICallStats struct {
	SessionCalls    int64
	TotalCalls      int64
	SessionDuration time.Duration
	CallsByEndpoint map[string]int64
	CallsPerMinute  float64
}

// PredictCallsForNextCycle estimates API calls needed for next execution cycle
func (t *APICallTracker) PredictCallsForNextCycle(activeWars int) int64 {
	// Base calls: GetFactionWars (1) + potential GetOwnFaction (0-1)
	baseCalls := int64(1)

	// Attack calls: depends on war state and update mode
	// Conservative estimate: 1-3 calls per war (incremental vs full population)
	attackCalls := int64(activeWars) * 2

	return baseCalls + attackCalls
}
