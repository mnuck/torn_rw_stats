# Torn RW Stats

A Go application that monitors Torn faction wars and automatically updates Google Sheets with war statistics and attack records.

## Features

- Monitors active faction wars (ranked, raids, territory wars)
- Automatically creates "Summary - {war_id}" and "Records - {war_id}" sheets
- Updates war summaries with real-time statistics
- Tracks all attacks with detailed records
- Configurable update intervals
- Robust error handling and retry logic

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

1. **War Detection**: Fetches current wars from the Torn API (`/v2/faction/wars`)
2. **Sheet Management**: Creates summary and records sheets for each active war if they don't exist
3. **Attack Collection**: Fetches all attacks from the war timeframe (`/v2/faction/attacks`)
4. **Data Processing**: Processes attacks and generates summary statistics
5. **Sheet Updates**: Updates both summary and records sheets with current data

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
  - Attack ID, Timestamp, Direction
  - Attacker/Defender names and levels
  - Result, Respect gain/loss, Chain

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

The application makes API calls to:
- `/v2/faction/wars` - To detect active wars
- `/v2/faction/attacks` - To fetch attack data

API call frequency depends on:
- Number of active wars
- Amount of attack data per war
- Update interval configured

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