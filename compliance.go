package main

import (
	"torn_rw_stats/internal/application/ports"
	"torn_rw_stats/internal/bigquery"
	"torn_rw_stats/internal/deployment"
	"torn_rw_stats/internal/domain/attack"
	"torn_rw_stats/internal/domain/travel"
	"torn_rw_stats/internal/sheets"
	"torn_rw_stats/internal/torn"
)

// Compile-time checks that the adapters and domain services satisfy the
// application ports. Kept in the composition root so that neither the
// ports package nor the adapters need to import each other.
var (
	_ ports.TornClient              = (*torn.Client)(nil)
	_ ports.SheetsClient            = (*sheets.Client)(nil)
	_ ports.LocationService         = (*travel.LocationService)(nil)
	_ ports.TravelTimeService       = (*travel.TravelTimeService)(nil)
	_ ports.AttackProcessingService = (*attack.AttackProcessingService)(nil)
	_ ports.BigQueryClient          = (*bigquery.Client)(nil)
	_ ports.Deployer                = (*deployment.SSHDeployer)(nil)
)
