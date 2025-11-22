package war

import (
	"torn_rw_stats/internal/app"
)

// AttackFetchDecision describes how to fetch attacks for a war
type AttackFetchDecision struct {
	UseFullMode     bool
	UseIncremental  bool
	ShouldFetch     bool
	Reason          string
	HasExistingData bool
	RecordCount     int
	LatestTimestamp int64
}

// DetermineAttackFetchMode decides whether to use full or incremental mode
// based on existing records in the sheets
func DetermineAttackFetchMode(existingRecordCount int, latestTimestamp int64) AttackFetchDecision {
	if existingRecordCount == 0 {
		return AttackFetchDecision{
			UseFullMode:     true,
			UseIncremental:  false,
			ShouldFetch:     true,
			Reason:          "No existing records - full population mode",
			HasExistingData: false,
			RecordCount:     0,
			LatestTimestamp: 0,
		}
	}

	return AttackFetchDecision{
		UseFullMode:     false,
		UseIncremental:  true,
		ShouldFetch:     true,
		Reason:          "Existing records found - incremental update mode",
		HasExistingData: true,
		RecordCount:     existingRecordCount,
		LatestTimestamp: latestTimestamp,
	}
}

// DetermineOurFactionID identifies which faction in the war is ours
// Returns 0 if our faction is not found in the war
func DetermineOurFactionID(war *app.War, knownFactionID int) int {
	for factionID := range war.Factions {
		if factionID == knownFactionID {
			return knownFactionID
		}
	}
	return 0
}

// ShouldProcessMember determines if a member should be included in processing
// based on faction membership
func ShouldProcessMember(memberID string, factionMembers map[string]app.FactionMember) bool {
	_, exists := factionMembers[memberID]
	return exists
}
