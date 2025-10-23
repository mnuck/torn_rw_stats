package state

import (
	"sort"

	"torn_rw_stats/internal/app"
)

// FilterRecordsByMember filters state records to only include those for a specific member.
//
// Pure function: No I/O operations, fully testable with direct inputs.
func FilterRecordsByMember(records []app.StateRecord, memberID string) []app.StateRecord {
	var filtered []app.StateRecord
	for _, record := range records {
		if record.MemberID == memberID {
			filtered = append(filtered, record)
		}
	}
	return filtered
}

// SortRecordsByTimestamp sorts state records chronologically by timestamp.
//
// Pure function: No I/O operations, fully testable with direct inputs.
func SortRecordsByTimestamp(records []app.StateRecord) []app.StateRecord {
	sorted := make([]app.StateRecord, len(records))
	copy(sorted, records)

	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Timestamp.Before(sorted[j].Timestamp)
	})

	return sorted
}

// GetMemberRecordsChronologically filters records for a specific member and sorts them chronologically.
// This is a convenience function combining filtering and sorting operations.
//
// Pure function: No I/O operations, fully testable with direct inputs.
func GetMemberRecordsChronologically(allRecords []app.StateRecord, memberID string) []app.StateRecord {
	filtered := FilterRecordsByMember(allRecords, memberID)
	return SortRecordsByTimestamp(filtered)
}
