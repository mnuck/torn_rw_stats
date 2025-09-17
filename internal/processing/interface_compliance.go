package processing

import (
	"torn_rw_stats/internal/domain/attack"
	"torn_rw_stats/internal/domain/travel"
	"torn_rw_stats/internal/sheets"
	"torn_rw_stats/internal/torn"
)

// Compile-time interface compliance checks
// These will cause compilation errors if the types don't implement the interfaces

var (
	_ TornClientInterface              = (*torn.Client)(nil)
	_ SheetsClientInterface            = (*sheets.Client)(nil)
	_ LocationServiceInterface         = (*travel.LocationService)(nil)
	_ TravelTimeServiceInterface       = (*travel.TravelTimeService)(nil)
	_ AttackProcessingServiceInterface = (*attack.AttackProcessingService)(nil)
	// WarSummaryServiceInterface compliance check moved to application/services package
)
