package domain

import "time"

// Attack represents a single attack between two players
type Attack struct {
	ID                  int64
	Code                string
	Started             int64
	Ended               int64
	Attacker            User
	Defender            User
	Result              string
	RespectGain         float64
	RespectLoss         float64
	Chain               int
	IsInterrupted       bool
	IsStealthed         bool
	IsRaid              bool
	IsRankedWar         bool
	Modifiers           AttackModifiers
	FinishingHitEffects []FinishingHitEffect
}

// AttackModifiers represents the modifiers applied to an attack
type AttackModifiers struct {
	FairFight   float64
	War         float64
	Retaliation float64
	Group       float64
	Overseas    float64
	Chain       float64
	Warlord     float64
}

// FinishingHitEffect represents a finishing hit effect
type FinishingHitEffect struct {
	Name  string
	Value float64
}

// User represents a user in an attack
type User struct {
	ID      int
	Name    string
	Level   int
	Faction *Faction
}

// AttackRecord represents a single attack for the records sheet
type AttackRecord struct {
	AttackID            int64
	Code                string
	Started             time.Time
	Ended               time.Time
	Direction           string // "Outgoing" or "Incoming"
	AttackerID          int
	AttackerName        string
	AttackerLevel       int
	AttackerFactionID   *int
	AttackerFactionName string
	DefenderID          int
	DefenderName        string
	DefenderLevel       int
	DefenderFactionID   *int
	DefenderFactionName string
	Result              string
	RespectGain         float64
	RespectLoss         float64
	Chain               int
	IsInterrupted       bool
	IsStealthed         bool
	IsRaid              bool
	IsRankedWar         bool
	ModifierFairFight   float64
	ModifierWar         float64
	ModifierRetaliation float64
	ModifierGroup       float64
	ModifierOverseas    float64
	ModifierChain       float64
	ModifierWarlord     float64
	FinishingHitName    string
	FinishingHitValue   float64
}

// RecordsInfo describes the attack records already present in a records
// sheet, used to compute incremental updates.
type RecordsInfo struct {
	AttackCodes      map[string]bool
	LatestTimestamp  int64
	RecordCount      int
	LastRowProcessed int
}
