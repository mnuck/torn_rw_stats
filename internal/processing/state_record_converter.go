package processing

import (
	"strconv"
	"time"

	"torn_rw_stats/internal/app"
)

// StateRecordConverter handles conversion between API responses and StateRecords
type StateRecordConverter struct{}

// NewStateRecordConverter creates a new StateRecord converter
func NewStateRecordConverter() *StateRecordConverter {
	return &StateRecordConverter{}
}

// ConvertFromFactionBasic converts FactionBasicResponse to StateRecords
func (c *StateRecordConverter) ConvertFromFactionBasic(response *app.FactionBasicResponse, currentTime time.Time) []app.StateRecord {
	var records []app.StateRecord

	factionIDStr := strconv.Itoa(response.ID)
	factionName := response.Name

	for memberIDStr, member := range response.Members {
		record := c.convertMemberToStateRecord(memberIDStr, member, factionIDStr, factionName, currentTime)
		records = append(records, record)
	}

	return records
}

// ConvertFromFactionInfo converts FactionInfoResponse to StateRecords
func (c *StateRecordConverter) ConvertFromFactionInfo(response *app.FactionInfoResponse, currentTime time.Time) []app.StateRecord {
	var records []app.StateRecord

	factionIDStr := strconv.Itoa(response.ID)
	factionName := response.Name

	for memberIDStr, member := range response.Members {
		record := c.convertMemberToStateRecord(memberIDStr, member, factionIDStr, factionName, currentTime)
		records = append(records, record)
	}

	return records
}

// Note: ConvertFromWarsResponse is not implemented because /v2/faction/wars
// does not include member data. Member data during wars is obtained through
// separate faction API calls (/faction/{id}?selections=basic)

// convertMemberToStateRecord converts a FactionMember to a StateRecord
func (c *StateRecordConverter) convertMemberToStateRecord(memberIDStr string, member app.FactionMember, factionIDStr, factionName string, currentTime time.Time) app.StateRecord {
	// Convert StatusUntil from *int64 to time.Time - only if it's a meaningful timestamp
	var statusUntil time.Time
	if member.Status.Until != nil && *member.Status.Until > 0 {
		statusUntil = time.Unix(*member.Status.Until, 0).UTC()
	}

	return app.StateRecord{
		Timestamp:         currentTime,
		MemberID:          memberIDStr,
		MemberName:        member.Name,
		FactionID:         factionIDStr,
		FactionName:       factionName,
		LastActionStatus:  member.LastAction.Status,
		StatusDescription: member.Status.Description,
		StatusState:       member.Status.State,
		StatusUntil:       statusUntil,
		StatusTravelType:  member.Status.TravelType,
	}
}
