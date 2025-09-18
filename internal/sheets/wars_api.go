package sheets

import (
	"context"

	"torn_rw_stats/internal/app"
)

// War-related API functions that use the infrastructure layer
// These functions delegate to the specialized managers for actual business logic

// EnsureWarSheets creates summary and records sheets for a war if they don't exist
func (c *Client) EnsureWarSheets(ctx context.Context, spreadsheetID string, war *app.War) (*app.SheetConfig, error) {
	manager := NewWarSheetsManager(c)
	return manager.EnsureWarSheets(ctx, spreadsheetID, war)
}

// UpdateWarSummary updates the summary sheet with current war statistics
func (c *Client) UpdateWarSummary(ctx context.Context, spreadsheetID string, config *app.SheetConfig, summary *app.WarSummary) error {
	manager := NewWarSheetsManager(c)
	return manager.UpdateWarSummary(ctx, spreadsheetID, config, summary)
}

// ReadExistingRecords analyzes existing attack records in the sheet
func (c *Client) ReadExistingRecords(ctx context.Context, spreadsheetID, sheetName string) (*RecordsInfo, error) {
	processor := NewAttackRecordsProcessor(c)
	return processor.ReadExistingRecords(ctx, spreadsheetID, sheetName)
}

// UpdateAttackRecords updates the records sheet with new attack data using append strategy
func (c *Client) UpdateAttackRecords(ctx context.Context, spreadsheetID string, config *app.SheetConfig, records []app.AttackRecord) error {
	processor := NewAttackRecordsProcessor(c)
	return processor.UpdateAttackRecords(ctx, spreadsheetID, config, records)
}

// Travel and State Management Functions - delegate to specialized managers

// EnsureStatusV2Sheet creates Status v2 sheet for a faction if it doesn't exist
func (c *Client) EnsureStatusV2Sheet(ctx context.Context, spreadsheetID string, factionID int) (string, error) {
	manager := NewStatusV2Manager(c)
	return manager.EnsureStatusV2Sheet(ctx, spreadsheetID, factionID)
}

// UpdateStatusV2 updates the Status v2 sheet with current state record data
func (c *Client) UpdateStatusV2(ctx context.Context, spreadsheetID, sheetName string, records []app.StatusV2Record) error {
	manager := NewStatusV2Manager(c)
	return manager.UpdateStatusV2(ctx, spreadsheetID, sheetName, records)
}
