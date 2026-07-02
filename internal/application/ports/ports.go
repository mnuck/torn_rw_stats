// Package ports defines the interfaces through which the application core
// talks to the outside world (Torn API, Google Sheets, BigQuery) and to
// injectable domain services. Adapters implement these interfaces; the
// application layer depends only on them. Only domain types cross these
// boundaries.
package ports

import (
	"context"
	"io"
	"time"

	"torn_rw_stats/internal/domain"
	"torn_rw_stats/internal/domain/travel"
	"torn_rw_stats/internal/domain/war"
)

// Deployer publishes generated artifacts (e.g. the Status v2 JSON export)
// to an external destination.
type Deployer interface {
	DeployData(data io.Reader, size int64, filename string) error
}

// TornClient defines the Torn API operations used by the application
type TornClient interface {
	GetOwnFaction(ctx context.Context) (*domain.FactionInfo, error)
	GetFactionWars(ctx context.Context) (*domain.FactionWars, error)
	GetFactionAttacks(ctx context.Context, from, to int64) ([]domain.Attack, error)
	GetFactionBasic(ctx context.Context, factionID int) (*domain.FactionInfo, error)
	GetAPICallCount() int64
	IncrementAPICall()
	ResetAPICallCount()
}

// SheetsClient defines the spreadsheet operations used by the application
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

	// Status v2 methods
	EnsureStatusV2Sheet(ctx context.Context, spreadsheetID string, factionID int) (string, error)
	UpdateStatusV2(ctx context.Context, spreadsheetID, sheetName string, records []domain.StatusV2Record) error
}

// LocationService defines the location parsing operations used by the application
type LocationService interface {
	ParseLocation(description string) string
	GetTravelDestinationForCalculation(description, parsedLocation string) string
}

// TravelTimeService defines the travel time operations used by the application
type TravelTimeService interface {
	GetTravelTime(destination string, travelType string) time.Duration
	FormatTravelTime(d time.Duration) string
	CalculateTravelTimes(ctx context.Context, userID int, destination string, travelType string, currentTime time.Time, updateInterval time.Duration) *travel.TravelTimeData
	CalculateTravelTimesFromDeparture(ctx context.Context, userID int, destination, departureStr, existingArrivalStr string, travelType string, currentTime time.Time, locationService *travel.LocationService, statusDescription string) *travel.TravelTimeData
}

// AttackProcessingService defines the interface for attack processing
type AttackProcessingService interface {
	ProcessAttacksIntoRecords(attacks []domain.Attack, war *domain.War, ourFactionID int) []domain.AttackRecord
}

// WarSummaryService defines the interface for war summary generation
type WarSummaryService interface {
	GenerateWarSummary(war *domain.War, attacks []domain.Attack, ourFactionID int) *domain.WarSummary
}

// WarStateManager defines the interface for war state management
type WarStateManager interface {
	GetCurrentState() war.WarState
	GetCurrentWar() *domain.War
}

// BigQueryClient defines the BigQuery operations used for state record
// insertion and querying
type BigQueryClient interface {
	InsertStateRecords(ctx context.Context, records []domain.StateRecord) error
	QueryLatestStatePerMember(ctx context.Context, memberIDs []string) ([]domain.StateRecord, error)
	QueryLatestStatePerFaction(ctx context.Context, factionID string) ([]domain.StateRecord, error)
	QueryDepartureTimes(ctx context.Context, memberIDs []string) (map[string]time.Time, error)
}
