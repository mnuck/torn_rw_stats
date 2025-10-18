package war

import (
	"time"

	"torn_rw_stats/internal/app"

	"github.com/rs/zerolog/log"
)

// Configuration constants for war state management
const (
	// State update intervals
	NoWarsPlaceholderInterval  = 24 * time.Hour // Placeholder for NoWars (uses Tuesday matchmaking)
	PreWarUpdateInterval       = 5 * time.Minute
	ActiveWarUpdateInterval    = 1 * time.Minute
	PostWarPlaceholderInterval = 24 * time.Hour // Placeholder for PostWar (uses next week matchmaking)

	// State transition timing
	MinStateTransitionDuration = 30 * time.Second  // Minimum time in state before transition
	PreWarStartCheckOffset     = -2 * time.Minute  // Time before war start to check
	CheckTimeTolerance         = -30 * time.Second // Tolerance for check time comparison

	// War classification
	RecentlyEndedWarThreshold = 1 * time.Hour      // Wars ended within this time are "recent"
	PreWarSchedulingWindow    = 7 * 24 * time.Hour // Wars starting within 7 days are "upcoming"

	// Tuesday matchmaking configuration
	MatchmakingHour   = 12 // Matchmaking occurs at 12:05 UTC
	MatchmakingMinute = 5
	DaysInWeek        = 7
)

// WarState represents the different phases a faction can be in regarding wars,
// enabling intelligent API call optimization based on war lifecycle.
type WarState int

const (
	// NoWars indicates no active, upcoming, or recent wars exist
	NoWars WarState = iota

	// PreWar indicates a war is scheduled but hasn't started yet
	PreWar

	// ActiveWar indicates a war is currently in progress
	ActiveWar

	// PostWar indicates a war recently ended
	PostWar
)

// String returns the string representation of a war state
func (ws WarState) String() string {
	switch ws {
	case NoWars:
		return "NoWars"
	case PreWar:
		return "PreWar"
	case ActiveWar:
		return "ActiveWar"
	case PostWar:
		return "PostWar"
	default:
		return "Unknown"
	}
}

// WarStateConfig defines the behavior for each war state including update intervals,
// descriptions, and next check strategies.
type WarStateConfig struct {
	UpdateInterval    time.Duration // How often to check for updates
	Description       string        // Human-readable description
	NextCheckStrategy CheckStrategy // Strategy for determining next check
}

// CheckStrategy defines how to determine the next check time, supporting fixed
// intervals and smart waiting until specific events (Tuesday matchmaking, war start).
type CheckStrategy int

const (
	// FixedInterval indicates checking at regular time intervals
	FixedInterval CheckStrategy = iota

	// UntilTuesdayMatchmaking indicates waiting until next Tuesday 12:05 UTC matchmaking
	UntilTuesdayMatchmaking

	// UntilWarStart indicates waiting until the scheduled war start time
	UntilWarStart

	// UntilNextWeekMatchmaking indicates waiting until next week's Tuesday matchmaking
	UntilNextWeekMatchmaking
)

// WarStateManager manages war states and determines optimal check intervals based on
// war lifecycle, Tuesday matchmaking schedules, and state transition logic.
type WarStateManager struct {
	currentState    WarState
	lastStateChange time.Time
	currentWar      *app.War
	stateConfigs    map[WarState]WarStateConfig
}

// NewWarStateManager creates a new war state manager
func NewWarStateManager() *WarStateManager {
	return &WarStateManager{
		currentState:    NoWars,
		lastStateChange: time.Now(),
		stateConfigs: map[WarState]WarStateConfig{
			NoWars: {
				UpdateInterval:    NoWarsPlaceholderInterval,
				Description:       "No active wars - waiting for matchmaking",
				NextCheckStrategy: UntilTuesdayMatchmaking,
			},
			PreWar: {
				UpdateInterval:    PreWarUpdateInterval,
				Description:       "War scheduled - reconnaissance phase",
				NextCheckStrategy: FixedInterval,
			},
			ActiveWar: {
				UpdateInterval:    ActiveWarUpdateInterval,
				Description:       "War in progress - real-time monitoring",
				NextCheckStrategy: FixedInterval,
			},
			PostWar: {
				UpdateInterval:    PostWarPlaceholderInterval,
				Description:       "War completed - waiting for next matchmaking",
				NextCheckStrategy: UntilNextWeekMatchmaking,
			},
		},
	}
}

// UpdateState analyzes current war data and updates the state
func (wsm *WarStateManager) UpdateState(warResponse *app.WarResponse) WarState {
	newState := wsm.determineState(warResponse)

	// Validate state transition to prevent oscillation
	if wsm.isValidStateTransition(wsm.currentState, newState) {
		if newState != wsm.currentState {
			log.Info().
				Str("previous_state", wsm.currentState.String()).
				Str("new_state", newState.String()).
				Dur("time_in_previous_state", time.Since(wsm.lastStateChange)).
				Msg("War state transition")

			wsm.currentState = newState
			wsm.lastStateChange = time.Now()
		}
	} else {
		log.Debug().
			Str("current_state", wsm.currentState.String()).
			Str("attempted_state", newState.String()).
			Dur("time_since_last_change", time.Since(wsm.lastStateChange)).
			Msg("Invalid state transition blocked")
	}

	return wsm.currentState
}

// isValidStateTransition checks if a state transition is logically valid
func (wsm *WarStateManager) isValidStateTransition(from, to WarState) bool {
	// Allow same state (no transition)
	if from == to {
		return true
	}

	// Prevent rapid oscillation - require minimum time in state
	timeSinceLastChange := time.Since(wsm.lastStateChange)
	minTimeInState := MinStateTransitionDuration

	switch from {
	case NoWars:
		// From NoWars, can transition to any state
		return true

	case PreWar:
		// From PreWar, can go to:
		// - ActiveWar (war started)
		// - NoWars (war cancelled)
		// - PostWar (war scheduled but immediately ended/cancelled)
		// But prevent rapid oscillation between PreWar and NoWars
		if to == NoWars && timeSinceLastChange < minTimeInState {
			return false
		}
		return to == ActiveWar || to == NoWars || to == PostWar

	case ActiveWar:
		// From ActiveWar, can go to PostWar (war ended) or back to PreWar (very rare edge case)
		// Never allow direct transition back to NoWars from ActiveWar without going through PostWar
		return to == PostWar || to == PreWar

	case PostWar:
		// From PostWar, typically go to NoWars (post-war period expires)
		// Can go to PreWar if new war is scheduled quickly
		// Prevent rapid oscillation
		if (to == NoWars || to == PreWar) && timeSinceLastChange < minTimeInState {
			return false
		}
		return to == NoWars || to == PreWar

	default:
		return false
	}
}

// determineState analyzes war response and determines current state
func (wsm *WarStateManager) determineState(warResponse *app.WarResponse) WarState {
	now := time.Now()
	wars := wsm.getAllWars(warResponse)

	// Find the most relevant war using priority-based selection
	selectedWar, state := wsm.selectMostRelevantWar(wars, now)

	if selectedWar != nil {
		wsm.currentWar = selectedWar
		return state
	}

	// No relevant wars found
	wsm.currentWar = nil
	return NoWars
}

// selectMostRelevantWar chooses the most important war and its corresponding state
func (wsm *WarStateManager) selectMostRelevantWar(wars []app.War, now time.Time) (*app.War, WarState) {
	var activeWars, preWars, recentlyEndedWars []app.War

	// Categorize all wars
	for _, war := range wars {
		warStart := time.Unix(war.Start, 0)

		if now.After(warStart) {
			// War has started - check if it's still active
			if war.End != nil {
				warEnd := time.Unix(*war.End, 0)
				if now.Before(warEnd) {
					// Active war (started but not ended)
					activeWars = append(activeWars, war)
				} else if now.Sub(warEnd) <= RecentlyEndedWarThreshold {
					// Recently ended war
					recentlyEndedWars = append(recentlyEndedWars, war)
				}
			} else {
				// War started with no end time (still active)
				activeWars = append(activeWars, war)
			}
		} else if warStart.Sub(now) <= PreWarSchedulingWindow {
			// War scheduled within next 7 days
			preWars = append(preWars, war)
		}
	}

	// Priority 1: Active wars (choose the most recent one)
	if len(activeWars) > 0 {
		mostRecentActive := wsm.selectMostRecentWar(activeWars)
		return &mostRecentActive, ActiveWar
	}

	// Priority 2: Pre-wars (choose the soonest one)
	if len(preWars) > 0 {
		soonestPre := wsm.selectSoonestWar(preWars)
		return &soonestPre, PreWar
	}

	// Priority 3: Recently ended wars (choose the most recent one)
	if len(recentlyEndedWars) > 0 {
		mostRecentEnded := wsm.selectMostRecentEndedWar(recentlyEndedWars)
		return &mostRecentEnded, PostWar
	}

	return nil, NoWars
}

// selectMostRecentWar finds the war with the latest start time
func (wsm *WarStateManager) selectMostRecentWar(wars []app.War) app.War {
	if len(wars) == 0 {
		return app.War{}
	}

	mostRecent := wars[0]
	for _, war := range wars[1:] {
		if war.Start > mostRecent.Start {
			mostRecent = war
		}
	}
	return mostRecent
}

// selectSoonestWar finds the war with the earliest start time
func (wsm *WarStateManager) selectSoonestWar(wars []app.War) app.War {
	if len(wars) == 0 {
		return app.War{}
	}

	soonest := wars[0]
	for _, war := range wars[1:] {
		if war.Start < soonest.Start {
			soonest = war
		}
	}
	return soonest
}

// selectMostRecentEndedWar finds the war with the latest end time
func (wsm *WarStateManager) selectMostRecentEndedWar(wars []app.War) app.War {
	if len(wars) == 0 {
		return app.War{}
	}

	mostRecent := wars[0]
	for _, war := range wars[1:] {
		if war.End != nil && mostRecent.End != nil && *war.End > *mostRecent.End {
			mostRecent = war
		}
	}
	return mostRecent
}

// getAllWars extracts all wars from the response
func (wsm *WarStateManager) getAllWars(warResponse *app.WarResponse) []app.War {
	var wars []app.War

	if warResponse.Wars.Ranked != nil {
		wars = append(wars, *warResponse.Wars.Ranked)
	}

	wars = append(wars, warResponse.Wars.Raids...)
	wars = append(wars, warResponse.Wars.Territory...)

	return wars
}

// GetNextCheckTime calculates when the next check should occur
func (wsm *WarStateManager) GetNextCheckTime() time.Time {
	config := wsm.stateConfigs[wsm.currentState]
	now := time.Now()

	switch config.NextCheckStrategy {
	case FixedInterval:
		return now.Add(config.UpdateInterval)

	case UntilTuesdayMatchmaking:
		return wsm.getNextTuesdayMatchmaking(now)

	case UntilWarStart:
		if wsm.currentWar != nil {
			warStart := time.Unix(wsm.currentWar.Start, 0)
			// Check a few minutes before war starts
			return warStart.Add(PreWarStartCheckOffset)
		}
		// Fallback to fixed interval if no war info
		return now.Add(config.UpdateInterval)

	case UntilNextWeekMatchmaking:
		return wsm.getNextTuesdayMatchmaking(now)

	default:
		return now.Add(config.UpdateInterval)
	}
}

// getNextTuesdayMatchmaking calculates the next Tuesday 12:05 UTC
func (wsm *WarStateManager) getNextTuesdayMatchmaking(now time.Time) time.Time {
	// Convert to UTC for consistency
	nowUTC := now.UTC()

	// Find next Tuesday
	daysUntilTuesday := (int(time.Tuesday) - int(nowUTC.Weekday()) + DaysInWeek) % DaysInWeek
	if daysUntilTuesday == 0 {
		// It's Tuesday - check if we're past matchmaking time
		matchmakingTime := time.Date(nowUTC.Year(), nowUTC.Month(), nowUTC.Day(), MatchmakingHour, MatchmakingMinute, 0, 0, time.UTC)
		if nowUTC.After(matchmakingTime) {
			// Past today's matchmaking, wait for next week
			daysUntilTuesday = DaysInWeek
		}
	}

	nextTuesday := nowUTC.AddDate(0, 0, daysUntilTuesday)

	// Set to 12:05 UTC (5 minutes after matchmaking starts)
	matchmakingTime := time.Date(
		nextTuesday.Year(),
		nextTuesday.Month(),
		nextTuesday.Day(),
		MatchmakingHour, MatchmakingMinute, 0, 0,
		time.UTC,
	)

	return matchmakingTime
}

// ShouldProcessNow determines if processing should happen now
func (wsm *WarStateManager) ShouldProcessNow() bool {
	// Always process for states that require immediate/real-time monitoring
	switch wsm.currentState {
	case ActiveWar:
		// Real-time monitoring - always process
		return true
	case PreWar:
		// Reconnaissance monitoring - always process
		return true
	case PostWar:
		// Always process briefly after war ends
		return true
	case NoWars:
		// Only process if it's time for matchmaking check
		nextCheck := wsm.GetNextCheckTime()
		return time.Now().After(nextCheck.Add(CheckTimeTolerance))
	default:
		// Default to time-based check
		nextCheck := wsm.GetNextCheckTime()
		return time.Now().After(nextCheck.Add(CheckTimeTolerance))
	}
}

// GetCurrentState returns the current war state
func (wsm *WarStateManager) GetCurrentState() WarState {
	return wsm.currentState
}

// GetStateConfig returns the configuration for the current state
func (wsm *WarStateManager) GetStateConfig() WarStateConfig {
	return wsm.stateConfigs[wsm.currentState]
}

// GetCurrentWar returns the current war if any
func (wsm *WarStateManager) GetCurrentWar() *app.War {
	return wsm.currentWar
}

// GetStateInfo returns detailed information about the current state
func (wsm *WarStateManager) GetStateInfo() WarStateInfo {
	config := wsm.stateConfigs[wsm.currentState]
	nextCheck := wsm.GetNextCheckTime()

	return WarStateInfo{
		State:          wsm.currentState,
		Description:    config.Description,
		TimeInState:    time.Since(wsm.lastStateChange),
		NextCheckTime:  nextCheck,
		TimeUntilCheck: time.Until(nextCheck),
		UpdateInterval: config.UpdateInterval,
		CurrentWar:     wsm.currentWar,
	}
}

// WarStateInfo provides comprehensive state information including current state,
// time in state, next check time, and current war details.
type WarStateInfo struct {
	State          WarState
	Description    string
	TimeInState    time.Duration
	NextCheckTime  time.Time
	TimeUntilCheck time.Duration
	UpdateInterval time.Duration
	CurrentWar     *app.War
}
