package services

import (
	"strings"
	"time"

	"torn_rw_stats/internal/app"
)

// ConvertToJSON converts StatusV2Records to the JSON export format
func (s *StatusV2Service) ConvertToJSON(records []app.StatusV2Record, factionName string, currentTime time.Time, updateInterval time.Duration) app.StatusV2JSON {
	locations := s.groupRecordsByLocation(records)

	return app.StatusV2JSON{
		Faction:   factionName,
		Updated:   currentTime.Format(time.RFC3339),
		Interval:  int(updateInterval.Seconds()),
		Locations: locations,
	}
}

// groupRecordsByLocation organizes records by location and filters empty locations
func (s *StatusV2Service) groupRecordsByLocation(records []app.StatusV2Record) map[string]app.LocationData {
	locations := make(map[string]app.LocationData)

	for _, record := range records {
		if record.Location == "" {
			continue
		}

		member := s.createJSONMember(record)
		s.addMemberToLocation(locations, record.Location, member, s.isTraveling(record))
	}

	// Filter out empty locations
	return s.filterEmptyLocations(locations)
}

// createJSONMember creates a JSONMember from a StatusV2Record
func (s *StatusV2Service) createJSONMember(record app.StatusV2Record) app.JSONMember {
	member := app.JSONMember{
		Name:     record.Name,
		MemberID: record.MemberID,
		Level:    record.Level,
		State:    record.State,
	}

	if !record.Until.IsZero() {
		member.Until = record.Until.Format(time.RFC3339)
	}

	if s.isTraveling(record) {
		s.addTravelingFields(&member, record)
	} else {
		s.addLocatedFields(&member, record)
	}

	return member
}

// isTraveling determines if a member is currently traveling
func (s *StatusV2Service) isTraveling(record app.StatusV2Record) bool {
	return strings.Contains(strings.ToLower(record.Status), "traveling")
}

// addTravelingFields adds travel-specific fields to a JSON member
func (s *StatusV2Service) addTravelingFields(member *app.JSONMember, record app.StatusV2Record) {
	if record.Countdown != "" && record.Countdown != "00:00:00" {
		member.Countdown = strings.TrimPrefix(record.Countdown, "'")
	}
	if record.Arrival != "" {
		member.Arrival = record.Arrival
	}
	if record.BusinessArrival != "" {
		member.BusinessArrival = record.BusinessArrival
	}
}

// addLocatedFields adds location-specific fields to a JSON member
func (s *StatusV2Service) addLocatedFields(member *app.JSONMember, record app.StatusV2Record) {
	if record.Status != "" && record.Status != "Okay" {
		member.Status = record.Status
		if record.Countdown != "" && record.Countdown != "00:00:00" {
			member.Countdown = strings.TrimPrefix(record.Countdown, "'")
		}
	}
}

// addMemberToLocation adds a member to the appropriate location array
func (s *StatusV2Service) addMemberToLocation(locations map[string]app.LocationData, location string, member app.JSONMember, isTraveling bool) {
	if _, exists := locations[location]; !exists {
		locations[location] = app.LocationData{
			Traveling: []app.JSONMember{},
			LocatedIn: []app.JSONMember{},
		}
	}

	locationData := locations[location]
	if isTraveling {
		locationData.Traveling = append(locationData.Traveling, member)
	} else {
		locationData.LocatedIn = append(locationData.LocatedIn, member)
	}
	locations[location] = locationData
}

// filterEmptyLocations removes locations with no members
func (s *StatusV2Service) filterEmptyLocations(locations map[string]app.LocationData) map[string]app.LocationData {
	filteredLocations := make(map[string]app.LocationData)
	for location, data := range locations {
		if len(data.Traveling) > 0 || len(data.LocatedIn) > 0 {
			filteredLocations[location] = data
		}
	}
	return filteredLocations
}
