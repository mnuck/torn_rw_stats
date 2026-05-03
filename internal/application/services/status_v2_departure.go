package services

import (
	"context"
	"fmt"
	"time"

	"torn_rw_stats/internal/app"
	"torn_rw_stats/internal/domain/state"
	"torn_rw_stats/internal/domain/travel"

	"github.com/rs/zerolog/log"
)

// buildDepartureMap builds a map of member departure times from state changes.
// Uses BigQuery when available; falls back to a full sheet scan.
func (s *StatusV2Service) buildDepartureMap(ctx context.Context, spreadsheetID string, currentStateRecords []app.StateRecord) (map[string]time.Time, error) {
	departureMap := make(map[string]time.Time)

	// Collect IDs of currently-traveling members only
	var travelingIDs []string
	travelingRecords := make(map[string]app.StateRecord)
	for _, r := range currentStateRecords {
		if r.StatusState == "Traveling" {
			travelingIDs = append(travelingIDs, r.MemberID)
			travelingRecords[r.MemberID] = r
		}
	}
	if len(travelingIDs) == 0 {
		return departureMap, nil
	}

	if s.bigqueryClient != nil {
		times, err := s.bigqueryClient.QueryDepartureTimes(ctx, travelingIDs)
		if err != nil {
			log.Warn().Err(err).Msg("BigQuery departure query failed, falling back to sheet scan")
		} else {
			for memberID, t := range times {
				r := travelingRecords[memberID]
				memberKey := fmt.Sprintf("%s_%s", r.FactionID, memberID)
				departureMap[memberKey] = t
			}
			return departureMap, nil
		}
	}

	// Fall back: full sheet scan
	allStateRecords, err := s.ReadAllStateRecords(ctx, spreadsheetID)
	if err != nil {
		return departureMap, fmt.Errorf("failed to read state records: %w", err)
	}
	for memberID, currentRecord := range travelingRecords {
		memberKey := fmt.Sprintf("%s_%s", currentRecord.FactionID, memberID)
		currentParsedLocation := s.locationService.ParseLocation(currentRecord.StatusDescription)
		departureTime := s.findMostRecentTravelingTransition(allStateRecords, memberID, currentParsedLocation)
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
