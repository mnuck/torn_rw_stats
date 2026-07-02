package processing

import (
	"context"
	"time"

	"torn_rw_stats/internal/domain"
	"torn_rw_stats/internal/domain/travel"
	"torn_rw_stats/internal/domain/war"
	"torn_rw_stats/internal/sheets"
)

// TornClientInterface defines the torn API client methods used by WarProcessor
type TornClientInterface interface {
	GetOwnFaction(ctx context.Context) (*domain.FactionInfo, error)
	GetFactionWars(ctx context.Context) (*domain.FactionWars, error)
	GetFactionAttacks(ctx context.Context, from, to int64) ([]domain.Attack, error)
	GetFactionBasic(ctx context.Context, factionID int) (*domain.FactionInfo, error)
	GetAPICallCount() int64
	IncrementAPICall()
	ResetAPICallCount()
}

// SheetsClientInterface defines the sheets API client methods used by WarProcessor
type SheetsClientInterface interface {
	EnsureWarSheets(ctx context.Context, spreadsheetID string, war *domain.War) (*domain.SheetConfig, error)
	ReadExistingRecords(ctx context.Context, spreadsheetID, sheetName string) (*sheets.RecordsInfo, error)
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

	// Status v2 methods
	EnsureStatusV2Sheet(ctx context.Context, spreadsheetID string, factionID int) (string, error)
	UpdateStatusV2(ctx context.Context, spreadsheetID, sheetName string, records []domain.StatusV2Record) error
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
	CalculateTravelTimes(ctx context.Context, userID int, destination string, travelType string, currentTime time.Time, updateInterval time.Duration) *travel.TravelTimeData
	CalculateTravelTimesFromDeparture(ctx context.Context, userID int, destination, departureStr, existingArrivalStr string, travelType string, currentTime time.Time, locationService *travel.LocationService, statusDescription string) *travel.TravelTimeData
}

// AttackProcessingServiceInterface defines the interface for attack processing
type AttackProcessingServiceInterface interface {
	ProcessAttacksIntoRecords(attacks []domain.Attack, war *domain.War, ourFactionID int) []domain.AttackRecord
}

// WarSummaryServiceInterface defines the interface for war summary generation
type WarSummaryServiceInterface interface {
	GenerateWarSummary(war *domain.War, attacks []domain.Attack, ourFactionID int) *domain.WarSummary
}

// WarStateManagerInterface defines the interface for war state management
type WarStateManagerInterface interface {
	GetCurrentState() war.WarState
	GetCurrentWar() *domain.War
}

// BigQueryClientInterface defines the BigQuery client methods used for state record insertion and querying
type BigQueryClientInterface interface {
	InsertStateRecords(ctx context.Context, records []domain.StateRecord) error
	QueryLatestStatePerMember(ctx context.Context, memberIDs []string) ([]domain.StateRecord, error)
	QueryLatestStatePerFaction(ctx context.Context, factionID string) ([]domain.StateRecord, error)
	QueryDepartureTimes(ctx context.Context, memberIDs []string) (map[string]time.Time, error)
}
