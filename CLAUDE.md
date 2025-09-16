# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Torn RW Stats is a Go application that monitors faction wars in the Torn browser game and automatically updates Google Sheets with real-time war statistics and attack records. The application features sophisticated state-based optimization that adapts to Torn's weekly war cycle, reducing API usage by 45% while providing intelligent monitoring based on war phases.

## Build Commands

```bash
# Build the application
go build -o torn_rw_stats

# Run tests (if any exist)
go test ./...

# Run a specific test
go test ./internal/processing -v

# Check for formatting issues
go fmt ./...

# Vet code for potential issues
go vet ./...

# Run with modules tidy
go mod tidy
```

## Running the Application

```bash
# Run continuously with default 5-minute intervals
./torn_rw_stats

# Run once and exit (useful for testing)
./torn_rw_stats -once

# Run with custom interval
./torn_rw_stats -interval=10m
```

## Architecture Overview

### Core Components

The application follows a clean architecture pattern with distinct layers:

**main.go**: Entry point that orchestrates the application flow, handles CLI flags, and sets up the processing loop with configurable intervals.

**internal/app/**: Application layer containing configuration management and core data types
- `config.go`: Environment setup, configuration loading from .env files
- `types.go`: Complete type definitions for Torn API responses (wars, attacks, factions, users)

**internal/torn/**: Torn API client layer
- `client.go`: HTTP client for Torn API with rate limiting, retry logic, and API call counting
- Handles `/v2/faction/wars` and `/v2/faction/attacks` endpoints

**internal/sheets/**: Google Sheets integration layer
- `client.go`: Google Sheets API client initialization
- `wars.go`: Sheet management, creation, and data writing for war summaries and attack records

**internal/processing/**: Business logic layer
- `wars.go`: Core war processing logic that orchestrates data flow from Torn API to Google Sheets
- `optimized_war_processor.go`: State-based optimization wrapper with intelligent scheduling
- `war_state_manager.go`: Sophisticated 4-state war lifecycle management (NoWars, PreWar, ActiveWar, PostWar)
- `cached_torn_client.go`: API call caching reducing usage by 45%
- `api_optimizer.go`: Smart frequency optimization based on war activity patterns
- Various service layers: attack processing, travel tracking, state change detection

**internal/config/**: Utility configurations
- `resilience.go`: Retry and resilience patterns

### Data Flow

#### Intelligent State-Based Processing
1. **War State Analysis**: Fetches `/v2/faction/wars` and determines current state (NoWars/PreWar/ActiveWar/PostWar)
2. **Smart Scheduling Decision**:
   - **NoWars/PostWar**: Pauses until next Tuesday 12:05 UTC matchmaking
   - **PreWar**: 5-minute reconnaissance monitoring of opponent faction
   - **ActiveWar**: 1-minute real-time monitoring with full data collection

#### Data Collection (Active States Only)
3. **Sheet Setup**: Creates "Summary - {war_id}", "Records - {war_id}", and travel status sheets
4. **Attack Collection**: Fetches attacks using optimized time-range queries
5. **Travel Monitoring**: Tracks both faction members' locations, travel status, and arrival times
6. **Data Processing**: Processes attacks, generates statistics, detects state changes
7. **Sheet Updates**: Updates all relevant sheets with current data

### Key Design Patterns

#### State-Based Optimization
- **War State Management**: 4-state lifecycle with intelligent transition validation
- **Tuesday Matchmaking Awareness**: Precise UTC calculations for Torn's weekly war schedule
- **Priority-Based War Selection**: Handles overlapping/multiple war scenarios correctly
- **State Transition Validation**: Prevents rapid oscillation with minimum state duration

#### Performance & Reliability
- **API Call Caching**: 45% reduction through intelligent caching of faction and war data
- **Smart Frequency Adaptation**: API calls adapt to war urgency (1min active, 5min pre-war, paused otherwise)
- **Incremental Updates**: Only fetches new data since last update
- **Resilient Processing**: Continues processing other wars even if individual wars fail
- **Comprehensive Error Handling**: Retry logic, graceful degradation, structured logging with correlation IDs

#### Advanced Features
- **Travel Time Calculations**: Real-time departure/arrival predictions with countdown timers
- **State Change Detection**: Tracks member status changes (hospital, travel, etc.) with normalization
- **Property-Based Testing**: Comprehensive test coverage including edge cases and state transitions

## Configuration

The application requires environment variables set in `.env`:

```bash
TORN_API_KEY=your_torn_api_key
SPREADSHEET_ID=google_sheets_id
GOOGLE_CREDENTIALS_FILE=credentials.json
OUR_FACTION_ID=your_faction_id  # Optional
```

## Development Notes

- The application is designed to run as a long-running service
- All external dependencies (Torn API, Google Sheets) have error handling and retry logic
- The `internal/` directory structure prevents external imports, keeping the API surface clean
- Types in `internal/app/types.go` mirror the Torn API structure exactly for reliable JSON unmarshaling
- Sheet formatting and structure is managed in `internal/sheets/wars.go` with consistent headers and data types

## Testing Strategy

### Unit Testing
- Comprehensive test coverage for war state management and transitions
- Property-based testing for state change detection and normalization
- Mock-based testing for API clients and sheet operations
- Edge case testing for timing scenarios and war overlaps

### Integration Testing
- Test API integration with `-once` flag for single execution
- Use structured logging to verify data flow through components
- Monitor API call counts and optimization effectiveness
- Verify sheet creation and updates in Google Sheets interface

### War State Testing
```bash
# Test specific war state scenarios
go test ./internal/processing -v -run TestWarState
go test ./internal/processing -v -run TestTuesday
go test ./internal/processing -v -run TestEdgeCases

# Test API optimization effectiveness
go test ./internal/processing -v -run TestAPICallEfficiency
```

### Key Test Areas
- **State Transition Logic**: Validates all valid/invalid state transitions
- **Tuesday Matchmaking Calculations**: Tests UTC timezone handling and edge cases
- **Priority War Selection**: Multiple/overlapping war scenario handling
- **API Call Optimization**: Measures actual vs expected API usage reduction
- **Travel Time Calculations**: Location parsing and countdown accuracy

## Release Process

When making changes and preparing for release, follow this standardized process:

### Pre-Release Validation
1. **Run all tests** - ALL tests must pass:
   ```bash
   go test ./...
   ```

2. **Run code analysis** - Fix all issues:
   ```bash
   go vet ./...
   ```

3. **Format code** - Apply standard formatting:
   ```bash
   go fmt ./...
   ```

### Release Workflow
4. **Create feature branch** (if not already on one):
   ```bash
   git checkout -b feature/your-feature-name
   ```

5. **Commit changes** to the feature branch:
   ```bash
   git add .
   git commit -m "Descriptive commit message"
   ```

6. **Push feature branch** to origin:
   ```bash
   git push -u origin feature/your-feature-name
   ```

7. **Create Pull Request** using GitHub CLI or web interface:
   ```bash
   gh pr create --title "PR Title" --body "PR Description"
   ```

8. **Get PR reviewed and merged**:
   - If reviewer has comments, read and address them
   - Work collaboratively to get the PR approved and merged

### Post-Release Cleanup
9. **Switch to main branch**:
   ```bash
   git checkout main
   ```

10. **Pull latest changes** from origin:
    ```bash
    git pull origin main
    ```

11. **Delete feature branch** locally and on origin:
    ```bash
    git branch -d feature/your-feature-name
    git push origin --delete feature/your-feature-name
    ```

### Important Notes
- **Never skip pre-release validation** - All tests, vetting, and formatting must be clean
- **Use descriptive commit messages** that explain the "why" not just the "what"
- **Address reviewer feedback promptly** and work collaboratively
- **Clean up branches** after successful merge to keep repository organized