package services

import (
	"context"
	"fmt"
	"sort"
	"time"

	"torn_rw_stats/internal/app"
)

// buildDepartureMap builds a map of member departure times from state changes
func (s *StatusV2Service) buildDepartureMap(ctx context.Context, spreadsheetID string, currentStateRecords []app.StateRecord) (map[string]time.Time, error) {
	departureMap := make(map[string]time.Time)

	// Read all state records from Changed States sheet
	allStateRecords, err := s.ReadAllStateRecords(ctx, spreadsheetID)
	if err != nil {
		return departureMap, fmt.Errorf("failed to read state records: %w", err)
	}

	// For each current traveling member, find their most recent transition to traveling
	for _, currentRecord := range currentStateRecords {
		if currentRecord.StatusState != "Traveling" {
			continue
		}

		memberKey := fmt.Sprintf("%s_%s", currentRecord.FactionID, currentRecord.MemberID)
		// Parse the current destination location instead of using raw status description
		currentParsedLocation := s.locationService.ParseLocation(currentRecord.StatusDescription)
		departureTime := s.findMostRecentTravelingTransition(allStateRecords, currentRecord.MemberID, currentParsedLocation)

		if !departureTime.IsZero() {
			departureMap[memberKey] = departureTime
		}
	}

	return departureMap, nil
}

// findMostRecentTravelingTransition finds when a member most recently started traveling to their current destination
func (s *StatusV2Service) findMostRecentTravelingTransition(allRecords []app.StateRecord, memberID, currentDestination string) time.Time {
	memberRecords := s.getMemberRecordsChronologically(allRecords, memberID)
	return s.findLastDepartureToDestination(memberRecords, currentDestination)
}

// getMemberRecordsChronologically filters and sorts records for a specific member
func (s *StatusV2Service) getMemberRecordsChronologically(allRecords []app.StateRecord, memberID string) []app.StateRecord {
	var records []app.StateRecord
	for _, record := range allRecords {
		if record.MemberID == memberID {
			records = append(records, record)
		}
	}

	sort.Slice(records, func(i, j int) bool {
		return records[i].Timestamp.Before(records[j].Timestamp)
	})

	return records
}

// findLastDepartureToDestination finds the most recent departure time to a specific destination
func (s *StatusV2Service) findLastDepartureToDestination(records []app.StateRecord, destination string) time.Time {
	var lastDeparture time.Time

	for i := 0; i < len(records); i++ {
		current := records[i]
		if current.StatusState != "Traveling" {
			continue
		}

		currentDestination := s.locationService.ParseLocation(current.StatusDescription)
		if currentDestination != destination {
			continue
		}

		// Check if this is a new journey (different from previous travel)
		if s.isNewJourneyToDestination(records, i, destination) {
			lastDeparture = current.Timestamp
		}
	}

	return lastDeparture
}

// isNewJourneyToDestination checks if this record represents a new journey to the destination
func (s *StatusV2Service) isNewJourneyToDestination(records []app.StateRecord, currentIndex int, destination string) bool {
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
	previousDestination := s.locationService.ParseLocation(previous.StatusDescription)
	currentDestination := s.locationService.ParseLocation(current.StatusDescription)

	return previousDestination != currentDestination
}
