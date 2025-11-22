package status

import (
	"strings"
	"time"

	"torn_rw_stats/internal/app"
)

// GroupRecordsByLocation organizes records by location and filters empty locations.
// Returns a map of location names to LocationData containing traveling and located members.
//
// Pure function: No I/O operations, fully testable with direct inputs.
func GroupRecordsByLocation(records []app.StatusV2Record) map[string]app.LocationData {
	locations := make(map[string]app.LocationData)

	for _, record := range records {
		if record.Location == "" {
			continue
		}

		member := ConvertToJSONMember(record)
		AddMemberToLocationData(locations, record.Location, member, IsTraveling(record))
	}

	return FilterEmptyLocations(locations)
}

// ConvertToJSONMember creates a JSONMember from a StatusV2Record with appropriate fields
// based on travel status and member state.
//
// Pure function: No I/O operations, fully testable with direct inputs.
func ConvertToJSONMember(record app.StatusV2Record) app.JSONMember {
	member := app.JSONMember{
		Name:     record.Name,
		MemberID: record.MemberID,
		Level:    record.Level,
		State:    record.State,
	}

	if !record.Until.IsZero() {
		member.Until = record.Until.Format(time.RFC3339)
	}

	if IsTraveling(record) {
		PopulateTravelingFields(&member, record)
	} else {
		PopulateLocatedFields(&member, record)
	}

	return member
}

// IsTraveling determines if a member is currently traveling based on their status.
//
// Pure function: No I/O operations, fully testable with direct inputs.
func IsTraveling(record app.StatusV2Record) bool {
	return strings.Contains(strings.ToLower(record.Status), "traveling")
}

// PopulateTravelingFields adds travel-specific fields to a JSON member,
// including countdown, arrival time, and business class arrival.
//
// Pure function: No I/O operations, fully testable with direct inputs.
func PopulateTravelingFields(member *app.JSONMember, record app.StatusV2Record) {
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

// PopulateLocatedFields adds location-specific fields to a JSON member,
// filtering out "Okay" status and including countdown for non-Okay statuses.
//
// Pure function: No I/O operations, fully testable with direct inputs.
func PopulateLocatedFields(member *app.JSONMember, record app.StatusV2Record) {
	if record.Status != "" && record.Status != "Okay" {
		member.Status = record.Status
		if record.Countdown != "" && record.Countdown != "00:00:00" {
			member.Countdown = strings.TrimPrefix(record.Countdown, "'")
		}
	}
}

// AddMemberToLocationData adds a member to the appropriate array (traveling or located)
// within the location data structure.
//
// Pure function: No I/O operations, fully testable with direct inputs.
func AddMemberToLocationData(locations map[string]app.LocationData, location string, member app.JSONMember, isTraveling bool) {
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

// FilterEmptyLocations removes locations with no members (neither traveling nor located).
//
// Pure function: No I/O operations, fully testable with direct inputs.
func FilterEmptyLocations(locations map[string]app.LocationData) map[string]app.LocationData {
	filteredLocations := make(map[string]app.LocationData)
	for location, data := range locations {
		if len(data.Traveling) > 0 || len(data.LocatedIn) > 0 {
			filteredLocations[location] = data
		}
	}
	return filteredLocations
}
