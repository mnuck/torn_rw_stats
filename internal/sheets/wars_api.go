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

// ExistingRecordsInfo contains information about existing attack records in the sheet
// Deprecated: Use RecordsInfo from AttackRecordsProcessor instead
type ExistingRecordsInfo = RecordsInfo

// ReadExistingRecords analyzes existing attack records in the sheet
func (c *Client) ReadExistingRecords(ctx context.Context, spreadsheetID, sheetName string) (*ExistingRecordsInfo, error) {
	processor := NewAttackRecordsProcessor(c)
	return processor.ReadExistingRecords(ctx, spreadsheetID, sheetName)
}

// UpdateAttackRecords updates the records sheet with new attack data using append strategy
func (c *Client) UpdateAttackRecords(ctx context.Context, spreadsheetID string, config *app.SheetConfig, records []app.AttackRecord) error {
	processor := NewAttackRecordsProcessor(c)
	return processor.UpdateAttackRecords(ctx, spreadsheetID, config, records)
}

// Travel and State Management Functions - delegate to specialized managers

// convertMembersToStateRows converts member map to sheet row format
func (c *Client) convertMembersToStateRows(members map[string]app.FactionMember) [][]interface{} {
	manager := NewStateChangeManager(c)
	return manager.ConvertMembersToStateRows(members)
}

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

// EnsureStateChangeRecordsSheet creates state change records sheet for a faction if it doesn't exist
func (c *Client) EnsureStateChangeRecordsSheet(ctx context.Context, spreadsheetID string, factionID int) (string, error) {
	manager := NewStateChangeManager(c)
	return manager.EnsureStateChangeRecordsSheet(ctx, spreadsheetID, factionID)
}

// AddStateChangeRecord adds a new state change record to the sheet
func (c *Client) AddStateChangeRecord(ctx context.Context, spreadsheetID, sheetName string, record app.StateChangeRecord) error {
	manager := NewStateChangeManager(c)
	return manager.AddStateChangeRecord(ctx, spreadsheetID, sheetName, record)
}

// EnsurePreviousStateSheet creates previous state sheet for a faction if it doesn't exist
func (c *Client) EnsurePreviousStateSheet(ctx context.Context, spreadsheetID string, factionID int) (string, error) {
	manager := NewStateChangeManager(c)
	return manager.EnsurePreviousStateSheet(ctx, spreadsheetID, factionID)
}

// StorePreviousMemberStates stores current member states to the previous state sheet
func (c *Client) StorePreviousMemberStates(ctx context.Context, spreadsheetID, sheetName string, members map[string]app.FactionMember) error {
	manager := NewStateChangeManager(c)
	return manager.StorePreviousMemberStates(ctx, spreadsheetID, sheetName, members)
}

// LoadPreviousMemberStates loads previous member states from the sheet
func (c *Client) LoadPreviousMemberStates(ctx context.Context, spreadsheetID, sheetName string) (map[string]app.FactionMember, error) {
	manager := NewStateChangeManager(c)
	return manager.LoadPreviousMemberStates(ctx, spreadsheetID, sheetName)
}
