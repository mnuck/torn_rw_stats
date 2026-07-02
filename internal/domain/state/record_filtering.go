package state

import (
	"sort"

	"torn_rw_stats/internal/domain"
)

// FilterRecordsByMember filters state records to only include those for a specific member.
//
// Pure function: No I/O operations, fully testable with direct inputs.
func FilterRecordsByMember(records []domain.StateRecord, memberID string) []domain.StateRecord {
	var filtered []domain.StateRecord
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
func SortRecordsByTimestamp(records []domain.StateRecord) []domain.StateRecord {
	sorted := make([]domain.StateRecord, len(records))
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
func GetMemberRecordsChronologically(allRecords []domain.StateRecord, memberID string) []domain.StateRecord {
	filtered := FilterRecordsByMember(allRecords, memberID)
	return SortRecordsByTimestamp(filtered)
}
