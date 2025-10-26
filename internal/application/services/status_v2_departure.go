package services

import (
	"context"
	"fmt"
	"time"

	"torn_rw_stats/internal/app"
	"torn_rw_stats/internal/domain/state"
	"torn_rw_stats/internal/domain/travel"
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
	// Use domain function to filter and sort records
	memberRecords := state.GetMemberRecordsChronologically(allRecords, memberID)

	// Use domain function to find departure, passing location parser as dependency
	return travel.FindLastDepartureToDestination(memberRecords, currentDestination, s.locationService.ParseLocation)
}
