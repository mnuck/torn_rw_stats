# Functional Core, Imperative Shell Refactoring Plan

## Context

This refactoring applies the "Functional Core, Imperative Shell" pattern documented in [Google Testing Blog](https://testing.googleblog.com/2025/10/simplify-your-code-functional-core.html).

**Goal:** Separate pure business logic (functional core) from side effects (imperative shell) to improve testability, maintainability, and reusability.

## Current Architecture Assessment

### ✅ Already Well-Designed (40% of codebase)

**internal/domain/** layer follows the pattern correctly:
- `internal/domain/attack/attack_service.go` - Pure attack processing
- `internal/domain/travel/location_service.go` - Pure string parsing
- `internal/domain/travel/travel_time_service.go` - Pure time calculations
- `internal/domain/war/war_state_manager.go` - Pure state machine

**internal/torn/client.go** and **internal/sheets/client.go** - Clean infrastructure, no business logic

### ❌ Mixed Layer Problems (40% of codebase)

**internal/application/services/** mixes business decisions with I/O:

#### Problem 1: War Processing Logic
**File:** `internal/application/services/wars.go:135-264`
**Issue:** `processWar()` method embeds decision logic inside I/O operations

```go
// CURRENT: Business logic mixed with I/O
func (wp *WarProcessor) processWar(ctx context.Context, ourFactionID int, war *types.War) error {
    // Decision: full vs incremental
    if wp.fullMode {
        attacks, err = wp.tornClient.GetFactionAttacks(...)  // I/O!
    } else {
        attacks, err = wp.attackProcessor.GetAttacksForTimeRange(...)  // I/O!
    }

    // More decisions embedded in execution
    if err := wp.sheetsClient.EnsureWarSheets(...); err != nil {  // I/O!
        return err
    }
}
```

**Problems:**
- Cannot test war processing logic without mocking Torn API and Sheets
- Decision logic (full vs incremental, time ranges) is buried in orchestration
- Hard to understand "what will happen" without reading I/O code

#### Problem 2: Attack Fetching Strategy
**File:** `internal/torn/processor.go:36-91`
**Issue:** `GetAttacksForTimeRange()` makes strategy decisions then immediately executes I/O

```go
// CURRENT: Strategy mixed with execution
func (ap *AttackProcessor) GetAttacksForTimeRange(
    ctx context.Context,
    ourFactionID int,
    opponentFactionID int,
    startTime time.Time,
    endTime time.Time,
) ([]*types.Attack, error) {
    // Pure decision logic
    if ap.ShouldUseSimpleApproach(startTime, endTime) {
        return ap.fetchAttacksSimple(ctx, ourFactionID, opponentFactionID)  // I/O!
    }
    return ap.fetchAttacksPaginated(ctx, ourFactionID, opponentFactionID, startTime, endTime)  // I/O!
}
```

**Problems:**
- Cannot test pagination strategy without mocking API calls
- Strategy logic (`ShouldUseSimpleApproach`) is already pure but embedded in I/O code
- Cannot reuse strategy decisions in other contexts

#### Problem 3: Status Conversion
**File:** `internal/application/services/status_v2_service.go:23-43`
**Issue:** `ConvertStateRecordsToStatusV2()` fetches data mid-conversion

```go
// CURRENT: Transformation mixed with data retrieval
func (s *StatusV2Service) ConvertStateRecordsToStatusV2(
    ctx context.Context,
    stateRecords []*types.StateRecord,
    war *types.War,
) ([][]interface{}, error) {
    // Fetches data mid-conversion!
    existingData := s.getExistingStatusV2Data(ctx, war.ID)

    // Transformation logic continues...
    for _, record := range stateRecords {
        row := s.convertSingleStateRecord(record, war, existingData)
        result = append(result, row)
    }
    return result, nil
}
```

**Problems:**
- Cannot test conversion logic without mocking Sheets client
- Transformation is inherently pure but requires I/O to test
- Existing data should be an input parameter, not fetched internally

#### Problem 4: State Change Processing
**File:** `internal/application/services/optimized_war_processor.go:155-184`
**Issue:** `processStateChanges()` mixes "which factions to track" with "track them"

```go
// CURRENT: Decision mixed with execution
func (owp *OptimizedWarProcessor) processStateChanges(
    ctx context.Context,
    currentStates map[int]*types.StateRecord,
    previousStates map[int]*types.StateRecord,
) error {
    changes := owp.stateComparator.FindChangedStates(currentStates, previousStates)

    // Decision about which factions to track
    factionsToTrack := make(map[int]bool)
    for _, change := range changes {
        factionsToTrack[change.FactionID] = true
    }

    // Immediately executes tracking
    for factionID := range factionsToTrack {
        owp.trackFaction(ctx, factionID)  // I/O!
    }
}
```

**Problems:**
- Cannot test "which factions to track" logic without executing tracking
- Decision logic is simple but tied to I/O
- Cannot dry-run or preview what would be tracked

## Phased Refactoring Plan

### Phase 1: Extract Pure Decision Functions (Low Risk)

**Objective:** Create pure functional core for decision logic without changing existing behavior.

**Strategy:** Add new files with pure functions alongside existing code. No behavior changes, only code organization.

#### 1.1: War Processing Decisions
**New File:** `internal/domain/war/war_processing_plan.go`

```go
package war

import (
    "time"
    "github.com/mnuck/torn_rw_stats/internal/app/types"
)

// ProcessingPlan describes what to fetch and how to process a war
type ProcessingPlan struct {
    WarID            int
    FetchMode        FetchMode
    AttackTimeRange  TimeRange
    RequiresSheets   bool
    SheetNames       []string
}

type FetchMode string

const (
    FetchModeAll         FetchMode = "all"
    FetchModeIncremental FetchMode = "incremental"
    FetchModeNone        FetchMode = "none"
)

type TimeRange struct {
    Start time.Time
    End   time.Time
}

// DetermineProcessingPlan decides what to fetch and how based on war state
func DetermineProcessingPlan(
    war *types.War,
    fullMode bool,
    lastProcessedTime time.Time,
) ProcessingPlan {
    plan := ProcessingPlan{
        WarID:          war.ID,
        RequiresSheets: true,
    }

    if fullMode {
        plan.FetchMode = FetchModeAll
        plan.AttackTimeRange = TimeRange{
            Start: time.Unix(war.Start, 0),
            End:   time.Now(),
        }
    } else {
        plan.FetchMode = FetchModeIncremental
        plan.AttackTimeRange = TimeRange{
            Start: lastProcessedTime,
            End:   time.Now(),
        }
    }

    plan.SheetNames = []string{
        fmt.Sprintf("Summary - %d", war.ID),
        fmt.Sprintf("Records - %d", war.ID),
        fmt.Sprintf("Status - %d", war.ID),
    }

    return plan
}

// ShouldProcessWar determines if a war needs processing
func ShouldProcessWar(war *types.War, currentTime time.Time) bool {
    warStart := time.Unix(war.Start, 0)
    warEnd := time.Unix(war.End, 0)

    // War must be started
    if currentTime.Before(warStart) {
        return false
    }

    // Skip wars ended more than 1 hour ago
    if currentTime.After(warEnd.Add(1 * time.Hour)) {
        return false
    }

    return true
}
```

**Tests:** `internal/domain/war/war_processing_plan_test.go`
- Test full mode vs incremental mode plans
- Test edge cases (war just started, war just ended, war in progress)
- Test time range calculations
- All testable without mocks

#### 1.2: Attack Fetch Strategy
**New File:** `internal/domain/attack/fetch_strategy.go`

```go
package attack

import (
    "time"
)

// FetchStrategy describes how to fetch attacks
type FetchStrategy struct {
    Method     FetchMethod
    TimeRange  TimeRange
    Pagination PaginationConfig
}

type FetchMethod string

const (
    FetchMethodSimple     FetchMethod = "simple"      // Fetch all at once
    FetchMethodPaginated  FetchMethod = "paginated"   // Fetch with pagination
)

type TimeRange struct {
    Start time.Time
    End   time.Time
}

type PaginationConfig struct {
    Enabled      bool
    MaxPages     int
    StopOnGap    bool
    GapThreshold time.Duration
}

// DetermineFetchStrategy decides how to fetch attacks based on time range
func DetermineFetchStrategy(startTime, endTime time.Time) FetchStrategy {
    duration := endTime.Sub(startTime)

    strategy := FetchStrategy{
        TimeRange: TimeRange{Start: startTime, End: endTime},
    }

    // Use simple approach for time ranges under 1 hour
    if ShouldUseSimpleApproach(startTime, endTime) {
        strategy.Method = FetchMethodSimple
        strategy.Pagination = PaginationConfig{Enabled: false}
    } else {
        strategy.Method = FetchMethodPaginated
        strategy.Pagination = PaginationConfig{
            Enabled:      true,
            MaxPages:     100,
            StopOnGap:    true,
            GapThreshold: 5 * time.Minute,
        }
    }

    return strategy
}

// ShouldUseSimpleApproach determines if simple fetching is appropriate
// Extracted from torn/processor.go but made pure
func ShouldUseSimpleApproach(startTime, endTime time.Time) bool {
    duration := endTime.Sub(startTime)

    // For time ranges under 1 hour, use simple approach
    if duration < time.Hour {
        return true
    }

    // For time ranges over 24 hours, must use pagination
    if duration > 24*time.Hour {
        return false
    }

    // For intermediate ranges, use simple approach
    return true
}

// EstimateAPICallsRequired estimates how many API calls will be needed
func EstimateAPICallsRequired(strategy FetchStrategy) int {
    switch strategy.Method {
    case FetchMethodSimple:
        return 1
    case FetchMethodPaginated:
        // Conservative estimate based on typical war activity
        duration := strategy.TimeRange.End.Sub(strategy.TimeRange.Start)
        hoursInRange := int(duration.Hours())
        // Estimate ~10 attacks per hour, 100 attacks per page
        estimatedPages := (hoursInRange * 10) / 100
        if estimatedPages < 1 {
            estimatedPages = 1
        }
        return estimatedPages
    default:
        return 0
    }
}
```

**Tests:** `internal/domain/attack/fetch_strategy_test.go`
- Test strategy selection for various time ranges
- Test API call estimation
- Test edge cases (zero duration, very long duration)
- All testable without mocks

#### 1.3: Status Conversion
**New File:** `internal/domain/status/status_converter.go`

```go
package status

import (
    "github.com/mnuck/torn_rw_stats/internal/app/types"
)

// StatusRow represents a single row in the status sheet
type StatusRow struct {
    MemberID   int
    Name       string
    Level      int
    Status     string
    Location   string
    // ... other fields
}

// ConversionInput contains all data needed for conversion
type ConversionInput struct {
    StateRecords  []*types.StateRecord
    ExistingData  map[int]StatusRow  // Keyed by member ID
    War           *types.War
}

// ConvertToStatusV2 converts state records to sheet rows (pure function)
func ConvertToStatusV2(input ConversionInput) [][]interface{} {
    result := make([][]interface{}, 0, len(input.StateRecords))

    for _, record := range input.StateRecords {
        row := convertSingleRecord(record, input.War, input.ExistingData)
        result = append(result, row)
    }

    return result
}

// convertSingleRecord converts one state record (pure function)
func convertSingleRecord(
    record *types.StateRecord,
    war *types.War,
    existingData map[int]StatusRow,
) []interface{} {
    // Check for existing data
    existing, hasExisting := existingData[record.MemberID]

    row := make([]interface{}, 0, 10)
    row = append(row, record.MemberID)
    row = append(row, record.Name)
    row = append(row, record.Level)

    // Use existing data if available for unchanged fields
    if hasExisting && record.Status == existing.Status {
        row = append(row, existing.Status)
    } else {
        row = append(row, record.Status)
    }

    // ... rest of conversion logic

    return row
}

// ParseExistingStatusData converts raw sheet data to structured format (pure function)
func ParseExistingStatusData(rawData [][]interface{}) map[int]StatusRow {
    result := make(map[int]StatusRow)

    for _, row := range rawData {
        if len(row) < 3 {
            continue
        }

        memberID, ok := row[0].(int)
        if !ok {
            continue
        }

        result[memberID] = StatusRow{
            MemberID: memberID,
            Name:     toString(row[1]),
            Level:    toInt(row[2]),
            // ... parse other fields
        }
    }

    return result
}

func toString(v interface{}) string {
    if s, ok := v.(string); ok {
        return s
    }
    return ""
}

func toInt(v interface{}) int {
    if i, ok := v.(int); ok {
        return i
    }
    return 0
}
```

**Tests:** `internal/domain/status/status_converter_test.go`
- Test conversion with no existing data
- Test conversion with existing data (merge logic)
- Test parsing of various raw data formats
- Test edge cases (empty data, malformed data)
- All testable without mocks

#### 1.4: State Change Analysis
**New File:** `internal/domain/state/state_change_analyzer.go`

```go
package state

import (
    "github.com/mnuck/torn_rw_stats/internal/app/types"
)

// TrackingPlan describes which factions should be tracked
type TrackingPlan struct {
    FactionsToTrack []int
    Reason          map[int]string  // Why each faction should be tracked
}

// DetermineFactionsToTrack decides which factions need tracking based on state changes
func DetermineFactionsToTrack(
    changes []*types.StateChange,
    currentStates map[int]*types.StateRecord,
) TrackingPlan {
    plan := TrackingPlan{
        FactionsToTrack: make([]int, 0),
        Reason:          make(map[int]string),
    }

    factionsSeen := make(map[int]bool)

    for _, change := range changes {
        if factionsSeen[change.FactionID] {
            continue
        }

        // Track factions with significant state changes
        if isSignificantChange(change) {
            plan.FactionsToTrack = append(plan.FactionsToTrack, change.FactionID)
            plan.Reason[change.FactionID] = change.Description
            factionsSeen[change.FactionID] = true
        }
    }

    return plan
}

// isSignificantChange determines if a state change warrants tracking
func isSignificantChange(change *types.StateChange) bool {
    // Track hospital admissions
    if change.NewState.Status == "Hospital" {
        return true
    }

    // Track travel departures
    if change.NewState.Status == "Traveling" {
        return true
    }

    // Track federal jail
    if change.NewState.Status == "Federal" {
        return true
    }

    return false
}
```

**Tests:** `internal/domain/state/state_change_analyzer_test.go`
- Test tracking plan generation
- Test significance determination
- Test deduplication of factions
- All testable without mocks

### Phase 2: Refactor Application Services (Medium Risk)

**Objective:** Modify orchestration layer to use pure decision functions from Phase 1.

**Strategy:** Change one service at a time, verify tests pass, commit incrementally.

#### 2.1: Refactor WarProcessor
**File:** `internal/application/services/wars.go`

**Before:**
```go
func (wp *WarProcessor) processWar(ctx context.Context, ourFactionID int, war *types.War) error {
    if wp.fullMode {
        attacks, err = wp.tornClient.GetFactionAttacks(...)
    } else {
        attacks, err = wp.attackProcessor.GetAttacksForTimeRange(...)
    }
    // ... more mixed logic
}
```

**After:**
```go
func (wp *WarProcessor) processWar(ctx context.Context, ourFactionID int, war *types.War) error {
    // Functional core: Determine plan
    plan := war_domain.DetermineProcessingPlan(war, wp.fullMode, wp.lastProcessedTime)

    // Imperative shell: Execute plan
    attacks, err := wp.executeAttackFetch(ctx, ourFactionID, plan)
    if err != nil {
        return fmt.Errorf("fetch attacks: %w", err)
    }

    sheets, err := wp.ensureSheets(ctx, plan)
    if err != nil {
        return fmt.Errorf("ensure sheets: %w", err)
    }

    err = wp.writeResults(ctx, attacks, sheets, plan)
    if err != nil {
        return fmt.Errorf("write results: %w", err)
    }

    return nil
}

// New helper methods (imperative shell)
func (wp *WarProcessor) executeAttackFetch(
    ctx context.Context,
    ourFactionID int,
    plan war_domain.ProcessingPlan,
) ([]*types.Attack, error) {
    switch plan.FetchMode {
    case war_domain.FetchModeAll:
        return wp.tornClient.GetFactionAttacks(ctx, ourFactionID)
    case war_domain.FetchModeIncremental:
        return wp.attackProcessor.GetAttacksForTimeRange(
            ctx,
            ourFactionID,
            plan.AttackTimeRange.Start,
            plan.AttackTimeRange.End,
        )
    default:
        return nil, nil
    }
}

func (wp *WarProcessor) ensureSheets(
    ctx context.Context,
    plan war_domain.ProcessingPlan,
) (*sheets.WarSheets, error) {
    return wp.sheetsClient.EnsureWarSheets(ctx, plan.WarID, plan.SheetNames)
}

func (wp *WarProcessor) writeResults(
    ctx context.Context,
    attacks []*types.Attack,
    sheets *sheets.WarSheets,
    plan war_domain.ProcessingPlan,
) error {
    // Write logic stays the same, just cleaner separation
    return wp.sheetsClient.UpdateWarSummary(ctx, sheets, attacks)
}
```

**Changes:**
- Extract pure planning logic to `war_domain.DetermineProcessingPlan()`
- Imperative shell executes the plan step-by-step
- Clear separation: decision vs execution
- Same behavior, better structure

**Tests:**
- Existing integration tests should still pass
- Add unit tests for new helper methods
- Verify API call counts unchanged

#### 2.2: Refactor AttackProcessor
**File:** `internal/torn/processor.go`

**Before:**
```go
func (ap *AttackProcessor) GetAttacksForTimeRange(...) ([]*types.Attack, error) {
    if ap.ShouldUseSimpleApproach(startTime, endTime) {
        return ap.fetchAttacksSimple(ctx, ourFactionID, opponentFactionID)
    }
    return ap.fetchAttacksPaginated(ctx, ourFactionID, opponentFactionID, startTime, endTime)
}
```

**After:**
```go
func (ap *AttackProcessor) GetAttacksForTimeRange(
    ctx context.Context,
    ourFactionID int,
    opponentFactionID int,
    startTime time.Time,
    endTime time.Time,
) ([]*types.Attack, error) {
    // Functional core: Determine strategy
    strategy := attack_domain.DetermineFetchStrategy(startTime, endTime)

    // Log estimated API calls
    estimatedCalls := attack_domain.EstimateAPICallsRequired(strategy)
    log.Printf("Fetch strategy: %s (estimated %d API calls)", strategy.Method, estimatedCalls)

    // Imperative shell: Execute strategy
    return ap.executeFetchStrategy(ctx, ourFactionID, opponentFactionID, strategy)
}

// New helper method (imperative shell)
func (ap *AttackProcessor) executeFetchStrategy(
    ctx context.Context,
    ourFactionID int,
    opponentFactionID int,
    strategy attack_domain.FetchStrategy,
) ([]*types.Attack, error) {
    switch strategy.Method {
    case attack_domain.FetchMethodSimple:
        return ap.fetchAttacksSimple(ctx, ourFactionID, opponentFactionID)
    case attack_domain.FetchMethodPaginated:
        return ap.fetchAttacksPaginated(
            ctx,
            ourFactionID,
            opponentFactionID,
            strategy.TimeRange.Start,
            strategy.TimeRange.End,
        )
    default:
        return nil, fmt.Errorf("unknown fetch method: %s", strategy.Method)
    }
}
```

**Changes:**
- Extract strategy decision to `attack_domain.DetermineFetchStrategy()`
- Add API call estimation for better observability
- Imperative shell executes the strategy
- Same behavior, testable strategy logic

**Tests:**
- Existing tests should pass
- Add unit tests for strategy execution
- Verify pagination logic unchanged

#### 2.3: Refactor StatusV2Service
**File:** `internal/application/services/status_v2_service.go`

**Before:**
```go
func (s *StatusV2Service) ConvertStateRecordsToStatusV2(
    ctx context.Context,
    stateRecords []*types.StateRecord,
    war *types.War,
) ([][]interface{}, error) {
    // Fetches data mid-conversion!
    existingData := s.getExistingStatusV2Data(ctx, war.ID)

    // Transformation logic
    result := make([][]interface{}, 0)
    for _, record := range stateRecords {
        row := s.convertSingleStateRecord(record, war, existingData)
        result = append(result, row)
    }
    return result, nil
}
```

**After:**
```go
func (s *StatusV2Service) ConvertStateRecordsToStatusV2(
    ctx context.Context,
    stateRecords []*types.StateRecord,
    war *types.War,
) ([][]interface{}, error) {
    // Imperative shell: Fetch existing data
    rawData, err := s.sheetsClient.GetStatusV2Data(ctx, war.ID)
    if err != nil {
        return nil, fmt.Errorf("fetch existing data: %w", err)
    }

    // Functional core: Parse existing data
    existingData := status_domain.ParseExistingStatusData(rawData)

    // Functional core: Convert with existing data
    input := status_domain.ConversionInput{
        StateRecords: stateRecords,
        ExistingData: existingData,
        War:          war,
    }

    return status_domain.ConvertToStatusV2(input), nil
}
```

**Changes:**
- Fetch existing data in imperative shell first
- Pass existing data as parameter to pure function
- Conversion logic now completely testable
- Same behavior, better structure

**Tests:**
- Existing integration tests should pass
- Add comprehensive unit tests for conversion logic
- Test merge behavior with various existing data states

#### 2.4: Refactor OptimizedWarProcessor
**File:** `internal/application/services/optimized_war_processor.go`

**Before:**
```go
func (owp *OptimizedWarProcessor) processStateChanges(
    ctx context.Context,
    currentStates map[int]*types.StateRecord,
    previousStates map[int]*types.StateRecord,
) error {
    changes := owp.stateComparator.FindChangedStates(currentStates, previousStates)

    factionsToTrack := make(map[int]bool)
    for _, change := range changes {
        factionsToTrack[change.FactionID] = true
    }

    for factionID := range factionsToTrack {
        owp.trackFaction(ctx, factionID)
    }
}
```

**After:**
```go
func (owp *OptimizedWarProcessor) processStateChanges(
    ctx context.Context,
    currentStates map[int]*types.StateRecord,
    previousStates map[int]*types.StateRecord,
) error {
    // Already pure: Find changes
    changes := owp.stateComparator.FindChangedStates(currentStates, previousStates)

    // Functional core: Determine tracking plan
    plan := state_domain.DetermineFactionsToTrack(changes, currentStates)

    // Log plan for observability
    log.Printf("State change tracking plan: %d factions", len(plan.FactionsToTrack))
    for _, factionID := range plan.FactionsToTrack {
        log.Printf("  - Faction %d: %s", factionID, plan.Reason[factionID])
    }

    // Imperative shell: Execute tracking
    return owp.executeTrackingPlan(ctx, plan)
}

// New helper method (imperative shell)
func (owp *OptimizedWarProcessor) executeTrackingPlan(
    ctx context.Context,
    plan state_domain.TrackingPlan,
) error {
    for _, factionID := range plan.FactionsToTrack {
        if err := owp.trackFaction(ctx, factionID); err != nil {
            log.Printf("Failed to track faction %d: %v", factionID, err)
            // Continue tracking other factions
        }
    }
    return nil
}
```

**Changes:**
- Extract tracking decision to `state_domain.DetermineFactionsToTrack()`
- Add logging for better observability
- Imperative shell executes tracking plan
- Same behavior, testable decision logic

**Tests:**
- Existing tests should pass
- Add unit tests for tracking plan generation
- Test significance determination with various state changes

### Phase 3: Expand Test Coverage (High Value)

**Objective:** Add comprehensive tests for all pure decision logic.

**Strategy:** Focus on edge cases, boundary conditions, and complex scenarios that are now easily testable.

#### 3.1: War Processing Plan Tests
**File:** `internal/domain/war/war_processing_plan_test.go`

```go
func TestDetermineProcessingPlan(t *testing.T) {
    tests := []struct {
        name              string
        war               *types.War
        fullMode          bool
        lastProcessed     time.Time
        expectedFetchMode war.FetchMode
        expectedTimeRange war.TimeRange
    }{
        {
            name: "full mode fetches all attacks",
            war: &types.War{
                ID:    12345,
                Start: time.Now().Add(-2 * time.Hour).Unix(),
                End:   time.Now().Add(1 * time.Hour).Unix(),
            },
            fullMode:          true,
            lastProcessed:     time.Now().Add(-30 * time.Minute),
            expectedFetchMode: war.FetchModeAll,
        },
        {
            name: "incremental mode fetches since last processed",
            war: &types.War{
                ID:    12345,
                Start: time.Now().Add(-2 * time.Hour).Unix(),
                End:   time.Now().Add(1 * time.Hour).Unix(),
            },
            fullMode:          false,
            lastProcessed:     time.Now().Add(-30 * time.Minute),
            expectedFetchMode: war.FetchModeIncremental,
        },
        // Add more test cases for edge conditions
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            plan := war.DetermineProcessingPlan(tt.war, tt.fullMode, tt.lastProcessed)

            if plan.FetchMode != tt.expectedFetchMode {
                t.Errorf("expected fetch mode %s, got %s", tt.expectedFetchMode, plan.FetchMode)
            }

            // Verify time range
            // Verify sheet names
            // etc.
        })
    }
}

func TestShouldProcessWar(t *testing.T) {
    now := time.Now()

    tests := []struct {
        name     string
        war      *types.War
        expected bool
    }{
        {
            name: "war in progress should be processed",
            war: &types.War{
                Start: now.Add(-1 * time.Hour).Unix(),
                End:   now.Add(1 * time.Hour).Unix(),
            },
            expected: true,
        },
        {
            name: "war not yet started should not be processed",
            war: &types.War{
                Start: now.Add(1 * time.Hour).Unix(),
                End:   now.Add(3 * time.Hour).Unix(),
            },
            expected: false,
        },
        {
            name: "war ended more than 1 hour ago should not be processed",
            war: &types.War{
                Start: now.Add(-5 * time.Hour).Unix(),
                End:   now.Add(-2 * time.Hour).Unix(),
            },
            expected: false,
        },
        {
            name: "war just ended (within 1 hour) should be processed",
            war: &types.War{
                Start: now.Add(-3 * time.Hour).Unix(),
                End:   now.Add(-30 * time.Minute).Unix(),
            },
            expected: true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result := war.ShouldProcessWar(tt.war, now)
            if result != tt.expected {
                t.Errorf("expected %v, got %v", tt.expected, result)
            }
        })
    }
}
```

#### 3.2: Attack Fetch Strategy Tests
**File:** `internal/domain/attack/fetch_strategy_test.go`

```go
func TestDetermineFetchStrategy(t *testing.T) {
    now := time.Now()

    tests := []struct {
        name           string
        startTime      time.Time
        endTime        time.Time
        expectedMethod attack.FetchMethod
    }{
        {
            name:           "30 minute range uses simple fetch",
            startTime:      now.Add(-30 * time.Minute),
            endTime:        now,
            expectedMethod: attack.FetchMethodSimple,
        },
        {
            name:           "2 hour range uses paginated fetch",
            startTime:      now.Add(-2 * time.Hour),
            endTime:        now,
            expectedMethod: attack.FetchMethodPaginated,
        },
        {
            name:           "24 hour range uses paginated fetch",
            startTime:      now.Add(-24 * time.Hour),
            endTime:        now,
            expectedMethod: attack.FetchMethodPaginated,
        },
        {
            name:           "1 week range uses paginated fetch",
            startTime:      now.Add(-7 * 24 * time.Hour),
            endTime:        now,
            expectedMethod: attack.FetchMethodPaginated,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            strategy := attack.DetermineFetchStrategy(tt.startTime, tt.endTime)

            if strategy.Method != tt.expectedMethod {
                t.Errorf("expected method %s, got %s", tt.expectedMethod, strategy.Method)
            }
        })
    }
}

func TestEstimateAPICallsRequired(t *testing.T) {
    now := time.Now()

    tests := []struct {
        name             string
        strategy         attack.FetchStrategy
        expectedMinCalls int
        expectedMaxCalls int
    }{
        {
            name: "simple fetch requires 1 call",
            strategy: attack.FetchStrategy{
                Method: attack.FetchMethodSimple,
            },
            expectedMinCalls: 1,
            expectedMaxCalls: 1,
        },
        {
            name: "1 hour paginated fetch estimates low calls",
            strategy: attack.FetchStrategy{
                Method: attack.FetchMethodPaginated,
                TimeRange: attack.TimeRange{
                    Start: now.Add(-1 * time.Hour),
                    End:   now,
                },
            },
            expectedMinCalls: 1,
            expectedMaxCalls: 2,
        },
        // Add more test cases
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            calls := attack.EstimateAPICallsRequired(tt.strategy)

            if calls < tt.expectedMinCalls || calls > tt.expectedMaxCalls {
                t.Errorf("expected %d-%d calls, got %d", tt.expectedMinCalls, tt.expectedMaxCalls, calls)
            }
        })
    }
}
```

#### 3.3: Status Conversion Tests
**File:** `internal/domain/status/status_converter_test.go`

```go
func TestConvertToStatusV2(t *testing.T) {
    tests := []struct {
        name          string
        input         status.ConversionInput
        expectedRows  int
        expectedData  map[int][]interface{}  // Expected data for specific member IDs
    }{
        {
            name: "converts basic state records",
            input: status.ConversionInput{
                StateRecords: []*types.StateRecord{
                    {MemberID: 1, Name: "Player1", Level: 50, Status: "Okay"},
                    {MemberID: 2, Name: "Player2", Level: 60, Status: "Hospital"},
                },
                ExistingData: make(map[int]status.StatusRow),
                War:          &types.War{ID: 12345},
            },
            expectedRows: 2,
        },
        {
            name: "merges with existing data",
            input: status.ConversionInput{
                StateRecords: []*types.StateRecord{
                    {MemberID: 1, Name: "Player1", Level: 50, Status: "Okay"},
                },
                ExistingData: map[int]status.StatusRow{
                    1: {MemberID: 1, Name: "Player1", Level: 50, Status: "Okay"},
                },
                War: &types.War{ID: 12345},
            },
            expectedRows: 1,
        },
        // Add more test cases for edge conditions
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result := status.ConvertToStatusV2(tt.input)

            if len(result) != tt.expectedRows {
                t.Errorf("expected %d rows, got %d", tt.expectedRows, len(result))
            }

            // Verify specific data if provided
            // ...
        })
    }
}

func TestParseExistingStatusData(t *testing.T) {
    tests := []struct {
        name         string
        rawData      [][]interface{}
        expectedSize int
        expectedData map[int]status.StatusRow
    }{
        {
            name: "parses valid data",
            rawData: [][]interface{}{
                {1, "Player1", 50, "Okay"},
                {2, "Player2", 60, "Hospital"},
            },
            expectedSize: 2,
        },
        {
            name: "handles malformed rows",
            rawData: [][]interface{}{
                {1, "Player1", 50, "Okay"},
                {"invalid"},  // Malformed
                {2, "Player2", 60, "Hospital"},
            },
            expectedSize: 2,  // Should skip malformed row
        },
        // Add more test cases
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result := status.ParseExistingStatusData(tt.rawData)

            if len(result) != tt.expectedSize {
                t.Errorf("expected %d entries, got %d", tt.expectedSize, len(result))
            }
        })
    }
}
```

#### 3.4: State Change Analysis Tests
**File:** `internal/domain/state/state_change_analyzer_test.go`

```go
func TestDetermineFactionsToTrack(t *testing.T) {
    tests := []struct {
        name              string
        changes           []*types.StateChange
        currentStates     map[int]*types.StateRecord
        expectedFactions  []int
        expectedReasons   map[int]string
    }{
        {
            name: "tracks hospital admissions",
            changes: []*types.StateChange{
                {
                    FactionID:   100,
                    MemberID:    1,
                    Description: "hospitalized",
                    NewState:    &types.StateRecord{Status: "Hospital"},
                },
            },
            currentStates:    make(map[int]*types.StateRecord),
            expectedFactions: []int{100},
        },
        {
            name: "tracks travel departures",
            changes: []*types.StateChange{
                {
                    FactionID:   100,
                    MemberID:    1,
                    Description: "started traveling",
                    NewState:    &types.StateRecord{Status: "Traveling"},
                },
            },
            currentStates:    make(map[int]*types.StateRecord),
            expectedFactions: []int{100},
        },
        {
            name: "deduplicates factions",
            changes: []*types.StateChange{
                {FactionID: 100, NewState: &types.StateRecord{Status: "Hospital"}},
                {FactionID: 100, NewState: &types.StateRecord{Status: "Traveling"}},
            },
            currentStates:    make(map[int]*types.StateRecord),
            expectedFactions: []int{100},  // Should only appear once
        },
        {
            name: "ignores insignificant changes",
            changes: []*types.StateChange{
                {
                    FactionID:   100,
                    MemberID:    1,
                    Description: "leveled up",
                    NewState:    &types.StateRecord{Status: "Okay"},
                },
            },
            currentStates:    make(map[int]*types.StateRecord),
            expectedFactions: []int{},  // Should not track
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            plan := state.DetermineFactionsToTrack(tt.changes, tt.currentStates)

            if len(plan.FactionsToTrack) != len(tt.expectedFactions) {
                t.Errorf("expected %d factions, got %d", len(tt.expectedFactions), len(plan.FactionsToTrack))
            }

            // Verify specific factions
            for _, expectedID := range tt.expectedFactions {
                found := false
                for _, actualID := range plan.FactionsToTrack {
                    if actualID == expectedID {
                        found = true
                        break
                    }
                }
                if !found {
                    t.Errorf("expected faction %d not found in tracking plan", expectedID)
                }
            }
        })
    }
}
```

## Implementation Workflow

### Per-Phase Process

For each phase:

1. **Create Feature Branch**
   ```bash
   git checkout -b feature/functional-core-phase-N
   ```

2. **Implement Changes**
   - Follow the plan for that specific phase
   - Make incremental commits
   - Run tests frequently

3. **Write/Update Tests**
   - Add new unit tests for pure functions
   - Verify existing integration tests still pass
   - Run full test suite

4. **Validate Code Quality**
   ```bash
   go mod tidy
   go test ./...
   go vet ./...
   go fmt ./...
   golangci-lint run
   ```

5. **Commit and Push**
   ```bash
   git add .
   git commit -m "Phase N: [description]"
   git push -u origin feature/functional-core-phase-N
   ```

6. **Create Pull Request**
   ```bash
   gh pr create --title "Phase N: Functional Core Refactoring" --body "..."
   ```

7. **Wait for Review**
   - Address reviewer feedback
   - Make requested changes
   - Get approval and merge

8. **Clean Up**
   ```bash
   git checkout main
   git pull origin main
   git branch -d feature/functional-core-phase-N
   git push origin --delete feature/functional-core-phase-N
   ```

### Context Reset Between Phases

After each phase PR is merged:
1. User will reset Claude's context window
2. Claude reads this REFACTORING_PLAN.md to resume
3. Continue with next phase

## Success Metrics

### Phase 1 Success
- ✅ All new domain packages compile and pass tests
- ✅ No changes to existing behavior
- ✅ Pure functions have 100% test coverage
- ✅ Zero dependencies on external services in domain layer

### Phase 2 Success
- ✅ All existing tests still pass
- ✅ Application services use new domain functions
- ✅ Clear separation of planning vs execution
- ✅ Improved logging and observability

### Phase 3 Success
- ✅ Comprehensive test coverage for all decision logic
- ✅ Edge cases and boundary conditions tested
- ✅ Complex scenarios testable without mocks
- ✅ Tests document expected behavior

### Overall Success
- ✅ Business logic testable without external dependencies
- ✅ Clear separation of concerns
- ✅ Improved maintainability and understanding
- ✅ Easier to add new features
- ✅ Better debugging and observability

## Notes for Future Context Windows

When resuming after context reset:

1. **Read this file first** to understand overall plan
2. **Check git status** to see which phase was completed
3. **Review recent commits** to understand what changed
4. **Run tests** to verify current state
5. **Continue with next phase** following the plan

Key files to reference:
- This file: `REFACTORING_PLAN.md`
- Architecture docs: `CLAUDE.md`
- Test results: Output of `go test ./...`
- Recent commits: `git log --oneline -10`
