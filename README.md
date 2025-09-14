# Torn RW Stats

A Go application that monitors Torn faction wars and automatically updates Google Sheets with war statistics and attack records, featuring sophisticated state-based optimization that adapts to Torn's weekly war cycle.

## Features

### War Monitoring & Data Collection
- Monitors active faction wars (ranked, raids, territory wars)
- Automatically creates "Summary - {war_id}" and "Records - {war_id}" sheets
- Updates war summaries with real-time statistics
- Tracks all attacks with detailed records
- Travel status monitoring for both factions
- State change detection and logging

### Sophisticated War State Management
- **4-State War Lifecycle**: NoWars → PreWar → ActiveWar → PostWar
- **Intelligent Scheduling**: Automatically pauses during NoWars/PostWar states until Tuesday 12:05 UTC matchmaking
- **Adaptive Update Intervals**:
  - Active wars: 1-minute real-time monitoring
  - Pre-war reconnaissance: 5-minute opponent monitoring
  - No wars/Post-war: Paused until next matchmaking
- **Tuesday Matchmaking Intelligence**: Precise UTC calculations for Torn's weekly war cycle
- **Priority-Based War Selection**: Handles overlapping/multiple war scenarios correctly

### Performance & Reliability
- API call caching with 45% reduction in API usage
- State transition validation preventing rapid oscillation
- Comprehensive edge case handling for war timing scenarios
- Robust error handling and retry logic
- Structured logging with correlation IDs

## Setup

### Prerequisites

- Go 1.24.2 or later
- Google Sheets API credentials
- Torn API key

### Installation

1. Clone the repository and navigate to the directory
2. Copy the example environment file:
   ```bash
   cp .env.example .env
   ```

3. Edit `.env` and fill in your configuration:
   - `TORN_API_KEY`: Your Torn API key
   - `SPREADSHEET_ID`: The ID of your Google Spreadsheet
   - `GOOGLE_CREDENTIALS_FILE`: Path to your Google service account credentials file

4. Set up Google Sheets API:
   - Go to [Google Cloud Console](https://console.cloud.google.com/)
   - Create a new project or select existing one
   - Enable the Google Sheets API
   - Create a service account and download the JSON credentials file
   - Place the credentials file as `credentials.json` in the project directory
   - Share your spreadsheet with the service account email

5. Build the application:
   ```bash
   go build -o torn_rw_stats
   ```

## Usage

### Basic Usage

Run continuously with default 5-minute intervals:
```bash
./torn_rw_stats
```

### Command Line Options

```bash
./torn_rw_stats [options]

Options:
  -interval duration    Interval between war updates (default 5m0s)
  -once                 Run once and exit (don't start scheduler)
```

### Examples

Run once and exit:
```bash
./torn_rw_stats -once
```

Run with 10-minute intervals:
```bash
./torn_rw_stats -interval=10m
```

## How It Works

### Intelligent War State Detection
1. **War State Analysis**: Fetches current wars from Torn API (`/v2/faction/wars`) and determines current state
2. **Smart Scheduling**: Based on war state, decides whether to process now or wait:
   - **NoWars**: Pauses until next Tuesday 12:05 UTC matchmaking
   - **PreWar**: 5-minute reconnaissance monitoring of opponent faction
   - **ActiveWar**: 1-minute real-time war monitoring with full data collection
   - **PostWar**: Continues monitoring briefly, then pauses until next week

### Data Collection & Processing
3. **Sheet Management**: Creates summary and records sheets for each active war if they don't exist
4. **Attack Collection**: Fetches attacks using optimized time-range queries (`/v2/faction/attacks`)
5. **Travel Status Monitoring**: Tracks both faction members' travel status, locations, and arrival times
6. **Data Processing**: Processes attacks and generates comprehensive statistics
7. **Sheet Updates**: Updates summary, records, and travel status sheets with current data

### Optimization Features
- **API Call Caching**: Reduces redundant API calls by 45% through intelligent caching
- **Priority War Selection**: When multiple wars exist, selects most relevant (active > pre-war > post-war)
- **State Transition Validation**: Prevents rapid oscillation between states
- **Edge Case Handling**: Manages war cancellations, overlaps, and timing edge cases

## Sheet Structure

### Summary Sheet (`Summary - {war_id}`)
- War details (ID, status, start/end times)
- Faction information
- Current scores
- Attack statistics (total, won, lost, win rate)
- Respect statistics (gained, lost, net)
- Last updated timestamp

### Records Sheet (`Records - {war_id}`)
- Detailed attack log with columns:
  - Attack ID, Code, Started/Ended timestamps, Direction
  - Attacker/Defender names, levels, and faction information
  - Result, Respect gain/loss, Chain status
  - Attack modifiers (Fair Fight, War, Retaliation, etc.)
  - Finishing hit effects and values

### Status Sheets (`Status - {faction_id}`)
- Real-time faction member status monitoring:
  - Member name, level, current location, status (travel, hospital, okay, etc.)
  - Departure/arrival times with countdown timers for travel
  - Hospital countdown timers
  - State change detection and logging

## Configuration

All configuration is done via environment variables. See `.env.example` for available options.

### Google Sheets Setup

1. Create a Google Spreadsheet
2. Note the spreadsheet ID from the URL
3. Set up a Google service account with Sheets API access
4. Share the spreadsheet with the service account email
5. Download the service account credentials JSON file

## Deployment

The application is designed to run as a long-running service. You can:

- Run directly on a server
- Deploy as a Docker container
- Run as a systemd service
- Deploy to cloud platforms

## API Usage

### API Endpoints Used

- `/v2/faction/wars` - War detection and state analysis
- `/v2/faction/attacks` - Attack data collection
- `/v2/faction/{id}` - Faction member data and travel status
- `/v2/user/{id}` - Individual user information when needed

### Intelligent API Call Management

- **State-Based Frequency**: API calls adapt to war state (1min active, 5min pre-war, paused otherwise)
- **Smart Caching**: 45% reduction in API calls through intelligent caching of faction and war data
- **Time-Range Optimization**: Only fetches new attack data since last update
- **Tuesday Awareness**: Automatically pauses during no-war periods until Torn's matchmaking schedule

### API Call Efficiency

- **Baseline**: ~4 calls per minute during active monitoring
- **With Optimizations**: ~2.2 calls per minute (45% reduction)
- **During Paused States**: 0 calls (waits for next matchmaking window)
- **Call Tracking**: Comprehensive logging and statistics for API usage monitoring

## Error Handling

The application includes robust error handling:

- Automatic retries for API requests
- Graceful handling of network issues
- Logging of all operations and errors
- Continues processing even if individual wars fail

## Logging

Structured logging using zerolog:

- Debug: Detailed operation logs
- Info: Important events and statistics
- Error: Error conditions and failures

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests if applicable
5. Submit a pull request

## License

[Add your license here]
