package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"time"

	"log/slog"
	"torn_rw_stats/internal/deployment"
	"torn_rw_stats/internal/domain"

	"torn_rw_stats/internal/processing"
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
func NewStatusV2Processor(tornClient processing.TornClientInterface, sheetsClient processing.SheetsClientInterface, bqClient processing.BigQueryClientInterface, deployURL string) *StatusV2Processor {
	var deployer *deployment.SSHDeployer
	if deployURL != "" {
		deployer = deployment.NewSSHDeployer(deployURL)
	}

	var svc *StatusV2Service
	if bqClient != nil {
		svc = NewStatusV2ServiceWithBigQuery(sheetsClient, bqClient)
	} else {
		svc = NewStatusV2Service(sheetsClient)
	}

	return &StatusV2Processor{
		tornClient:   tornClient,
		sheetsClient: sheetsClient,
		service:      svc,
		ourFactionID: 0, // will be fetched via API when needed
		deployer:     deployer,
	}
}

// ensureOurFactionID fetches and caches our faction ID if not already set
func (p *StatusV2Processor) ensureOurFactionID(ctx context.Context) error {
	if p.ourFactionID == 0 {
		slog.Debug("StatusV2Processor: Fetching our faction ID from API")

		factionInfo, err := p.tornClient.GetOwnFaction(ctx)
		if err != nil {
			return fmt.Errorf("failed to get own faction info: %w", err)
		}

		p.ourFactionID = factionInfo.ID
		slog.Info("StatusV2Processor: Detected our faction ID",
			"faction_id", p.ourFactionID,
			"faction_name", factionInfo.Name,
			"faction_tag", factionInfo.Tag)
	}
	return nil
}

// ProcessStatusV2ForFactions processes Status v2 sheets for multiple factions
func (p *StatusV2Processor) ProcessStatusV2ForFactions(ctx context.Context, spreadsheetID string, factionIDs []int, updateInterval time.Duration) error {
	// Ensure our faction ID is loaded for proper filtering
	if err := p.ensureOurFactionID(ctx); err != nil {
		slog.Error("Failed to fetch our faction ID - continuing but filtering may be incorrect", "err", err)
	}

	slog.Info("Processing Status v2 for factions",
		"faction_count", len(factionIDs),
		"our_faction_id", p.ourFactionID)

	for _, factionID := range factionIDs {
		if err := p.ProcessStatusV2ForFaction(ctx, spreadsheetID, factionID, updateInterval); err != nil {
			slog.Error("Failed to process Status v2 for faction - continuing with others",
				"err", err,
				"faction_id", factionID)
			continue
		}

		slog.Debug("Successfully processed Status v2 for faction",
			"faction_id", factionID)
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

	// Step 3: Get current state records for this faction — prefer BigQuery over full sheet scan
	factionIDStr := fmt.Sprintf("%d", factionID)
	currentStateRecords, err := p.service.readCurrentStateRecords(ctx, spreadsheetID, factionIDStr)
	if err != nil {
		slog.Error("Failed to read state records",
			"err", err,
			"faction_id", factionID)
		return fmt.Errorf("failed to read state records: %w", err)
	}

	slog.Info("Filtered state records for faction",
		"faction_id", factionID,
		"filtered_state_records", len(currentStateRecords))

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

	slog.Info("Converted state records to Status v2 records",
		"faction_id", factionID,
		"status_v2_records", len(statusV2Records))

	// Step 6: Update the Status v2 sheet
	slog.Info("About to update Status v2 sheet",
		"faction_id", factionID,
		"sheet_name", sheetName,
		"records_to_write", len(statusV2Records))

	if len(statusV2Records) == 0 {
		slog.Warn("No Status v2 records to write - sheet will remain empty",
			"faction_id", factionID,
			"sheet_name", sheetName)
		return nil
	}

	if err := p.sheetsClient.UpdateStatusV2(ctx, spreadsheetID, sheetName, statusV2Records); err != nil {
		return fmt.Errorf("failed to update Status v2 sheet: %w", err)
	}

	slog.Info("Successfully updated Status v2 sheet",
		"faction_id", factionID,
		"records_count", len(statusV2Records),
		"sheet_name", sheetName,
		"state_records_found", len(currentStateRecords),
		"faction_members", len(factionData.Members))

	// Step 7: Export JSON alongside sheet update (only for opposing factions)
	if factionID != p.ourFactionID {
		if err := p.exportAndDeployJSON(statusV2Records, factionData.Name, factionID, updateInterval); err != nil {
			slog.Warn("Failed to export/deploy Status v2 JSON - continuing with processing",
				"err", err,
				"faction_id", factionID)
		}
	} else {
		slog.Debug("Skipping JSON export for our own faction",
			"faction_id", factionID)
	}

	return nil
}

// exportAndDeployJSON converts StatusV2Records to JSON format and deploys it
func (p *StatusV2Processor) exportAndDeployJSON(records []domain.StatusV2Record, factionName string, factionID int, updateInterval time.Duration) error {
	currentTime := time.Now().UTC()

	// Convert to JSON format using the service
	jsonData := p.service.ConvertToJSON(records, factionName, currentTime, updateInterval)

	// Marshal to JSON bytes
	jsonBytes, err := json.MarshalIndent(jsonData, "", "    ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	slog.Info("Successfully generated Status v2 JSON",
		"faction_id", factionID,
		"locations_count", len(jsonData.Locations),
		"json_size_bytes", len(jsonBytes))

	// Deploy to remote server if deployer is configured
	if p.deployer != nil {
		// Use a fixed filename for the remote deployment
		remoteFilename := "travel_data.json"

		// Deploy directly from memory without writing to disk
		if err := p.deployer.DeployData(bytes.NewReader(jsonBytes), int64(len(jsonBytes)), remoteFilename); err != nil {
			return fmt.Errorf("failed to deploy JSON data: %w", err)
		}

		slog.Info("Successfully deployed Status v2 JSON",
			"faction_id", factionID,
			"remote_file", remoteFilename,
			"size_bytes", len(jsonBytes))
	} else {
		slog.Debug("No deployer configured - skipping remote deployment",
			"faction_id", factionID)
	}

	return nil
}
