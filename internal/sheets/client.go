package sheets

import (
	"context"
	"fmt"

	"github.com/rs/zerolog/log"
	"google.golang.org/api/option"
	"google.golang.org/api/sheets/v4"
)

// Client implements the SheetsAPI interface using Google Sheets API.
//
// Note: This client uses [][]interface{} as required by the Google Sheets API.
// This is the only layer where interface{} should appear. All other code should
// use the Cell type wrapper for type-safe access to cell values.
type Client struct {
	service *sheets.Service
}

// NewClient creates a new Google Sheets client with the provided credentials
func NewClient(ctx context.Context, credentialsFile string) (*Client, error) {
	service, err := sheets.NewService(ctx, option.WithCredentialsFile(credentialsFile))
	if err != nil {
		return nil, fmt.Errorf("failed to create sheets service: %w", err)
	}

	return &Client{
		service: service,
	}, nil
}

// ReadSheet reads values from the specified sheet range.
// Returns [][]interface{} as mandated by Google Sheets API.
// Wrap returned values with NewCell() for type-safe access.
func (c *Client) ReadSheet(ctx context.Context, spreadsheetID, range_ string) ([][]interface{}, error) {
	resp, err := c.service.Spreadsheets.Values.Get(spreadsheetID, range_).Context(ctx).Do()
	if err != nil {
		return nil, fmt.Errorf("failed to read sheet: %w", err)
	}

	return resp.Values, nil
}

// UpdateRange updates the specified sheet range with the provided values.
// Accepts [][]interface{} as mandated by Google Sheets API.
func (c *Client) UpdateRange(ctx context.Context, spreadsheetID, range_ string, values [][]interface{}) error {
	valueRange := &sheets.ValueRange{
		Values: values,
	}

	_, err := c.service.Spreadsheets.Values.Update(spreadsheetID, range_, valueRange).
		ValueInputOption("USER_ENTERED").
		Context(ctx).
		Do()
	if err != nil {
		return fmt.Errorf("failed to update range: %w", err)
	}

	return nil
}

// ClearRange clears all values in the specified sheet range
func (c *Client) ClearRange(ctx context.Context, spreadsheetID, range_ string) error {
	_, err := c.service.Spreadsheets.Values.Clear(spreadsheetID, range_, &sheets.ClearValuesRequest{}).
		Context(ctx).
		Do()
	if err != nil {
		return fmt.Errorf("failed to clear range: %w", err)
	}

	return nil
}

// AppendRows appends rows to the specified sheet range.
// Accepts [][]interface{} as mandated by Google Sheets API.
func (c *Client) AppendRows(ctx context.Context, spreadsheetID, range_ string, rows [][]interface{}) error {
	valueRange := &sheets.ValueRange{
		Values: rows,
	}

	_, err := c.service.Spreadsheets.Values.Append(spreadsheetID, range_, valueRange).
		ValueInputOption("USER_ENTERED").
		InsertDataOption("INSERT_ROWS").
		Context(ctx).
		Do()
	if err != nil {
		return fmt.Errorf("failed to append rows: %w", err)
	}

	return nil
}

// CreateSheet creates a new sheet with the specified name
func (c *Client) CreateSheet(ctx context.Context, spreadsheetID, sheetName string) error {
	req := &sheets.Request{
		AddSheet: &sheets.AddSheetRequest{
			Properties: &sheets.SheetProperties{
				Title: sheetName,
			},
		},
	}

	batchUpdate := &sheets.BatchUpdateSpreadsheetRequest{
		Requests: []*sheets.Request{req},
	}

	_, err := c.service.Spreadsheets.BatchUpdate(spreadsheetID, batchUpdate).
		Context(ctx).
		Do()
	if err != nil {
		return fmt.Errorf("failed to create sheet %s: %w", sheetName, err)
	}

	return nil
}

// SheetExists checks if a sheet with the given name exists in the spreadsheet
func (c *Client) SheetExists(ctx context.Context, spreadsheetID, sheetName string) (bool, error) {
	spreadsheet, err := c.service.Spreadsheets.Get(spreadsheetID).Context(ctx).Do()
	if err != nil {
		return false, fmt.Errorf("failed to get spreadsheet: %w", err)
	}

	for _, sheet := range spreadsheet.Sheets {
		if sheet.Properties.Title == sheetName {
			return true, nil
		}
	}

	return false, nil
}

// EnsureSheetCapacity ensures the sheet has at least the required number of rows and columns.
// Automatically adds a buffer for future growth.
func (c *Client) EnsureSheetCapacity(ctx context.Context, spreadsheetID, sheetName string, requiredRows, requiredCols int) error {
	spreadsheet, err := c.service.Spreadsheets.Get(spreadsheetID).Context(ctx).Do()
	if err != nil {
		return fmt.Errorf("failed to get spreadsheet: %w", err)
	}

	var targetSheet *sheets.Sheet
	for _, sheet := range spreadsheet.Sheets {
		if sheet.Properties.Title == sheetName {
			targetSheet = sheet
			break
		}
	}

	if targetSheet == nil {
		return fmt.Errorf("sheet %s not found", sheetName)
	}

	currentRows := int(targetSheet.Properties.GridProperties.RowCount)
	currentCols := int(targetSheet.Properties.GridProperties.ColumnCount)

	needsResize := false
	newRows := currentRows
	newCols := currentCols

	if requiredRows > currentRows {
		newRows = requiredRows + 100 // Add buffer for future growth
		needsResize = true
	}

	if requiredCols > currentCols {
		newCols = requiredCols + 10 // Add buffer for future columns
		needsResize = true
	}

	if !needsResize {
		return nil
	}

	log.Debug().
		Str("sheet_name", sheetName).
		Int("current_rows", currentRows).
		Int("current_cols", currentCols).
		Int("required_rows", requiredRows).
		Int("required_cols", requiredCols).
		Int("new_rows", newRows).
		Int("new_cols", newCols).
		Msg("Expanding sheet capacity")

	req := &sheets.Request{
		UpdateSheetProperties: &sheets.UpdateSheetPropertiesRequest{
			Properties: &sheets.SheetProperties{
				SheetId: targetSheet.Properties.SheetId,
				GridProperties: &sheets.GridProperties{
					RowCount:    int64(newRows),
					ColumnCount: int64(newCols),
				},
			},
			Fields: "gridProperties.rowCount,gridProperties.columnCount",
		},
	}

	batchUpdate := &sheets.BatchUpdateSpreadsheetRequest{
		Requests: []*sheets.Request{req},
	}

	_, err = c.service.Spreadsheets.BatchUpdate(spreadsheetID, batchUpdate).
		Context(ctx).
		Do()
	if err != nil {
		return fmt.Errorf("failed to resize sheet %s: %w", sheetName, err)
	}

	log.Info().
		Str("sheet_name", sheetName).
		Int("new_rows", newRows).
		Int("new_cols", newCols).
		Msg("Successfully expanded sheet capacity")

	return nil
}

// FormatStatusSheet is disabled - formatting is handled manually in Google Sheets
// We only keep the apostrophe prefix in travel time formatting to prevent decimal conversion
func (c *Client) FormatStatusSheet(ctx context.Context, spreadsheetID, sheetName string) error {
	log.Debug().
		Str("sheet_name", sheetName).
		Msg("Skipping automatic formatting - handled manually")
	return nil
}
