package travel

import "torn_rw_stats/internal/app"

// IsNewJourneyToDestination checks if a record represents a new journey to the destination.
// A journey is new if:
// - It's the first record (no previous state)
// - Previous status was not traveling
// - Previous destination was different
//
// Pure function: No I/O operations, fully testable with direct inputs.
func IsNewJourneyToDestination(records []app.StateRecord, currentIndex int, destination string, locationParser LocationParser) bool {
	if currentIndex == 0 {
		return true // First record is always a new journey
	}

	current := records[currentIndex]
	previous := records[currentIndex-1]

	// New journey if previous status was not traveling
	if previous.StatusState != "Traveling" {
		return true
	}

	// New journey if previous destination was different
	previousDestination := locationParser(previous.StatusDescription)
	currentDestination := locationParser(current.StatusDescription)

	return previousDestination != currentDestination
}
