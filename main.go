package main

import (
	"context"
	"flag"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"torn_rw_stats/internal/app"
	"torn_rw_stats/internal/application/services"
	bqclient "torn_rw_stats/internal/bigquery"
	"torn_rw_stats/internal/processing"
	"torn_rw_stats/internal/sheets"
	"torn_rw_stats/internal/torn"
)

const (
	// Default timing constants
	DefaultUpdateInterval = 5 * time.Minute // Default interval between war updates
	MinCheckDuration      = time.Minute     // Minimum time between checks
)

func main() {
	app.SetupEnvironment()

	// Parse command line flags
	interval := flag.Duration("interval", DefaultUpdateInterval, "Interval between war updates (e.g., 5m, 10m)")
	runOnce := flag.Bool("once", false, "Run once and exit (don't start scheduler)")
	flag.Parse()

	slog.Info("Starting Torn RW Stats application",
		"interval", *interval,
		"run_once", *runOnce)

	// Load configuration
	config, err := app.LoadConfig()
	if err != nil {
		slog.Error("Failed to load configuration", "err", err)
		os.Exit(1)
	}

	// Set the update interval from command line flag
	config.UpdateInterval = *interval

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Set up signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		slog.Info("Shutdown signal received, stopping gracefully")
		cancel()
	}()

	// Initialize clients
	tornClient := torn.NewClient(config.TornAPIKey)
	sheetsClient, err := sheets.NewClient(ctx, config.CredentialsFile)
	if err != nil {
		slog.Error("Failed to create sheets client", "err", err)
		os.Exit(1)
	}

	// Optionally initialize BigQuery client (disabled if BIGQUERY_PROJECT_ID is unset)
	var bqClient processing.BigQueryClientInterface
	if config.BigQueryProjectID != "" {
		var bqErr error
		bqClient, bqErr = bqclient.NewClient(ctx, config.CredentialsFile,
			config.BigQueryProjectID, config.BigQueryDatasetID, config.BigQueryTableID)
		if bqErr != nil {
			slog.Error("Failed to create BigQuery client — BigQuery integration disabled", "err", bqErr)
			bqClient = nil
		} else {
			slog.Info("BigQuery client initialized",
				"project", config.BigQueryProjectID,
				"dataset", config.BigQueryDatasetID,
				"table", config.BigQueryTableID)
		}
	}

	// Initialize optimized war processor with state-based optimization
	warProcessor := services.NewOptimizedProcessor(tornClient, sheetsClient, config, bqClient)

	// Define the main processing function that returns next check time
	processWars := func() time.Duration {
		slog.Debug("Starting war processing cycle")

		// Reset API call counter at the start of each cycle
		tornClient.ResetAPICallCount()

		if err := warProcessor.ProcessActiveWars(ctx); err != nil {
			slog.Error("Failed to process active wars", "err", err)
			return *interval // Use CLI interval as fallback on error
		}

		apiCalls := tornClient.GetAPICallCount()

		// Get intelligent next check time from war processor
		nextCheckTime := warProcessor.GetNextCheckTime()
		nextCheckDuration := time.Until(nextCheckTime)

		// Use CLI interval as minimum/fallback
		if nextCheckDuration < MinCheckDuration {
			nextCheckDuration = MinCheckDuration
		}
		if nextCheckDuration > *interval && *interval > 0 {
			nextCheckDuration = *interval
		}

		slog.Info("Completed war processing cycle",
			"api_calls", apiCalls,
			"next_check_in", nextCheckDuration)

		return nextCheckDuration
	}

	// Run initial processing
	slog.Info("Running initial war processing")
	nextInterval := processWars()

	// Exit if run-once flag is set
	if *runOnce {
		slog.Info("Run-once mode: exiting after initial processing")
		return
	}

	// Start scheduled processing with dynamic intervals
	slog.Info("Starting scheduled war processing with intelligent timing",
		"fallback_interval", *interval,
		"initial_next_check", nextInterval)

	ticker := time.NewTicker(nextInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			nextInterval = processWars()
			ticker.Reset(nextInterval)
		case <-ctx.Done():
			slog.Info("Shutting down war processor")
			return
		}
	}
}
