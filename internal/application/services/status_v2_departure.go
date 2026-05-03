package services

import (
	"context"
	"fmt"
	"time"

	"torn_rw_stats/internal/app"

	"github.com/rs/zerolog/log"
)

// buildDepartureMap builds a map of member departure times from BigQuery state changes.
func (s *StatusV2Service) buildDepartureMap(ctx context.Context, _ string, currentStateRecords []app.StateRecord) (map[string]time.Time, error) {
	departureMap := make(map[string]time.Time)

	var travelingIDs []string
	travelingRecords := make(map[string]app.StateRecord)
	for _, r := range currentStateRecords {
		if r.StatusState == "Traveling" {
			travelingIDs = append(travelingIDs, r.MemberID)
			travelingRecords[r.MemberID] = r
		}
	}
	if len(travelingIDs) == 0 || s.bigqueryClient == nil {
		return departureMap, nil
	}

	times, err := s.bigqueryClient.QueryDepartureTimes(ctx, travelingIDs)
	if err != nil {
		log.Warn().Err(err).Msg("BigQuery departure query failed")
		return departureMap, fmt.Errorf("failed to query departure times: %w", err)
	}

	for memberID, t := range times {
		r := travelingRecords[memberID]
		memberKey := fmt.Sprintf("%s_%s", r.FactionID, memberID)
		departureMap[memberKey] = t
	}
	return departureMap, nil
}
