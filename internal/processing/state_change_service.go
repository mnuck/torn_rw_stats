package processing

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"time"

	"torn_rw_stats/internal/app"

	"github.com/rs/zerolog/log"
)

// StateChangeDetectionService handles faction member state tracking and change detection
type StateChangeDetectionService struct {
	sheetsClient SheetsClientInterface
}

// NewStateChangeDetectionService creates a new state change detection service
func NewStateChangeDetectionService(sheetsClient SheetsClientInterface) *StateChangeDetectionService {
	return &StateChangeDetectionService{
		sheetsClient: sheetsClient,
	}
}

// NormalizeHospitalDescription removes countdown from hospital descriptions for comparison
func (scds *StateChangeDetectionService) NormalizeHospitalDescription(description string) string {
	// Match "In hospital for X hrs Y mins" and "In a [Country/Countries] hospital for X mins" patterns
	// Handles single words (British, Mexican) and multi-word countries (South African)
	hospitalRegex := regexp.MustCompile(`(?i)^in\s+(a\s+[\w\s]+\s+)?hospital(\s+for\s+.*)?$`)
	if hospitalRegex.MatchString(description) {
		return "In hospital"
	}
	return description
}

// HasStatusChanged compares two member states to detect changes, ignoring hospital countdown changes
func (scds *StateChangeDetectionService) HasStatusChanged(oldMember, newMember app.FactionMember) bool {
	// Check LastAction.Status changes
	if oldMember.LastAction.Status != newMember.LastAction.Status {
		return true
	}

	// Check Status fields, normalizing hospital descriptions
	oldDesc := scds.NormalizeHospitalDescription(oldMember.Status.Description)
	newDesc := scds.NormalizeHospitalDescription(newMember.Status.Description)

	if oldDesc != newDesc ||
		oldMember.Status.State != newMember.Status.State ||
		oldMember.Status.Color != newMember.Status.Color ||
		oldMember.Status.Details != newMember.Status.Details ||
		oldMember.Status.TravelType != newMember.Status.TravelType ||
		oldMember.Status.PlaneImageType != newMember.Status.PlaneImageType {
		return true
	}

	// Check Until field (but be careful with nil pointers)
	if (oldMember.Status.Until == nil) != (newMember.Status.Until == nil) {
		return true
	}
	if oldMember.Status.Until != nil && newMember.Status.Until != nil {
		if *oldMember.Status.Until != *newMember.Status.Until {
			return true
		}
	}

	return false
}

// ProcessStateChanges detects and records state changes for faction members
func (scds *StateChangeDetectionService) ProcessStateChanges(ctx context.Context, factionID int, factionName string, oldMembers, newMembers map[string]app.FactionMember, spreadsheetID string) error {
	// Ensure state change records sheet exists
	sheetName, err := scds.sheetsClient.EnsureStateChangeRecordsSheet(ctx, spreadsheetID, factionID)
	if err != nil {
		return fmt.Errorf("failed to ensure state change records sheet: %w", err)
	}

	currentTime := time.Now().UTC()
	stateChanges := 0

	// Check each current member against previous state
	for userIDStr, newMember := range newMembers {
		userID, err := strconv.Atoi(userIDStr)
		if err != nil {
			log.Warn().
				Str("user_id_str", userIDStr).
				Str("member_name", newMember.Name).
				Msg("Failed to parse user ID for state change tracking")
			continue
		}

		// Check if we have previous state for this member
		if oldMember, exists := oldMembers[userIDStr]; exists {
			// Compare states
			if scds.HasStatusChanged(oldMember, newMember) {
				// Create state change record
				record := scds.createStateChangeRecord(currentTime, newMember, userID, factionName, factionID)

				// Add record to sheet
				if err := scds.sheetsClient.AddStateChangeRecord(ctx, spreadsheetID, sheetName, record); err != nil {
					log.Error().
						Err(err).
						Str("member_name", newMember.Name).
						Int("member_id", userID).
						Msg("Failed to add state change record")
				} else {
					stateChanges++
					log.Debug().
						Str("member_name", newMember.Name).
						Int("member_id", userID).
						Str("old_state", oldMember.Status.State).
						Str("new_state", newMember.Status.State).
						Str("old_last_action", oldMember.LastAction.Status).
						Str("new_last_action", newMember.LastAction.Status).
						Msg("Recorded state change")
				}
			}
		}
		// Note: We don't record first-time appearances as state changes
	}

	log.Info().
		Int("faction_id", factionID).
		Str("faction_name", factionName).
		Int("state_changes", stateChanges).
		Str("sheet_name", sheetName).
		Msg("Processed member state changes")

	return nil
}

// createStateChangeRecord creates a StateChangeRecord from member data
func (scds *StateChangeDetectionService) createStateChangeRecord(currentTime time.Time, member app.FactionMember, userID int, factionName string, factionID int) app.StateChangeRecord {
	return app.StateChangeRecord{
		Timestamp:         currentTime,
		MemberName:        member.Name,
		MemberID:          userID,
		FactionName:       factionName,
		FactionID:         factionID,
		LastActionStatus:  member.LastAction.Status,
		StatusDescription: member.Status.Description,
		StatusState:       member.Status.State,
		StatusColor:       member.Status.Color,
		StatusDetails:     member.Status.Details,
		StatusUntil: func() string {
			if member.Status.Until != nil {
				return strconv.FormatInt(*member.Status.Until, 10)
			}
			return ""
		}(),
		StatusTravelType:     member.Status.TravelType,
		StatusPlaneImageType: member.Status.PlaneImageType,
	}
}
