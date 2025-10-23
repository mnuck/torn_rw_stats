package state

import (
	"fmt"

	"torn_rw_stats/internal/app"
)

// ChangeDecision describes what actions should be taken based on state changes
type ChangeDecision struct {
	ShouldWriteChanges bool
	RecordsToWrite     []app.StateRecord
	ChangeCount        int
	Reason             string
}

// DetermineStateChangeAction decides what to do with detected state changes
// Returns a decision object describing whether to write and what to write
func DetermineStateChangeAction(
	currentStates []app.StateRecord,
	previousStates []app.StateRecord,
	changedStates []app.StateRecord,
) ChangeDecision {
	changeCount := len(changedStates)

	if changeCount == 0 {
		return ChangeDecision{
			ShouldWriteChanges: false,
			RecordsToWrite:     nil,
			ChangeCount:        0,
			Reason:             "No state changes detected",
		}
	}

	return ChangeDecision{
		ShouldWriteChanges: true,
		RecordsToWrite:     changedStates,
		ChangeCount:        changeCount,
		Reason:             fmt.Sprintf("Found %d state changes requiring write", changeCount),
	}
}

// ShouldUpdateStateRecord determines if a specific state change should be persisted
// This can be used for filtering changes based on business rules
func ShouldUpdateStateRecord(change app.StateRecord) bool {
	// Currently all changes are persisted, but this function provides
	// a hook for future filtering logic if needed
	return true
}
