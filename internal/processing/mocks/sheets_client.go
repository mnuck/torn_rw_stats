package mocks

import (
	"context"

	"torn_rw_stats/internal/app"
	"torn_rw_stats/internal/sheets"
)

// SheetsClient interface defines the methods used by WarProcessor from sheets.Client
type SheetsClient interface {
	EnsureWarSheets(ctx context.Context, spreadsheetID string, war *app.War) (*app.SheetConfig, error)
	ReadExistingRecords(ctx context.Context, spreadsheetID, sheetName string) (*sheets.ExistingRecordsInfo, error)
	UpdateWarSummary(ctx context.Context, spreadsheetID string, config *app.SheetConfig, summary *app.WarSummary) error
	UpdateAttackRecords(ctx context.Context, spreadsheetID string, config *app.SheetConfig, records []app.AttackRecord) error
	ReadSheet(ctx context.Context, spreadsheetID, range_ string) ([][]interface{}, error)
	EnsurePreviousStateSheet(ctx context.Context, spreadsheetID string, factionID int) (string, error)
	LoadPreviousMemberStates(ctx context.Context, spreadsheetID, sheetName string) (map[string]app.FactionMember, error)
	StorePreviousMemberStates(ctx context.Context, spreadsheetID, sheetName string, members map[string]app.FactionMember) error
	EnsureStateChangeRecordsSheet(ctx context.Context, spreadsheetID string, factionID int) (string, error)
	AddStateChangeRecord(ctx context.Context, spreadsheetID, sheetName string, record app.StateChangeRecord) error

	// Additional methods for state tracking
	UpdateRange(ctx context.Context, spreadsheetID, range_ string, values [][]interface{}) error
	ClearRange(ctx context.Context, spreadsheetID, range_ string) error
	AppendRows(ctx context.Context, spreadsheetID, range_ string, rows [][]interface{}) error
	CreateSheet(ctx context.Context, spreadsheetID, sheetName string) error
	SheetExists(ctx context.Context, spreadsheetID, sheetName string) (bool, error)
	EnsureSheetCapacity(ctx context.Context, spreadsheetID, sheetName string, requiredRows, requiredCols int) error

	// Status v2 methods
	EnsureStatusV2Sheet(ctx context.Context, spreadsheetID string, factionID int) (string, error)
	UpdateStatusV2(ctx context.Context, spreadsheetID, sheetName string, records []app.StatusV2Record) error
}

// MockSheetsClient is a test double for the sheets.Client
type MockSheetsClient struct {
	// Responses to return
	EnsureWarSheetsResponse               *app.SheetConfig
	ReadExistingRecordsResponse           *sheets.ExistingRecordsInfo
	ReadSheetResponse                     [][]interface{}
	EnsurePreviousStateSheetResponse      string
	LoadPreviousMemberStatesResponse      map[string]app.FactionMember
	EnsureStateChangeRecordsSheetResponse string
	SheetExistsResponse                   bool
	EnsureStatusV2SheetResponse           string

	// Errors to return
	EnsureWarSheetsError               error
	ReadExistingRecordsError           error
	UpdateWarSummaryError              error
	UpdateAttackRecordsError           error
	ReadSheetError                     error
	EnsurePreviousStateSheetError      error
	LoadPreviousMemberStatesError      error
	StorePreviousMemberStatesError     error
	EnsureStateChangeRecordsSheetError error
	AddStateChangeRecordError          error
	UpdateRangeError                   error
	ClearRangeError                    error
	AppendRowsError                    error
	CreateSheetError                   error
	SheetExistsError                   error
	EnsureSheetCapacityError           error
	EnsureStatusV2SheetError           error
	UpdateStatusV2Error                error

	// Call tracking
	EnsureWarSheetsCalled               bool
	ReadExistingRecordsCalled           bool
	UpdateWarSummaryCalled              bool
	UpdateAttackRecordsCalled           bool
	ReadSheetCalled                     bool
	EnsurePreviousStateSheetCalled      bool
	LoadPreviousMemberStatesCalled      bool
	StorePreviousMemberStatesCalled     bool
	EnsureStateChangeRecordsSheetCalled bool
	AddStateChangeRecordCalled          bool

	// Call parameters tracking
	EnsureWarSheetsCalledWith struct {
		SpreadsheetID string
		War           *app.War
	}
	ReadExistingRecordsCalledWith struct {
		SpreadsheetID string
		SheetName     string
	}
	UpdateWarSummaryCalledWith struct {
		SpreadsheetID string
		Config        *app.SheetConfig
		Summary       *app.WarSummary
	}
	UpdateAttackRecordsCalledWith struct {
		SpreadsheetID string
		Config        *app.SheetConfig
		Records       []app.AttackRecord
	}
	ReadSheetCalledWith struct {
		SpreadsheetID string
		Range         string
	}
	EnsurePreviousStateSheetCalledWith struct {
		SpreadsheetID string
		FactionID     int
	}
	LoadPreviousMemberStatesCalledWith struct {
		SpreadsheetID string
		SheetName     string
	}
	StorePreviousMemberStatesCalledWith struct {
		SpreadsheetID string
		SheetName     string
		Members       map[string]app.FactionMember
	}
	EnsureStateChangeRecordsSheetCalledWith struct {
		SpreadsheetID string
		FactionID     int
	}
	AddStateChangeRecordCalledWith struct {
		SpreadsheetID string
		SheetName     string
		Record        app.StateChangeRecord
	}
}

// NewMockSheetsClient creates a new mock sheets client
func NewMockSheetsClient() *MockSheetsClient {
	return &MockSheetsClient{}
}

func (m *MockSheetsClient) EnsureWarSheets(ctx context.Context, spreadsheetID string, war *app.War) (*app.SheetConfig, error) {
	m.EnsureWarSheetsCalled = true
	m.EnsureWarSheetsCalledWith.SpreadsheetID = spreadsheetID
	m.EnsureWarSheetsCalledWith.War = war
	return m.EnsureWarSheetsResponse, m.EnsureWarSheetsError
}

func (m *MockSheetsClient) ReadExistingRecords(ctx context.Context, spreadsheetID, sheetName string) (*sheets.ExistingRecordsInfo, error) {
	m.ReadExistingRecordsCalled = true
	m.ReadExistingRecordsCalledWith.SpreadsheetID = spreadsheetID
	m.ReadExistingRecordsCalledWith.SheetName = sheetName
	return m.ReadExistingRecordsResponse, m.ReadExistingRecordsError
}

func (m *MockSheetsClient) UpdateWarSummary(ctx context.Context, spreadsheetID string, config *app.SheetConfig, summary *app.WarSummary) error {
	m.UpdateWarSummaryCalled = true
	m.UpdateWarSummaryCalledWith.SpreadsheetID = spreadsheetID
	m.UpdateWarSummaryCalledWith.Config = config
	m.UpdateWarSummaryCalledWith.Summary = summary
	return m.UpdateWarSummaryError
}

func (m *MockSheetsClient) UpdateAttackRecords(ctx context.Context, spreadsheetID string, config *app.SheetConfig, records []app.AttackRecord) error {
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


func (m *MockSheetsClient) EnsurePreviousStateSheet(ctx context.Context, spreadsheetID string, factionID int) (string, error) {
	m.EnsurePreviousStateSheetCalled = true
	m.EnsurePreviousStateSheetCalledWith.SpreadsheetID = spreadsheetID
	m.EnsurePreviousStateSheetCalledWith.FactionID = factionID
	return m.EnsurePreviousStateSheetResponse, m.EnsurePreviousStateSheetError
}

func (m *MockSheetsClient) LoadPreviousMemberStates(ctx context.Context, spreadsheetID, sheetName string) (map[string]app.FactionMember, error) {
	m.LoadPreviousMemberStatesCalled = true
	m.LoadPreviousMemberStatesCalledWith.SpreadsheetID = spreadsheetID
	m.LoadPreviousMemberStatesCalledWith.SheetName = sheetName
	return m.LoadPreviousMemberStatesResponse, m.LoadPreviousMemberStatesError
}

func (m *MockSheetsClient) StorePreviousMemberStates(ctx context.Context, spreadsheetID, sheetName string, members map[string]app.FactionMember) error {
	m.StorePreviousMemberStatesCalled = true
	m.StorePreviousMemberStatesCalledWith.SpreadsheetID = spreadsheetID
	m.StorePreviousMemberStatesCalledWith.SheetName = sheetName
	m.StorePreviousMemberStatesCalledWith.Members = members
	return m.StorePreviousMemberStatesError
}


func (m *MockSheetsClient) EnsureStateChangeRecordsSheet(ctx context.Context, spreadsheetID string, factionID int) (string, error) {
	m.EnsureStateChangeRecordsSheetCalled = true
	m.EnsureStateChangeRecordsSheetCalledWith.SpreadsheetID = spreadsheetID
	m.EnsureStateChangeRecordsSheetCalledWith.FactionID = factionID
	return m.EnsureStateChangeRecordsSheetResponse, m.EnsureStateChangeRecordsSheetError
}

func (m *MockSheetsClient) AddStateChangeRecord(ctx context.Context, spreadsheetID, sheetName string, record app.StateChangeRecord) error {
	m.AddStateChangeRecordCalled = true
	m.AddStateChangeRecordCalledWith.SpreadsheetID = spreadsheetID
	m.AddStateChangeRecordCalledWith.SheetName = sheetName
	m.AddStateChangeRecordCalledWith.Record = record
	return m.AddStateChangeRecordError
}

// Reset clears all call tracking and responses
func (m *MockSheetsClient) Reset() {
	// Clear responses
	m.EnsureWarSheetsResponse = nil
	m.ReadExistingRecordsResponse = nil
	m.ReadSheetResponse = nil
	m.EnsurePreviousStateSheetResponse = ""
	m.LoadPreviousMemberStatesResponse = nil
	m.EnsureStateChangeRecordsSheetResponse = ""

	// Clear errors
	m.EnsureWarSheetsError = nil
	m.ReadExistingRecordsError = nil
	m.UpdateWarSummaryError = nil
	m.UpdateAttackRecordsError = nil
	m.ReadSheetError = nil
	m.EnsurePreviousStateSheetError = nil
	m.LoadPreviousMemberStatesError = nil
	m.StorePreviousMemberStatesError = nil
	m.EnsureStateChangeRecordsSheetError = nil
	m.AddStateChangeRecordError = nil

	// Clear call tracking
	m.EnsureWarSheetsCalled = false
	m.ReadExistingRecordsCalled = false
	m.UpdateWarSummaryCalled = false
	m.UpdateAttackRecordsCalled = false
	m.ReadSheetCalled = false
	m.EnsurePreviousStateSheetCalled = false
	m.LoadPreviousMemberStatesCalled = false
	m.StorePreviousMemberStatesCalled = false
	m.EnsureStateChangeRecordsSheetCalled = false
	m.AddStateChangeRecordCalled = false

	// Clear parameter tracking
	m.EnsureWarSheetsCalledWith = struct {
		SpreadsheetID string
		War           *app.War
	}{}
	m.ReadExistingRecordsCalledWith = struct {
		SpreadsheetID string
		SheetName     string
	}{}
	m.UpdateWarSummaryCalledWith = struct {
		SpreadsheetID string
		Config        *app.SheetConfig
		Summary       *app.WarSummary
	}{}
	m.UpdateAttackRecordsCalledWith = struct {
		SpreadsheetID string
		Config        *app.SheetConfig
		Records       []app.AttackRecord
	}{}
	m.ReadSheetCalledWith = struct {
		SpreadsheetID string
		Range         string
	}{}
	m.EnsurePreviousStateSheetCalledWith = struct {
		SpreadsheetID string
		FactionID     int
	}{}
	m.LoadPreviousMemberStatesCalledWith = struct {
		SpreadsheetID string
		SheetName     string
	}{}
	m.StorePreviousMemberStatesCalledWith = struct {
		SpreadsheetID string
		SheetName     string
		Members       map[string]app.FactionMember
	}{}
	m.EnsureStateChangeRecordsSheetCalledWith = struct {
		SpreadsheetID string
		FactionID     int
	}{}
	m.AddStateChangeRecordCalledWith = struct {
		SpreadsheetID string
		SheetName     string
		Record        app.StateChangeRecord
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

// Status v2 methods
func (m *MockSheetsClient) EnsureStatusV2Sheet(ctx context.Context, spreadsheetID string, factionID int) (string, error) {
	return m.EnsureStatusV2SheetResponse, m.EnsureStatusV2SheetError
}

func (m *MockSheetsClient) UpdateStatusV2(ctx context.Context, spreadsheetID, sheetName string, records []app.StatusV2Record) error {
	return m.UpdateStatusV2Error
}
