package main

import (
	"context"
	"flag"
	"time"

	"torn_rw_stats/internal/app"
	"torn_rw_stats/internal/application/services"
	"torn_rw_stats/internal/sheets"
	"torn_rw_stats/internal/torn"

	"github.com/rs/zerolog/log"
)

func main() {
	app.SetupEnvironment()

	// Parse command line flags
	interval := flag.Duration("interval", 5*time.Minute, "Interval between war updates (e.g., 5m, 10m)")
	runOnce := flag.Bool("once", false, "Run once and exit (don't start scheduler)")
	flag.Parse()

	log.Info().
		Dur("interval", *interval).
		Bool("run_once", *runOnce).
		Msg("Starting Torn RW Stats application")

	// Load configuration
	config, err := app.LoadConfig()
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to load configuration")
	}

	// Set the update interval from command line flag
	config.UpdateInterval = *interval

	ctx := context.Background()

	// Initialize clients
	tornClient := torn.NewClient(config.TornAPIKey)
	sheetsClient, err := sheets.NewClient(ctx, config.CredentialsFile)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to create sheets client")
	}

	// Initialize optimized war processor with state-based optimization
	warProcessor := services.NewOptimizedWarProcessorWithConcreteDependencies(tornClient, sheetsClient, config)

	// Define the main processing function that returns next check time
	processWars := func() time.Duration {
		log.Debug().Msg("Starting war processing cycle")

		// Reset API call counter at the start of each cycle
		tornClient.ResetAPICallCount()

		if err := warProcessor.ProcessActiveWars(ctx); err != nil {
			log.Error().Err(err).Msg("Failed to process active wars")
			return *interval // Use CLI interval as fallback on error
		}

		apiCalls := tornClient.GetAPICallCount()

		// Get intelligent next check time from war processor
		nextCheckTime := warProcessor.GetNextCheckTime()
		nextCheckDuration := time.Until(nextCheckTime)

		// Use CLI interval as minimum/fallback
		if nextCheckDuration < time.Minute {
			nextCheckDuration = time.Minute
		}
		if nextCheckDuration > *interval && *interval > 0 {
			nextCheckDuration = *interval
		}

		log.Info().
			Int64("api_calls", apiCalls).
			Dur("next_check_in", nextCheckDuration).
			Msg("Completed war processing cycle")

		return nextCheckDuration
	}

	// Run initial processing
	log.Info().Msg("Running initial war processing")
	nextInterval := processWars()

	// Exit if run-once flag is set
	if *runOnce {
		log.Info().Msg("Run-once mode: exiting after initial processing")
		return
	}

	// Start scheduled processing with dynamic intervals
	log.Info().
		Dur("fallback_interval", *interval).
		Dur("initial_next_check", nextInterval).
		Msg("Starting scheduled war processing with intelligent timing")

	for {
		time.Sleep(nextInterval)
		nextInterval = processWars()
	}
}
