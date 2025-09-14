package sheets

import (
	"context"
)

// SheetsAPI defines the interface for interacting with Google Sheets
// This separates infrastructure concerns from business logic
type SheetsAPI interface {
	// Basic sheet operations
	ReadSheet(ctx context.Context, spreadsheetID, range_ string) ([][]interface{}, error)
	UpdateRange(ctx context.Context, spreadsheetID, range_ string, values [][]interface{}) error
	ClearRange(ctx context.Context, spreadsheetID, range_ string) error
	AppendRows(ctx context.Context, spreadsheetID, range_ string, rows [][]interface{}) error

	// Sheet management
	CreateSheet(ctx context.Context, spreadsheetID, sheetName string) error
	SheetExists(ctx context.Context, spreadsheetID, sheetName string) (bool, error)
	EnsureSheetCapacity(ctx context.Context, spreadsheetID, sheetName string, requiredRows, requiredCols int) error

	// Sheet formatting
	FormatStatusSheet(ctx context.Context, spreadsheetID, sheetName string) error
}
