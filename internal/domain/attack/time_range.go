package attack

import "torn_rw_stats/internal/app"

// TimeRangeResult holds the calculated time range and update mode for fetching attacks
type TimeRangeResult struct {
	FromTime   int64
	ToTime     int64
	UpdateMode string
}

// UpdateMode constants
const (
	UpdateModeFull        = "full"
	UpdateModeIncremental = "incremental"
)

// CalculateTimeRange determines the time range and update mode for fetching attacks
// Pure function: Takes currentTime as parameter to enable deterministic testing
func CalculateTimeRange(
	war *app.War,
	latestExistingTimestamp *int64,
	currentTime int64,
) TimeRangeResult {
	var fromTime, toTime int64
	updateMode := UpdateModeFull

	if latestExistingTimestamp != nil && *latestExistingTimestamp > 0 {
		// Incremental update mode - only fetch new attacks
		updateMode = UpdateModeIncremental

		// Add 1-hour buffer to handle timing discrepancies
		const bufferTime = 3600 // 1 hour in seconds
		fromTime = *latestExistingTimestamp - bufferTime

		// Ensure we don't go before war start
		if fromTime < war.Start {
			fromTime = war.Start
		}
	} else {
		// Full population mode - fetch entire war
		fromTime = war.Start
	}

	// Set end time
	if war.End != nil {
		toTime = *war.End
	} else {
		// Ongoing war - use current time
		toTime = currentTime
	}

	return TimeRangeResult{
		FromTime:   fromTime,
		ToTime:     toTime,
		UpdateMode: updateMode,
	}
}
