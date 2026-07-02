// Package domain holds the core domain model for Torn RW Stats.
// Types here are owned by the application core and carry no knowledge of
// external systems. Adapters (Torn API, Google Sheets, BigQuery) map their
// wire formats to and from these types at the boundary.
package domain

import "time"

// War represents a faction war
type War struct {
	ID       int
	Start    int64
	End      *int64
	Target   int
	Winner   *int
	Factions []Faction
}

// Faction represents a faction participating in a war
type Faction struct {
	ID    int
	Name  string
	Score int
	Chain int
}

// FactionWars holds the current wars a faction is involved in,
// grouped by war type.
type FactionWars struct {
	Ranked    *War
	Raids     []War
	Territory []War
}

// WarSummary represents aggregated war statistics
type WarSummary struct {
	WarID         int
	WarName       string
	StartTime     time.Time
	EndTime       *time.Time
	Status        string
	OurFaction    Faction
	EnemyFaction  Faction
	TotalAttacks  int
	AttacksWon    int
	AttacksLost   int
	RespectGained float64
	RespectLost   float64
	LastUpdated   time.Time
}

// SheetConfig represents configuration for a war's sheets
type SheetConfig struct {
	WarID          int
	SummaryTabName string
	RecordsTabName string
	SpreadsheetID  string
}
