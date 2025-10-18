package services

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"torn_rw_stats/internal/app"
	"torn_rw_stats/internal/deployment"
	"torn_rw_stats/internal/processing"

	"github.com/rs/zerolog/log"
)

// StatusV2Processor handles Status v2 sheet processing, converting faction member
// states to status sheets and JSON exports for external consumption.
type StatusV2Processor struct {
	tornClient   processing.TornClientInterface
	sheetsClient processing.SheetsClientInterface
	service      *StatusV2Service
	ourFactionID int // cached faction ID, fetched via API
	deployer     *deployment.SSHDeployer
}

// NewStatusV2Processor creates a new Status v2 processor
func NewStatusV2Processor(tornClient processing.TornClientInterface, sheetsClient processing.SheetsClientInterface, deployURL string) *StatusV2Processor {
	var deployer *deployment.SSHDeployer
	if deployURL != "" {
		deployer = deployment.NewSSHDeployer(deployURL)
	}

	return &StatusV2Processor{
		tornClient:   tornClient,
		sheetsClient: sheetsClient,
		service:      NewStatusV2Service(sheetsClient),
		ourFactionID: 0, // will be fetched via API when needed
		deployer:     deployer,
	}
}

// ensureOurFactionID fetches and caches our faction ID if not already set
func (p *StatusV2Processor) ensureOurFactionID(ctx context.Context) error {
	if p.ourFactionID == 0 {
		log.Debug().Msg("StatusV2Processor: Fetching our faction ID from API")

		factionInfo, err := p.tornClient.GetOwnFaction(ctx)
		if err != nil {
			return fmt.Errorf("failed to get own faction info: %w", err)
		}

		p.ourFactionID = factionInfo.ID
		log.Info().
			Int("faction_id", p.ourFactionID).
			Str("faction_name", factionInfo.Name).
			Str("faction_tag", factionInfo.Tag).
			Msg("StatusV2Processor: Detected our faction ID")
	}
	return nil
}

// ProcessStatusV2ForFactions processes Status v2 sheets for multiple factions
func (p *StatusV2Processor) ProcessStatusV2ForFactions(ctx context.Context, spreadsheetID string, factionIDs []int, updateInterval time.Duration) error {
	// Ensure our faction ID is loaded for proper filtering
	if err := p.ensureOurFactionID(ctx); err != nil {
		log.Error().Err(err).Msg("Failed to fetch our faction ID - continuing but filtering may be incorrect")
	}

	log.Info().
		Int("faction_count", len(factionIDs)).
		Int("our_faction_id", p.ourFactionID).
		Msg("Processing Status v2 for factions")

	for _, factionID := range factionIDs {
		if err := p.ProcessStatusV2ForFaction(ctx, spreadsheetID, factionID, updateInterval); err != nil {
			log.Error().
				Err(err).
				Int("faction_id", factionID).
				Msg("Failed to process Status v2 for faction - continuing with others")
			continue
		}

		log.Debug().
			Int("faction_id", factionID).
			Msg("Successfully processed Status v2 for faction")
	}

	return nil
}

// ProcessStatusV2ForFaction processes Status v2 sheet for a single faction
func (p *StatusV2Processor) ProcessStatusV2ForFaction(ctx context.Context, spreadsheetID string, factionID int, updateInterval time.Duration) error {
	// Step 1: Ensure Status v2 sheet exists
	sheetName, err := p.sheetsClient.EnsureStatusV2Sheet(ctx, spreadsheetID, factionID)
	if err != nil {
		return fmt.Errorf("failed to ensure Status v2 sheet: %w", err)
	}

	// Step 2: Get current faction data
	factionData, err := p.tornClient.GetFactionBasic(ctx, factionID)
	if err != nil {
		return fmt.Errorf("failed to get faction data: %w", err)
	}

	// Step 3: Read all state records from Changed States sheet to get current state
	allStateRecords, err := p.service.ReadAllStateRecords(ctx, spreadsheetID)
	if err != nil {
		log.Error().
			Err(err).
			Int("faction_id", factionID).
			Msg("Failed to read state records")
		return fmt.Errorf("failed to read state records: %w", err)
	}

	log.Info().
		Int("faction_id", factionID).
		Int("total_state_records", len(allStateRecords)).
		Msg("Successfully read all state records")

	// Step 4: Find current state records for this faction
	currentStateRecords := p.filterStateRecordsForFaction(allStateRecords, factionID)

	log.Info().
		Int("faction_id", factionID).
		Int("filtered_state_records", len(currentStateRecords)).
		Msg("Filtered state records for faction")

	// Step 5: Convert to Status v2 records
	statusV2Records, err := p.service.ConvertStateRecordsToStatusV2(
		ctx,
		spreadsheetID,
		currentStateRecords,
		factionData.Members,
		factionID,
	)
	if err != nil {
		return fmt.Errorf("failed to convert state records to Status v2: %w", err)
	}

	log.Info().
		Int("faction_id", factionID).
		Int("status_v2_records", len(statusV2Records)).
		Msg("Converted state records to Status v2 records")

	// Step 6: Update the Status v2 sheet
	log.Info().
		Int("faction_id", factionID).
		Str("sheet_name", sheetName).
		Int("records_to_write", len(statusV2Records)).
		Msg("About to update Status v2 sheet")

	if len(statusV2Records) == 0 {
		log.Warn().
			Int("faction_id", factionID).
			Str("sheet_name", sheetName).
			Msg("No Status v2 records to write - sheet will remain empty")
		return nil
	}

	if err := p.sheetsClient.UpdateStatusV2(ctx, spreadsheetID, sheetName, statusV2Records); err != nil {
		return fmt.Errorf("failed to update Status v2 sheet: %w", err)
	}

	log.Info().
		Int("faction_id", factionID).
		Int("records_count", len(statusV2Records)).
		Str("sheet_name", sheetName).
		Int("state_records_found", len(currentStateRecords)).
		Int("faction_members", len(factionData.Members)).
		Msg("Successfully updated Status v2 sheet")

	// Step 7: Export JSON alongside sheet update (only for opposing factions)
	if factionID != p.ourFactionID {
		if err := p.exportAndDeployJSON(statusV2Records, factionData.Name, factionID, updateInterval); err != nil {
			log.Warn().
				Err(err).
				Int("faction_id", factionID).
				Msg("Failed to export/deploy Status v2 JSON - continuing with processing")
		}
	} else {
		log.Debug().
			Int("faction_id", factionID).
			Msg("Skipping JSON export for our own faction")
	}

	return nil
}

// filterStateRecordsForFaction filters state records to only include current records for the specified faction
func (p *StatusV2Processor) filterStateRecordsForFaction(allStateRecords []app.StateRecord, factionID int) []app.StateRecord {
	factionIDStr := fmt.Sprintf("%d", factionID)

	log.Debug().
		Int("faction_id", factionID).
		Str("faction_id_str", factionIDStr).
		Int("total_records_to_filter", len(allStateRecords)).
		Msg("Starting faction filtering")

	// Group by member ID and find the most recent record for each member
	memberLatest := make(map[string]app.StateRecord)
	matchingRecords := 0

	// Debug: Show what we're actually reading from each column
	if len(allStateRecords) > 0 {
		firstRecord := allStateRecords[0]
		log.Info().
			Str("member_id", firstRecord.MemberID).
			Str("member_name", firstRecord.MemberName).
			Str("faction_id", firstRecord.FactionID).
			Str("faction_name", firstRecord.FactionName).
			Str("last_action_status", firstRecord.LastActionStatus).
			Str("status_description", firstRecord.StatusDescription).
			Str("status_state", firstRecord.StatusState).
			Str("status_travel_type", firstRecord.StatusTravelType).
			Msg("DEBUG: First record parsed values")
	}

	for _, record := range allStateRecords {
		if record.FactionID != factionIDStr {
			continue
		}
		matchingRecords++

		existing, exists := memberLatest[record.MemberID]
		if !exists || record.Timestamp.After(existing.Timestamp) {
			memberLatest[record.MemberID] = record
		}
	}

	log.Info().
		Int("faction_id", factionID).
		Str("faction_id_str", factionIDStr).
		Int("total_records_checked", len(allStateRecords)).
		Int("matching_faction_records", matchingRecords).
		Int("unique_members", len(memberLatest)).
		Msg("Faction filtering progress")

	// Convert map back to slice
	var currentRecords []app.StateRecord
	for _, record := range memberLatest {
		currentRecords = append(currentRecords, record)
	}

	log.Debug().
		Int("faction_id", factionID).
		Int("final_current_records", len(currentRecords)).
		Msg("Completed faction filtering")

	return currentRecords
}

// exportAndDeployJSON converts StatusV2Records to JSON format, writes to file, and deploys it
func (p *StatusV2Processor) exportAndDeployJSON(records []app.StatusV2Record, factionName string, factionID int, updateInterval time.Duration) error {
	currentTime := time.Now()

	// Convert to JSON format using the service
	jsonData := p.service.ConvertToJSON(records, factionName, currentTime, updateInterval)

	// Marshal to JSON bytes
	jsonBytes, err := json.MarshalIndent(jsonData, "", "    ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	// Create filename
	filename := fmt.Sprintf("status_v2_%d.json", factionID)

	// Write to local file
	if err := os.WriteFile(filename, jsonBytes, 0644); err != nil {
		return fmt.Errorf("failed to write JSON file: %w", err)
	}

	log.Info().
		Int("faction_id", factionID).
		Str("filename", filename).
		Int("locations_count", len(jsonData.Locations)).
		Msg("Successfully exported Status v2 JSON")

	// Deploy to remote server if deployer is configured
	if p.deployer != nil {
		// Use a fixed filename for the remote deployment
		remoteFilename := "status.json"

		if err := p.deployer.DeployFile(filename, remoteFilename); err != nil {
			return fmt.Errorf("failed to deploy JSON file: %w", err)
		}

		log.Info().
			Int("faction_id", factionID).
			Str("local_file", filename).
			Str("remote_file", remoteFilename).
			Msg("Successfully deployed Status v2 JSON")
	} else {
		log.Debug().
			Int("faction_id", factionID).
			Msg("No deployer configured - skipping remote deployment")
	}

	return nil
}
