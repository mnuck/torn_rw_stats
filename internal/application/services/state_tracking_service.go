package services

import (
	"context"
	"fmt"
	"time"

	"torn_rw_stats/internal/app"
	"torn_rw_stats/internal/domain/state"
	"torn_rw_stats/internal/processing"

	"github.com/rs/zerolog/log"
)

// StateTrackingService handles the complete state tracking workflow, detecting
// and recording member state changes to BigQuery.
type StateTrackingService struct {
	tornClient     processing.TornClientInterface
	bigqueryClient processing.BigQueryClientInterface // nil = disabled
	converter      *processing.StateRecordConverter
	comparator     *processing.StateRecordComparator
}

// NewStateTrackingService creates a new state tracking service. bqClient may be nil to disable BigQuery.
func NewStateTrackingService(tornClient processing.TornClientInterface, bqClient processing.BigQueryClientInterface) *StateTrackingService {
	return &StateTrackingService{
		tornClient:     tornClient,
		bigqueryClient: bqClient,
		converter:      processing.NewStateRecordConverter(),
		comparator:     processing.NewStateRecordComparator(),
	}
}

// ProcessStateChanges executes the complete state tracking workflow
func (s *StateTrackingService) ProcessStateChanges(ctx context.Context, spreadsheetID string, factionIDs []int) error {
	currentTime := time.Now().UTC()

	log.Info().
		Int("faction_count", len(factionIDs)).
		Msg("Starting state change processing")

	// Step 1: Get current StateRecords for all factions
	currentStateRecords, err := s.getCurrentStateRecords(ctx, factionIDs, currentTime)
	if err != nil {
		return fmt.Errorf("failed to get current state records: %w", err)
	}

	log.Debug().
		Int("current_records", len(currentStateRecords)).
		Msg("Retrieved current state records")

	// Step 2: Read previous states from BigQuery
	var allPreviousStates []app.StateRecord
	if s.bigqueryClient != nil {
		memberIDs := make([]string, 0, len(currentStateRecords))
		for _, r := range currentStateRecords {
			memberIDs = append(memberIDs, r.MemberID)
		}
		allPreviousStates, err = s.bigqueryClient.QueryLatestStatePerMember(ctx, memberIDs)
		if err != nil {
			return fmt.Errorf("failed to query previous states from BigQuery: %w", err)
		}
	}

	log.Debug().
		Int("previous_records", len(allPreviousStates)).
		Msg("Read previous state records")

	// Step 3: Create previous state collection for comparison
	previousStateRecords := s.comparator.CreatePreviousStateCollection(currentStateRecords, allPreviousStates)

	log.Debug().
		Int("previous_for_comparison", len(previousStateRecords)).
		Msg("Created previous states collection for comparison")

	// Step 4: Compare states and find changes
	updatedStateRecords := s.comparator.FindChangedStates(currentStateRecords, s.mapToSlice(previousStateRecords))

	// Step 5: Use domain function to determine action
	decision := state.DetermineStateChangeAction(currentStateRecords, s.mapToSlice(previousStateRecords), updatedStateRecords)

	log.Info().
		Int("changed_states", decision.ChangeCount).
		Bool("should_write", decision.ShouldWriteChanges).
		Str("reason", decision.Reason).
		Msg("Determined state change action")

	// Step 6: Write changed records to BigQuery
	if decision.ShouldWriteChanges {
		if err := s.addStateRecords(ctx, decision.RecordsToWrite); err != nil {
			return fmt.Errorf("failed to add state records: %w", err)
		}

		log.Info().
			Int("records_added", len(decision.RecordsToWrite)).
			Msg("Successfully added state changes")
	} else {
		log.Info().Msg(decision.Reason)
	}

	return nil
}

// getCurrentStateRecords retrieves current state for all specified factions
func (s *StateTrackingService) getCurrentStateRecords(ctx context.Context, factionIDs []int, currentTime time.Time) ([]app.StateRecord, error) {
	var allRecords []app.StateRecord

	for _, factionID := range factionIDs {
		factionData, err := s.tornClient.GetFactionBasic(ctx, factionID)
		if err != nil {
			log.Error().
				Err(err).
				Int("faction_id", factionID).
				Msg("Failed to get faction data - skipping")
			continue
		}

		records := s.converter.ConvertFromFactionBasic(factionData, currentTime)
		allRecords = append(allRecords, records...)

		log.Debug().
			Int("faction_id", factionID).
			Int("member_count", len(records)).
			Msg("Retrieved state records for faction")
	}

	return allRecords, nil
}

// mapToSlice converts a map of StateRecords to a slice
func (s *StateTrackingService) mapToSlice(recordMap map[string]app.StateRecord) []app.StateRecord {
	var slice []app.StateRecord
	for _, record := range recordMap {
		slice = append(slice, record)
	}
	return slice
}

// addStateRecords streams state records to BigQuery.
func (s *StateTrackingService) addStateRecords(ctx context.Context, records []app.StateRecord) error {
	if len(records) == 0 || s.bigqueryClient == nil {
		return nil
	}
	return s.bigqueryClient.InsertStateRecords(ctx, records)
}
