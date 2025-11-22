package travel

import (
	"time"

	"torn_rw_stats/internal/app"
)

// LocationParser is a function type that parses location from status description
type LocationParser func(string) string

// FindLastDepartureToDestination finds the most recent departure time to a specific destination
// by scanning through chronologically sorted records and detecting new journeys.
//
// Pure function: No I/O operations, fully testable with direct inputs.
func FindLastDepartureToDestination(records []app.StateRecord, destination string, locationParser LocationParser) time.Time {
	var lastDeparture time.Time

	for i := 0; i < len(records); i++ {
		current := records[i]
		if current.StatusState != "Traveling" {
			continue
		}

		currentDestination := locationParser(current.StatusDescription)
		if currentDestination != destination {
			continue
		}

		// Check if this is a new journey (different from previous travel)
		if IsNewJourneyToDestination(records, i, destination, locationParser) {
			lastDeparture = current.Timestamp
		}
	}

	return lastDeparture
}
