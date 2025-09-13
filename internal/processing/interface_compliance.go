package processing

import (
	"torn_rw_stats/internal/sheets"
	"torn_rw_stats/internal/torn"
)

// Compile-time interface compliance checks
// These will cause compilation errors if the types don't implement the interfaces

var (
	_ TornClientInterface                  = (*torn.Client)(nil)
	_ SheetsClientInterface                = (*sheets.Client)(nil)
	_ LocationServiceInterface             = (*LocationService)(nil)
	_ TravelTimeServiceInterface           = (*TravelTimeService)(nil)
	_ AttackProcessingServiceInterface     = (*AttackProcessingService)(nil)
	_ WarSummaryServiceInterface           = (*WarSummaryService)(nil)
	_ StateChangeDetectionServiceInterface = (*StateChangeDetectionService)(nil)
)
