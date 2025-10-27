package attack

import "torn_rw_stats/internal/app"

// PaginationDecision contains the result of analyzing a page of attacks
type PaginationDecision struct {
	ShouldStop       bool
	Reason           string
	OldestTimestamp  int64
	AttacksProcessed int
}

// ShouldStopPagination determines if backward pagination should stop
// Pure function: Makes pagination decision based on page results
func ShouldStopPagination(
	totalAttacksInPage int,
	oldestAttackTime int64,
	fetchStartTime int64,
	pageSize int,
) PaginationDecision {
	// No more attacks returned
	if totalAttacksInPage == 0 {
		return PaginationDecision{
			ShouldStop:       true,
			Reason:           "no_more_attacks",
			OldestTimestamp:  oldestAttackTime,
			AttacksProcessed: totalAttacksInPage,
		}
	}

	// Got less than full page - indicates end of available data
	if totalAttacksInPage < pageSize {
		return PaginationDecision{
			ShouldStop:       true,
			Reason:           "partial_page",
			OldestTimestamp:  oldestAttackTime,
			AttacksProcessed: totalAttacksInPage,
		}
	}

	// Reached the fetch start time - collected all attacks in range
	if oldestAttackTime <= fetchStartTime {
		return PaginationDecision{
			ShouldStop:       true,
			Reason:           "reached_start_time",
			OldestTimestamp:  oldestAttackTime,
			AttacksProcessed: totalAttacksInPage,
		}
	}

	// Continue pagination
	return PaginationDecision{
		ShouldStop:       false,
		Reason:           "continue",
		OldestTimestamp:  oldestAttackTime,
		AttacksProcessed: totalAttacksInPage,
	}
}

// FindOldestAttackTime finds the oldest (minimum) timestamp in a list of attacks
// Pure function: Simple reduction operation
func FindOldestAttackTime(attacks []app.Attack, defaultTime int64) int64 {
	if len(attacks) == 0 {
		return defaultTime
	}

	oldest := defaultTime
	for _, attack := range attacks {
		if attack.Started < oldest {
			oldest = attack.Started
		}
	}

	return oldest
}
