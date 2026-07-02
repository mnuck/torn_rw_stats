package services

import (
	"context"
	"fmt"
	"time"

	"log/slog"
	"torn_rw_stats/internal/domain"
	"torn_rw_stats/internal/domain/state"

	"torn_rw_stats/internal/application/ports"
)

// StateTrackingService handles the complete state tracking workflow, detecting
// and recording member state changes to BigQuery.
type StateTrackingService struct {
	tornClient     ports.TornClient
	bigqueryClient ports.BigQueryClient // nil = disabled
	converter      *state.StateRecordConverter
	comparator     *state.StateRecordComparator
}

// NewStateTrackingService creates a new state tracking service. bqClient may be nil to disable BigQuery.
func NewStateTrackingService(tornClient ports.TornClient, bqClient ports.BigQueryClient) *StateTrackingService {
	return &StateTrackingService{
		tornClient:     tornClient,
		bigqueryClient: bqClient,
		converter:      state.NewStateRecordConverter(),
		comparator:     state.NewStateRecordComparator(),
	}
}

// ProcessStateChanges executes the complete state tracking workflow
func (s *StateTrackingService) ProcessStateChanges(ctx context.Context, spreadsheetID string, factionIDs []int) error {
	currentTime := time.Now().UTC()

	slog.Info("Starting state change processing",
		"faction_count", len(factionIDs))

	// Step 1: Get current StateRecords for all factions
	currentStateRecords, err := s.getCurrentStateRecords(ctx, factionIDs, currentTime)
	if err != nil {
		return fmt.Errorf("failed to get current state records: %w", err)
	}

	slog.Debug("Retrieved current state records",
		"current_records", len(currentStateRecords))

	// Step 2: Read previous states from BigQuery
	var allPreviousStates []domain.StateRecord
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

	slog.Debug("Read previous state records",
		"previous_records", len(allPreviousStates))

	// Step 3: Create previous state collection for comparison
	previousStateRecords := s.comparator.CreatePreviousStateCollection(currentStateRecords, allPreviousStates)

	slog.Debug("Created previous states collection for comparison",
		"previous_for_comparison", len(previousStateRecords))

	// Step 4: Compare states and find changes
	updatedStateRecords := s.comparator.FindChangedStates(currentStateRecords, s.mapToSlice(previousStateRecords))

	// Step 5: Use domain function to determine action
	decision := state.DetermineStateChangeAction(currentStateRecords, s.mapToSlice(previousStateRecords), updatedStateRecords)

	slog.Info("Determined state change action",
		"changed_states", decision.ChangeCount,
		"should_write", decision.ShouldWriteChanges,
		"reason", decision.Reason)

	// Step 6: Write changed records to BigQuery
	if decision.ShouldWriteChanges {
		if err := s.addStateRecords(ctx, decision.RecordsToWrite); err != nil {
			return fmt.Errorf("failed to add state records: %w", err)
		}

		slog.Info("Successfully added state changes",
			"records_added", len(decision.RecordsToWrite))
	} else {
		slog.Info(decision.Reason)
	}

	return nil
}

// getCurrentStateRecords retrieves current state for all specified factions
func (s *StateTrackingService) getCurrentStateRecords(ctx context.Context, factionIDs []int, currentTime time.Time) ([]domain.StateRecord, error) {
	var allRecords []domain.StateRecord

	for _, factionID := range factionIDs {
		factionData, err := s.tornClient.GetFactionBasic(ctx, factionID)
		if err != nil {
			slog.Error("Failed to get faction data - skipping",
				"err", err,
				"faction_id", factionID)
			continue
		}

		records := s.converter.ConvertFromFactionBasic(factionData, currentTime)
		allRecords = append(allRecords, records...)

		slog.Debug("Retrieved state records for faction",
			"faction_id", factionID,
			"member_count", len(records))
	}

	return allRecords, nil
}

// mapToSlice converts a map of StateRecords to a slice
func (s *StateTrackingService) mapToSlice(recordMap map[string]domain.StateRecord) []domain.StateRecord {
	var slice []domain.StateRecord
	for _, record := range recordMap {
		slice = append(slice, record)
	}
	return slice
}

// addStateRecords streams state records to BigQuery.
func (s *StateTrackingService) addStateRecords(ctx context.Context, records []domain.StateRecord) error {
	if len(records) == 0 || s.bigqueryClient == nil {
		return nil
	}
	return s.bigqueryClient.InsertStateRecords(ctx, records)
}
