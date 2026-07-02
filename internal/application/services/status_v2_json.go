package services

import (
	"time"

	"torn_rw_stats/internal/domain"
	"torn_rw_stats/internal/domain/status"
)

// ConvertToJSON converts StatusV2Records to the JSON export format
func (s *StatusV2Service) ConvertToJSON(records []domain.StatusV2Record, factionName string, currentTime time.Time, updateInterval time.Duration) domain.StatusV2JSON {
	// Use domain function for all JSON conversion logic
	locations := status.GroupRecordsByLocation(records)

	return domain.StatusV2JSON{
		Faction:   factionName,
		Updated:   currentTime.Format(time.RFC3339),
		Interval:  int(updateInterval.Seconds()),
		Locations: locations,
	}
}
