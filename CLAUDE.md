# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Torn RW Stats is a Go application that monitors faction wars in the Torn browser game and automatically updates Google Sheets with real-time war statistics and attack records. The application continuously polls the Torn API for war data and maintains comprehensive spreadsheets tracking war progress.

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
- War detection, attack aggregation, statistics calculation

**internal/config/**: Utility configurations
- `resilience.go`: Retry and resilience patterns

### Data Flow

1. **War Detection**: Polls `/v2/faction/wars` to find active ranked wars, raids, and territory wars
2. **Sheet Setup**: Creates "Summary - {war_id}" and "Records - {war_id}" sheets if they don't exist
3. **Attack Collection**: Fetches attacks from war timeframe using `/v2/faction/attacks`
4. **Data Processing**: Processes attacks to generate summary statistics and individual attack records
5. **Sheet Updates**: Updates both summary and records sheets with current data

### Key Design Patterns

- **API Rate Limiting**: Built-in API call counting and rate limiting to respect Torn API limits
- **Resilient Processing**: Continues processing other wars even if individual wars fail
- **Incremental Updates**: Only processes new data since last update where possible
- **Structured Logging**: Uses zerolog for comprehensive logging with correlation IDs

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

- Test API integration with `-once` flag for single execution
- Use structured logging to verify data flow through components
- Monitor API call counts to ensure rate limiting works correctly
- Verify sheet creation and updates in Google Sheets interface