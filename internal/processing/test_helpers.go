package processing

import (
	"torn_rw_stats/internal/app"
)

// newTestWarProcessor creates a WarProcessor for testing with concrete dependencies
func newTestWarProcessor(config *app.Config) *WarProcessor {
	attackService := NewAttackProcessingService(12345) // Default test faction ID
	summaryService := NewWarSummaryService(attackService)

	return NewWarProcessor(
		nil, // tornClient - not needed for most unit tests
		nil, // sheetsClient - not needed for most unit tests
		NewLocationService(),
		NewTravelTimeService(),
		attackService,
		summaryService,
		config,
	)
}

// newTestWarProcessorWithMocks creates a WarProcessor for testing with mock dependencies
func newTestWarProcessorWithMocks(
	tornClient TornClientInterface,
	sheetsClient SheetsClientInterface,
	config *app.Config,
) *WarProcessor {
	attackService := NewAttackProcessingService(12345) // Default test faction ID
	summaryService := NewWarSummaryService(attackService)

	return NewWarProcessor(
		tornClient,
		sheetsClient,
		NewLocationService(),
		NewTravelTimeService(),
		attackService,
		summaryService,
		config,
	)
}
