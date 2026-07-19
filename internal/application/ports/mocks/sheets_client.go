package mocks

import (
	"context"

	"torn_rw_stats/internal/domain"
)

// SheetsClient interface defines the methods used by WarProcessor from sheets.Client
type SheetsClient interface {
	EnsureWarSheets(ctx context.Context, spreadsheetID string, war *domain.War) (*domain.SheetConfig, error)
	ReadExistingRecords(ctx context.Context, spreadsheetID, sheetName string) (*domain.RecordsInfo, error)
	UpdateWarSummary(ctx context.Context, spreadsheetID string, config *domain.SheetConfig, summary *domain.WarSummary) error
	UpdateAttackRecords(ctx context.Context, spreadsheetID string, config *domain.SheetConfig, records []domain.AttackRecord) error
	ReadSheet(ctx context.Context, spreadsheetID, range_ string) ([][]interface{}, error)

	// Additional methods for state tracking
	UpdateRange(ctx context.Context, spreadsheetID, range_ string, values [][]interface{}) error
	ClearRange(ctx context.Context, spreadsheetID, range_ string) error
	AppendRows(ctx context.Context, spreadsheetID, range_ string, rows [][]interface{}) error
	CreateSheet(ctx context.Context, spreadsheetID, sheetName string) error
	SheetExists(ctx context.Context, spreadsheetID, sheetName string) (bool, error)
	EnsureSheetCapacity(ctx context.Context, spreadsheetID, sheetName string, requiredRows, requiredCols int) error
	ListSheetTitles(ctx context.Context, spreadsheetID string) ([]string, error)
	DeleteSheet(ctx context.Context, spreadsheetID, sheetName string) error

	// Status v2 methods
	EnsureStatusV2Sheet(ctx context.Context, spreadsheetID string, factionID int) (string, error)
	UpdateStatusV2(ctx context.Context, spreadsheetID, sheetName string, records []domain.StatusV2Record) error
}

// MockSheetsClient is a test double for the sheets.Client
type MockSheetsClient struct {
	// Responses to return
	EnsureWarSheetsResponse     *domain.SheetConfig
	ReadExistingRecordsResponse *domain.RecordsInfo
	ReadSheetResponse           [][]interface{}
	SheetExistsResponse         bool
	EnsureStatusV2SheetResponse string
	ListSheetTitlesResponse     []string

	// ReadExistingRecordsFunc, when set, overrides ReadExistingRecordsResponse
	// and lets a test return a different result per sheet name.
	ReadExistingRecordsFunc func(sheetName string) (*domain.RecordsInfo, error)

	// Errors to return
	EnsureWarSheetsError     error
	ReadExistingRecordsError error
	UpdateWarSummaryError    error
	UpdateAttackRecordsError error
	ReadSheetError           error
	UpdateRangeError         error
	ClearRangeError          error
	AppendRowsError          error
	CreateSheetError         error
	SheetExistsError         error
	EnsureSheetCapacityError error
	EnsureStatusV2SheetError error
	UpdateStatusV2Error      error
	ListSheetTitlesError     error
	DeleteSheetError         error

	// DeletedSheets records the names passed to DeleteSheet, in call order.
	DeletedSheets []string

	// Call tracking
	EnsureWarSheetsCalled     bool
	ReadExistingRecordsCalled bool
	UpdateWarSummaryCalled    bool
	UpdateAttackRecordsCalled bool
	ReadSheetCalled           bool

	// Call parameters tracking
	EnsureWarSheetsCalledWith struct {
		SpreadsheetID string
		War           *domain.War
	}
	ReadExistingRecordsCalledWith struct {
		SpreadsheetID string
		SheetName     string
	}
	UpdateWarSummaryCalledWith struct {
		SpreadsheetID string
		Config        *domain.SheetConfig
		Summary       *domain.WarSummary
	}
	UpdateAttackRecordsCalledWith struct {
		SpreadsheetID string
		Config        *domain.SheetConfig
		Records       []domain.AttackRecord
	}
	ReadSheetCalledWith struct {
		SpreadsheetID string
		Range         string
	}
}

// NewMockSheetsClient creates a new mock sheets client
func NewMockSheetsClient() *MockSheetsClient {
	return &MockSheetsClient{}
}

func (m *MockSheetsClient) EnsureWarSheets(ctx context.Context, spreadsheetID string, war *domain.War) (*domain.SheetConfig, error) {
	m.EnsureWarSheetsCalled = true
	m.EnsureWarSheetsCalledWith.SpreadsheetID = spreadsheetID
	m.EnsureWarSheetsCalledWith.War = war
	return m.EnsureWarSheetsResponse, m.EnsureWarSheetsError
}

func (m *MockSheetsClient) ReadExistingRecords(ctx context.Context, spreadsheetID, sheetName string) (*domain.RecordsInfo, error) {
	m.ReadExistingRecordsCalled = true
	m.ReadExistingRecordsCalledWith.SpreadsheetID = spreadsheetID
	m.ReadExistingRecordsCalledWith.SheetName = sheetName
	if m.ReadExistingRecordsFunc != nil {
		return m.ReadExistingRecordsFunc(sheetName)
	}
	return m.ReadExistingRecordsResponse, m.ReadExistingRecordsError
}

func (m *MockSheetsClient) UpdateWarSummary(ctx context.Context, spreadsheetID string, config *domain.SheetConfig, summary *domain.WarSummary) error {
	m.UpdateWarSummaryCalled = true
	m.UpdateWarSummaryCalledWith.SpreadsheetID = spreadsheetID
	m.UpdateWarSummaryCalledWith.Config = config
	m.UpdateWarSummaryCalledWith.Summary = summary
	return m.UpdateWarSummaryError
}

func (m *MockSheetsClient) UpdateAttackRecords(ctx context.Context, spreadsheetID string, config *domain.SheetConfig, records []domain.AttackRecord) error {
	m.UpdateAttackRecordsCalled = true
	m.UpdateAttackRecordsCalledWith.SpreadsheetID = spreadsheetID
	m.UpdateAttackRecordsCalledWith.Config = config
	m.UpdateAttackRecordsCalledWith.Records = records
	return m.UpdateAttackRecordsError
}

func (m *MockSheetsClient) ReadSheet(ctx context.Context, spreadsheetID, range_ string) ([][]interface{}, error) {
	m.ReadSheetCalled = true
	m.ReadSheetCalledWith.SpreadsheetID = spreadsheetID
	m.ReadSheetCalledWith.Range = range_
	return m.ReadSheetResponse, m.ReadSheetError
}

// Reset clears all call tracking and responses
func (m *MockSheetsClient) Reset() {
	// Clear responses
	m.EnsureWarSheetsResponse = nil
	m.ReadExistingRecordsResponse = nil
	m.ReadSheetResponse = nil

	// Clear errors
	m.EnsureWarSheetsError = nil
	m.ReadExistingRecordsError = nil
	m.UpdateWarSummaryError = nil
	m.UpdateAttackRecordsError = nil
	m.ReadSheetError = nil

	// Clear call tracking
	m.EnsureWarSheetsCalled = false
	m.ReadExistingRecordsCalled = false
	m.UpdateWarSummaryCalled = false
	m.UpdateAttackRecordsCalled = false
	m.ReadSheetCalled = false

	// Clear parameter tracking
	m.EnsureWarSheetsCalledWith = struct {
		SpreadsheetID string
		War           *domain.War
	}{}
	m.ReadExistingRecordsCalledWith = struct {
		SpreadsheetID string
		SheetName     string
	}{}
	m.UpdateWarSummaryCalledWith = struct {
		SpreadsheetID string
		Config        *domain.SheetConfig
		Summary       *domain.WarSummary
	}{}
	m.UpdateAttackRecordsCalledWith = struct {
		SpreadsheetID string
		Config        *domain.SheetConfig
		Records       []domain.AttackRecord
	}{}
	m.ReadSheetCalledWith = struct {
		SpreadsheetID string
		Range         string
	}{}
}

// Additional state tracking methods
func (m *MockSheetsClient) UpdateRange(ctx context.Context, spreadsheetID, range_ string, values [][]interface{}) error {
	return m.UpdateRangeError
}

func (m *MockSheetsClient) ClearRange(ctx context.Context, spreadsheetID, range_ string) error {
	return m.ClearRangeError
}

func (m *MockSheetsClient) AppendRows(ctx context.Context, spreadsheetID, range_ string, rows [][]interface{}) error {
	return m.AppendRowsError
}

func (m *MockSheetsClient) CreateSheet(ctx context.Context, spreadsheetID, sheetName string) error {
	return m.CreateSheetError
}

func (m *MockSheetsClient) SheetExists(ctx context.Context, spreadsheetID, sheetName string) (bool, error) {
	return m.SheetExistsResponse, m.SheetExistsError
}

func (m *MockSheetsClient) EnsureSheetCapacity(ctx context.Context, spreadsheetID, sheetName string, requiredRows, requiredCols int) error {
	return m.EnsureSheetCapacityError
}

func (m *MockSheetsClient) ListSheetTitles(ctx context.Context, spreadsheetID string) ([]string, error) {
	return m.ListSheetTitlesResponse, m.ListSheetTitlesError
}

func (m *MockSheetsClient) DeleteSheet(ctx context.Context, spreadsheetID, sheetName string) error {
	if m.DeleteSheetError != nil {
		return m.DeleteSheetError
	}
	m.DeletedSheets = append(m.DeletedSheets, sheetName)
	return nil
}

// Status v2 methods
func (m *MockSheetsClient) EnsureStatusV2Sheet(ctx context.Context, spreadsheetID string, factionID int) (string, error) {
	return m.EnsureStatusV2SheetResponse, m.EnsureStatusV2SheetError
}

func (m *MockSheetsClient) UpdateStatusV2(ctx context.Context, spreadsheetID, sheetName string, records []domain.StatusV2Record) error {
	return m.UpdateStatusV2Error
}
