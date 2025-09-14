package sheets

// This file now contains only the legacy wrapper functions that delegate to the new focused managers.
// The actual business logic has been moved to specialized managers for better separation of concerns.
//
// Architecture:
//   - war_manager.go: War sheet creation and summary management
//   - records_processor.go: Attack record processing and conversion
//   - travel_manager.go: Travel status sheet management
//   - state_manager.go: State change tracking and management
//   - wars_legacy.go: Legacy functions for backward compatibility
//
// The original functions have been moved to wars_legacy.go for backward compatibility
// and will be removed in a future version once all callers are updated to use the new managers.
