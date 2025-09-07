package app

import "time"

// War represents a faction war from the API
type War struct {
	ID       int        `json:"war_id"`
	Start    int64      `json:"start"`
	End      *int64     `json:"end"`
	Target   int        `json:"target"`
	Winner   *int       `json:"winner"`
	Factions []Faction  `json:"factions"`
}

// Faction represents a faction participating in a war
type Faction struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Score int    `json:"score"`
	Chain int    `json:"chain"`
}

// WarResponse represents the response from /v2/faction/wars
type WarResponse struct {
	Pacts []interface{} `json:"pacts"`
	Wars  struct {
		Ranked    *War   `json:"ranked"`
		Raids     []War  `json:"raids"`
		Territory []War  `json:"territory"`
	} `json:"wars"`
}

// Attack represents an attack from the API
type Attack struct {
	ID                  int64                   `json:"id"`
	Code                string                  `json:"code"`
	Started             int64                   `json:"started"`
	Ended               int64                   `json:"ended"`
	Attacker            User                    `json:"attacker"`
	Defender            User                    `json:"defender"`
	Result              string                  `json:"result"`
	RespectGain         float64                 `json:"respect_gain"`
	RespectLoss         float64                 `json:"respect_loss"`
	Chain               int                     `json:"chain"`
	IsInterrupted       bool                    `json:"is_interrupted"`
	IsStealthed         bool                    `json:"is_stealthed"`
	IsRaid              bool                    `json:"is_raid"`
	IsRankedWar         bool                    `json:"is_ranked_war"`
	Modifiers           AttackModifiers         `json:"modifiers"`
	FinishingHitEffects []FinishingHitEffect    `json:"finishing_hit_effects"`
}

// AttackModifiers represents the modifiers applied to an attack
type AttackModifiers struct {
	FairFight   float64 `json:"fair_fight"`
	War         float64 `json:"war"`
	Retaliation float64 `json:"retaliation"`
	Group       float64 `json:"group"`
	Overseas    float64 `json:"overseas"`
	Chain       float64 `json:"chain"`
	Warlord     float64 `json:"warlord"`
}

// FinishingHitEffect represents a finishing hit effect
type FinishingHitEffect struct {
	Name  string  `json:"name"`
	Value float64 `json:"value"`
}

// User represents a user in an attack
type User struct {
	ID      int      `json:"id"`
	Name    string   `json:"name"`
	Level   int      `json:"level"`
	Faction *Faction `json:"faction"`
}

// AttackResponse represents the response from /v2/faction/attacks
type AttackResponse struct {
	Attacks []Attack `json:"attacks"`
}

// SheetConfig represents configuration for a war's sheets
type SheetConfig struct {
	WarID           int
	SummaryTabName  string
	RecordsTabName  string
	SpreadsheetID   string
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

// AttackRecord represents a single attack for the records sheet
type AttackRecord struct {
	AttackID              int64
	Code                  string
	Started               time.Time
	Ended                 time.Time
	Direction             string // "Outgoing" or "Incoming"
	AttackerID            int
	AttackerName          string
	AttackerLevel         int
	AttackerFactionID     *int
	AttackerFactionName   string
	DefenderID            int
	DefenderName          string
	DefenderLevel         int
	DefenderFactionID     *int
	DefenderFactionName   string
	Result                string
	RespectGain           float64
	RespectLoss           float64
	Chain                 int
	IsInterrupted         bool
	IsStealthed           bool
	IsRaid                bool
	IsRankedWar           bool
	ModifierFairFight     float64
	ModifierWar           float64
	ModifierRetaliation   float64
	ModifierGroup         float64
	ModifierOverseas      float64
	ModifierChain         float64
	ModifierWarlord       float64
	FinishingHitName      string
	FinishingHitValue     float64
}