package state

import (
	"torn_rw_stats/internal/app"
)

// TrackingPlan describes which factions should be tracked
type TrackingPlan struct {
	FactionsToTrack []int
	Reason          map[int]string // Why each faction should be tracked
}

// DetermineFactionsToTrack decides which factions need tracking based on state changes
func DetermineFactionsToTrack(
	changes []app.StateChangeRecord,
	currentStates map[int]app.StateRecord,
) TrackingPlan {
	plan := TrackingPlan{
		FactionsToTrack: make([]int, 0),
		Reason:          make(map[int]string),
	}

	factionsSeen := make(map[int]bool)

	for _, change := range changes {
		if factionsSeen[change.FactionID] {
			continue
		}

		// Track factions with significant state changes
		if isSignificantChange(change) {
			plan.FactionsToTrack = append(plan.FactionsToTrack, change.FactionID)
			plan.Reason[change.FactionID] = change.CurrentState
			factionsSeen[change.FactionID] = true
		}
	}

	return plan
}

// isSignificantChange determines if a state change warrants tracking
func isSignificantChange(change app.StateChangeRecord) bool {
	// Track hospital admissions
	if change.StatusState == "Hospital" || change.CurrentState == "Hospital" {
		return true
	}

	// Track travel departures
	if change.StatusState == "Traveling" || change.CurrentState == "Traveling" {
		return true
	}

	// Track federal jail
	if change.StatusState == "Federal" || change.CurrentState == "Federal" {
		return true
	}

	return false
}
