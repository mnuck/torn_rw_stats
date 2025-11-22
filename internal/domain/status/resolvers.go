package status

import (
	"fmt"

	"torn_rw_stats/internal/app"
)

// GetExistingRecord finds existing data for a member using both ID and name keys
// Returns nil if no existing record found
func GetExistingRecord(
	factionID string,
	memberID string,
	memberName string,
	existingData map[string]app.StatusV2Record,
) *app.StatusV2Record {
	// Try member ID key first
	memberKey := fmt.Sprintf("%s_%s", factionID, memberID)
	if existing, hasExisting := existingData[memberKey]; hasExisting {
		return &existing
	}

	// Fallback to name key for compatibility
	nameKey := fmt.Sprintf("%s_%s", factionID, memberName)
	if existing, hasExisting := existingData[nameKey]; hasExisting {
		return &existing
	}

	return nil
}

// ResolveLevel determines the member's level from faction data or existing records
// Returns 0 if level cannot be determined
func ResolveLevel(
	memberID string,
	factionMembers map[string]app.FactionMember,
	existing *app.StatusV2Record,
) int {
	// Try to get level from current faction data first
	if member, exists := factionMembers[memberID]; exists {
		return member.Level
	}

	// Fall back to existing data if available
	if existing != nil {
		return existing.Level
	}

	return 0
}

// ShouldPreserveTravelData determines if existing travel data should be preserved
// This happens when the player is still traveling and we have existing departure/arrival times
func ShouldPreserveTravelData(
	statusState string,
	existing *app.StatusV2Record,
) bool {
	if statusState != "Traveling" {
		return false
	}

	if existing == nil {
		return false
	}

	// Preserve if we have existing departure or arrival data
	return existing.Departure != "" || existing.Arrival != ""
}
