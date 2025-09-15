package processing

import (
	"regexp"
	"time"

	"torn_rw_stats/internal/app"
)

// StateRecordComparator handles comparison logic for StateRecords
type StateRecordComparator struct {
	hospitalRegex *regexp.Regexp
	jailRegex     *regexp.Regexp
}

// NewStateRecordComparator creates a new StateRecord comparator
func NewStateRecordComparator() *StateRecordComparator {
	// Compile hospital regex once for reuse (copied from existing logic)
	hospitalRegex := regexp.MustCompile(`(?i)^in\s+(a\s+[\w\s]+\s+)?hospital(\s+for\s+.*)?$`)

	// Compile jail regex to handle jail countdown variations
	jailRegex := regexp.MustCompile(`(?i)^in\s+jail\s+for\s+.*$`)

	return &StateRecordComparator{
		hospitalRegex: hospitalRegex,
		jailRegex:     jailRegex,
	}
}

// FindChangedStates compares current states with previous states and returns only changed states
func (c *StateRecordComparator) FindChangedStates(currentStates []app.StateRecord, previousStates []app.StateRecord) []app.StateRecord {
	// Create map of previous states by member ID for quick lookup
	previousByID := make(map[string]app.StateRecord)
	for _, prev := range previousStates {
		previousByID[prev.MemberID] = prev
	}

	var changedStates []app.StateRecord

	// Compare each current state with its previous state
	for _, current := range currentStates {
		previous, exists := previousByID[current.MemberID]

		// If no previous state exists, this is a new member - include them
		if !exists {
			changedStates = append(changedStates, current)
			continue
		}

		// Compare states
		if c.HasStateChanged(previous, current) {
			changedStates = append(changedStates, current)
		}
	}

	return changedStates
}

// HasStateChanged compares two StateRecords to determine if a meaningful change occurred
func (c *StateRecordComparator) HasStateChanged(previous, current app.StateRecord) bool {
	// Check comparable fields for changes
	if previous.MemberName != current.MemberName ||
		previous.FactionName != current.FactionName ||
		previous.FactionID != current.FactionID ||
		previous.LastActionStatus != current.LastActionStatus ||
		previous.StatusState != current.StatusState ||
		previous.StatusTravelType != current.StatusTravelType {
		return true
	}

	// Compare StatusDescription with hospital and jail normalization
	prevDesc := c.normalizeStatusDescription(previous.StatusDescription)
	currDesc := c.normalizeStatusDescription(current.StatusDescription)
	if prevDesc != currDesc {
		return true
	}

	// Compare StatusUntil - only consider changes if both times are meaningful
	if c.hasStatusUntilChanged(previous.StatusUntil, current.StatusUntil) {
		return true
	}

	return false
}

// GetLatestStateByMember finds the most recent StateRecord for each member from a collection
func (c *StateRecordComparator) GetLatestStateByMember(records []app.StateRecord) map[string]app.StateRecord {
	latestByMember := make(map[string]app.StateRecord)

	for _, record := range records {
		existing, exists := latestByMember[record.MemberID]

		// If no existing record or this record is newer, update
		if !exists || record.Timestamp.After(existing.Timestamp) {
			latestByMember[record.MemberID] = record
		}
	}

	return latestByMember
}

// CreatePreviousStateCollection creates a map of the most recent StateRecord for each current member
func (c *StateRecordComparator) CreatePreviousStateCollection(currentStates []app.StateRecord, allPreviousStates []app.StateRecord) map[string]app.StateRecord {
	// Get latest state for each member from all previous records
	latestByMember := c.GetLatestStateByMember(allPreviousStates)

	// Create result map with only members that exist in current states
	result := make(map[string]app.StateRecord)
	for _, current := range currentStates {
		if previous, exists := latestByMember[current.MemberID]; exists {
			result[current.MemberID] = previous
		}
	}

	return result
}

// normalizeStatusDescription removes countdown from hospital and jail descriptions for comparison
// Prevents noise from countdown timer changes in Changed States tracking
func (c *StateRecordComparator) normalizeStatusDescription(description string) string {
	// Hospital normalization (copied from existing logic in state_change_service.go)
	if c.hospitalRegex.MatchString(description) {
		return "In hospital"
	}

	// Jail normalization - handles variations like "In jail for 4 hrs 14 mins" vs "In jail for 4 hours 12 mins"
	if c.jailRegex.MatchString(description) {
		return "In jail"
	}

	return description
}

// hasStatusUntilChanged determines if StatusUntil represents a meaningful change
func (c *StateRecordComparator) hasStatusUntilChanged(previous, current time.Time) bool {
	// If both are zero time, no change
	if previous.IsZero() && current.IsZero() {
		return false
	}

	// If one is zero and the other isn't, that's a change
	if previous.IsZero() != current.IsZero() {
		return true
	}

	// If both have values, compare them
	// Consider a change significant if it's more than a minute difference
	// (to avoid noise from small API timing differences)
	diff := current.Sub(previous)
	if diff < 0 {
		diff = -diff
	}

	return diff > time.Minute
}