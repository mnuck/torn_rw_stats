package sheets

import (
	"context"
)

// SheetsAPI defines the interface for interacting with Google Sheets.
// This separates infrastructure concerns from business logic.
//
// Note on interface{} usage:
// The Google Sheets API (google.golang.org/api/sheets/v4) uses [][]interface{}
// for cell values. This is outside our control and required for API compatibility.
// To minimize unsafe interface{} usage in our codebase:
// - Use the Cell type wrapper for type-safe value extraction
// - Keep interface{} constrained to this API boundary layer
// - Never expose interface{} in business logic or domain layers
type SheetsAPI interface {
	// ReadSheet reads values from a sheet range.
	// Returns [][]interface{} as required by Google Sheets API.
	// Use NewCell() to wrap values for type-safe access.
	ReadSheet(ctx context.Context, spreadsheetID, range_ string) ([][]interface{}, error)

	// UpdateRange updates values in a sheet range.
	// Accepts [][]interface{} as required by Google Sheets API.
	UpdateRange(ctx context.Context, spreadsheetID, range_ string, values [][]interface{}) error

	// ClearRange clears all values in a sheet range
	ClearRange(ctx context.Context, spreadsheetID, range_ string) error

	// AppendRows appends rows to a sheet.
	// Accepts [][]interface{} as required by Google Sheets API.
	AppendRows(ctx context.Context, spreadsheetID, range_ string, rows [][]interface{}) error

	// CreateSheet creates a new sheet in the spreadsheet
	CreateSheet(ctx context.Context, spreadsheetID, sheetName string) error

	// SheetExists checks if a sheet with the given name exists
	SheetExists(ctx context.Context, spreadsheetID, sheetName string) (bool, error)

	// EnsureSheetCapacity ensures a sheet has at least the required number of rows and columns
	EnsureSheetCapacity(ctx context.Context, spreadsheetID, sheetName string, requiredRows, requiredCols int) error

	// FormatStatusSheet applies formatting to a status sheet
	FormatStatusSheet(ctx context.Context, spreadsheetID, sheetName string) error
}
