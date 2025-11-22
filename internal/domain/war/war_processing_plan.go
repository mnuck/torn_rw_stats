package war

import (
	"fmt"
	"time"

	"torn_rw_stats/internal/app"
)

// ProcessingPlan describes what to fetch and how to process a war
type ProcessingPlan struct {
	WarID           int
	FetchMode       FetchMode
	AttackTimeRange TimeRange
	RequiresSheets  bool
	SheetNames      []string
}

// FetchMode describes how to fetch attack data
type FetchMode string

const (
	// FetchModeAll fetches all attacks from war start
	FetchModeAll FetchMode = "all"
	// FetchModeIncremental fetches only new attacks since last update
	FetchModeIncremental FetchMode = "incremental"
	// FetchModeNone skips attack fetching
	FetchModeNone FetchMode = "none"
)

// TimeRange represents a time range for fetching attacks
type TimeRange struct {
	Start time.Time
	End   time.Time
}

// DetermineProcessingPlan decides what to fetch and how based on war state
func DetermineProcessingPlan(
	war *app.War,
	fullMode bool,
	lastProcessedTime time.Time,
) ProcessingPlan {
	plan := ProcessingPlan{
		WarID:          war.ID,
		RequiresSheets: true,
	}

	if fullMode {
		plan.FetchMode = FetchModeAll
		plan.AttackTimeRange = TimeRange{
			Start: time.Unix(war.Start, 0),
			End:   time.Now(),
		}
	} else {
		plan.FetchMode = FetchModeIncremental
		plan.AttackTimeRange = TimeRange{
			Start: lastProcessedTime,
			End:   time.Now(),
		}
	}

	plan.SheetNames = []string{
		fmt.Sprintf("Summary - %d", war.ID),
		fmt.Sprintf("Records - %d", war.ID),
		fmt.Sprintf("Status - %d", war.ID),
	}

	return plan
}

// ShouldProcessWar determines if a war needs processing
func ShouldProcessWar(war *app.War, currentTime time.Time) bool {
	warStart := time.Unix(war.Start, 0)

	// War must be started
	if currentTime.Before(warStart) {
		return false
	}

	// If war has ended, check if it's within grace period
	if war.End != nil {
		warEnd := time.Unix(*war.End, 0)
		// Skip wars ended more than 1 hour ago
		if currentTime.After(warEnd.Add(1 * time.Hour)) {
			return false
		}
	}

	return true
}
