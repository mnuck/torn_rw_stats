package processing

import (
	"context"
	"time"

	"torn_rw_stats/internal/app"
	"torn_rw_stats/internal/sheets"
)

// TornClientInterface defines the torn API client methods used by WarProcessor
type TornClientInterface interface {
	GetOwnFaction(ctx context.Context) (*app.FactionInfoResponse, error)
	GetFactionWars(ctx context.Context) (*app.WarResponse, error)
	GetAllAttacksForWar(ctx context.Context, war *app.War) ([]app.Attack, error)
	GetAttacksForTimeRange(ctx context.Context, war *app.War, fromTime int64, latestExistingTimestamp *int64) ([]app.Attack, error)
	GetFactionBasic(ctx context.Context, factionID int) (*app.FactionBasicResponse, error)
}

// SheetsClientInterface defines the sheets API client methods used by WarProcessor
type SheetsClientInterface interface {
	EnsureWarSheets(ctx context.Context, spreadsheetID string, war *app.War) (*app.SheetConfig, error)
	ReadExistingRecords(ctx context.Context, spreadsheetID, sheetName string) (*sheets.ExistingRecordsInfo, error)
	UpdateWarSummary(ctx context.Context, spreadsheetID string, config *app.SheetConfig, summary *app.WarSummary) error
	UpdateAttackRecords(ctx context.Context, spreadsheetID string, config *app.SheetConfig, records []app.AttackRecord) error
	ReadSheet(ctx context.Context, spreadsheetID, range_ string) ([][]interface{}, error)
	EnsureTravelStatusSheet(ctx context.Context, spreadsheetID string, factionID int) (string, error)
	EnsurePreviousStateSheet(ctx context.Context, spreadsheetID string, factionID int) (string, error)
	LoadPreviousMemberStates(ctx context.Context, spreadsheetID, sheetName string) (map[string]app.FactionMember, error)
	StorePreviousMemberStates(ctx context.Context, spreadsheetID, sheetName string, members map[string]app.FactionMember) error
	UpdateTravelStatus(ctx context.Context, spreadsheetID, sheetName string, records []app.TravelRecord) error
	EnsureStateChangeRecordsSheet(ctx context.Context, spreadsheetID string, factionID int) (string, error)
	AddStateChangeRecord(ctx context.Context, spreadsheetID, sheetName string, record app.StateChangeRecord) error

	// Additional methods for state tracking
	UpdateRange(ctx context.Context, spreadsheetID, range_ string, values [][]interface{}) error
	ClearRange(ctx context.Context, spreadsheetID, range_ string) error
	AppendRows(ctx context.Context, spreadsheetID, range_ string, rows [][]interface{}) error
	CreateSheet(ctx context.Context, spreadsheetID, sheetName string) error
	SheetExists(ctx context.Context, spreadsheetID, sheetName string) (bool, error)
	EnsureSheetCapacity(ctx context.Context, spreadsheetID, sheetName string, requiredRows, requiredCols int) error
}

// LocationServiceInterface defines the location service methods used by WarProcessor
type LocationServiceInterface interface {
	ParseLocation(description string) string
	GetTravelDestinationForCalculation(description, parsedLocation string) string
}

// TravelTimeServiceInterface defines the travel time service methods used by WarProcessor
type TravelTimeServiceInterface interface {
	GetTravelTime(destination string, travelType string) time.Duration
	FormatTravelTime(d time.Duration) string
	CalculateTravelTimes(ctx context.Context, userID int, destination string, travelType string, currentTime time.Time, updateInterval time.Duration) *TravelTimeData
	CalculateTravelTimesFromDeparture(ctx context.Context, userID int, destination, departureStr, existingArrivalStr string, travelType string, currentTime time.Time, locationService LocationServiceInterface) *TravelTimeData
}

// AttackProcessingServiceInterface defines the interface for attack processing
type AttackProcessingServiceInterface interface {
	ProcessAttacksIntoRecords(attacks []app.Attack, war *app.War) []app.AttackRecord
}

// WarSummaryServiceInterface defines the interface for war summary generation
type WarSummaryServiceInterface interface {
	GenerateWarSummary(war *app.War, attacks []app.Attack) *app.WarSummary
}

// StateChangeDetectionServiceInterface defines the interface for state change detection
type StateChangeDetectionServiceInterface interface {
	NormalizeHospitalDescription(description string) string
	HasStatusChanged(oldMember, newMember app.FactionMember) bool
	ProcessStateChanges(ctx context.Context, factionID int, factionName string, oldMembers, newMembers map[string]app.FactionMember, spreadsheetID string) error
}

// WarStateManagerInterface defines the interface for war state management
type WarStateManagerInterface interface {
	GetCurrentState() WarState
	GetCurrentWar() *app.War
}
