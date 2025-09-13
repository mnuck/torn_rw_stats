package processing

import (
	"context"
	"time"

	"torn_rw_stats/internal/app"

	"github.com/rs/zerolog/log"
)

// APIOptimizer implements intelligent API call reduction strategies
type APIOptimizer struct {
	client  TornClientInterface
	tracker *APICallTracker

	// State for optimization decisions
	lastWarCheck     time.Time
	lastKnownWars    []int // War IDs from last check
	consecutiveEmpty int   // Count of consecutive empty war responses
}

// NewAPIOptimizer creates a new API optimizer
func NewAPIOptimizer(client TornClientInterface, tracker *APICallTracker) *APIOptimizer {
	return &APIOptimizer{
		client:  client,
		tracker: tracker,
	}
}

// GetOptimizedWars fetches wars with intelligent frequency adjustment
func (o *APIOptimizer) GetOptimizedWars(ctx context.Context) (*app.WarResponse, error) {
	now := time.Now()

	// Skip war checks if we checked recently and have no active wars
	if o.shouldSkipWarCheck(now) {
		log.Debug().
			Dur("since_last_check", now.Sub(o.lastWarCheck)).
			Int("consecutive_empty", o.consecutiveEmpty).
			Msg("Skipping war check - recent empty results")

		// Return empty response to indicate skip
		return &app.WarResponse{
			Wars: struct {
				Ranked    *app.War  `json:"ranked"`
				Raids     []app.War `json:"raids"`
				Territory []app.War `json:"territory"`
			}{},
		}, nil
	}

	// Fetch wars normally
	wars, err := o.client.GetFactionWars(ctx)
	if err != nil {
		return nil, err
	}

	o.tracker.RecordCall("GetFactionWars")

	// Update optimization state
	o.updateWarState(wars, now)

	return wars, nil
}

// shouldSkipWarCheck determines if war checking can be skipped
func (o *APIOptimizer) shouldSkipWarCheck(now time.Time) bool {
	// Never skip on first check
	if o.lastWarCheck.IsZero() {
		return false
	}

	// If we have active wars, always check frequently
	if len(o.lastKnownWars) > 0 {
		// Active wars: check every 2 minutes max
		return now.Sub(o.lastWarCheck) < 2*time.Minute
	}

	// No active wars: use backoff strategy
	timeSinceCheck := now.Sub(o.lastWarCheck)

	switch {
	case o.consecutiveEmpty >= 10:
		// After 10 consecutive empty checks, check every 30 minutes
		return timeSinceCheck < 30*time.Minute
	case o.consecutiveEmpty >= 5:
		// After 5 consecutive empty checks, check every 15 minutes
		return timeSinceCheck < 15*time.Minute
	case o.consecutiveEmpty >= 2:
		// After 2 consecutive empty checks, check every 5 minutes
		return timeSinceCheck < 5*time.Minute
	default:
		// First few empty checks: maintain normal frequency
		return timeSinceCheck < 2*time.Minute
	}
}

// updateWarState updates internal state based on war response
func (o *APIOptimizer) updateWarState(wars *app.WarResponse, checkTime time.Time) {
	o.lastWarCheck = checkTime

	// Extract current war IDs
	var currentWars []int
	if wars.Wars.Ranked != nil {
		currentWars = append(currentWars, wars.Wars.Ranked.ID)
	}
	for _, war := range wars.Wars.Raids {
		currentWars = append(currentWars, war.ID)
	}
	for _, war := range wars.Wars.Territory {
		currentWars = append(currentWars, war.ID)
	}

	// Update consecutive empty counter
	if len(currentWars) == 0 {
		o.consecutiveEmpty++
	} else {
		o.consecutiveEmpty = 0
	}

	// Log optimization decision
	if len(currentWars) != len(o.lastKnownWars) {
		log.Info().
			Int("previous_wars", len(o.lastKnownWars)).
			Int("current_wars", len(currentWars)).
			Int("consecutive_empty", o.consecutiveEmpty).
			Msg("War count changed - optimization state updated")
	}

	o.lastKnownWars = currentWars
}

// GetOptimizationStats returns current optimization statistics
func (o *APIOptimizer) GetOptimizationStats() OptimizationStats {
	return OptimizationStats{
		LastWarCheck:      o.lastWarCheck,
		KnownActiveWars:   len(o.lastKnownWars),
		ConsecutiveEmpty:  o.consecutiveEmpty,
		NextCheckInterval: o.getNextCheckInterval(),
	}
}

// getNextCheckInterval calculates when the next war check should happen
func (o *APIOptimizer) getNextCheckInterval() time.Duration {
	if len(o.lastKnownWars) > 0 {
		return 2 * time.Minute // Active wars
	}

	switch {
	case o.consecutiveEmpty >= 10:
		return 30 * time.Minute
	case o.consecutiveEmpty >= 5:
		return 15 * time.Minute
	case o.consecutiveEmpty >= 2:
		return 5 * time.Minute
	default:
		return 2 * time.Minute
	}
}

// OptimizationStats represents API optimization statistics
type OptimizationStats struct {
	LastWarCheck      time.Time
	KnownActiveWars   int
	ConsecutiveEmpty  int
	NextCheckInterval time.Duration
}

// EstimateCallsForPeriod estimates API calls needed for a given time period
func (o *APIOptimizer) EstimateCallsForPeriod(duration time.Duration) APICallEstimate {
	stats := o.GetOptimizationStats()

	// Calculate war check frequency
	checkInterval := stats.NextCheckInterval
	warChecks := int64(duration / checkInterval)
	if warChecks == 0 {
		warChecks = 1 // At least one check
	}

	// Estimate attack calls based on active wars
	// Conservative estimate: 2 calls per war per check (pagination)
	attackCalls := int64(stats.KnownActiveWars) * warChecks * 2

	// Add occasional faction info calls (cached but may expire)
	factionCalls := int64(1) // Once per period typically

	total := warChecks + attackCalls + factionCalls

	return APICallEstimate{
		WarChecks:     warChecks,
		AttackCalls:   attackCalls,
		FactionCalls:  factionCalls,
		TotalEstimate: total,
		Period:        duration,
		CheckInterval: checkInterval,
	}
}

// APICallEstimate represents projected API usage
type APICallEstimate struct {
	WarChecks     int64
	AttackCalls   int64
	FactionCalls  int64
	TotalEstimate int64
	Period        time.Duration
	CheckInterval time.Duration
}
