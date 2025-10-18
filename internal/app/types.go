package app

import "time"

// War represents a faction war from the API
type War struct {
	ID       int       `json:"war_id"`
	Start    int64     `json:"start"`
	End      *int64    `json:"end"`
	Target   int       `json:"target"`
	Winner   *int      `json:"winner"`
	Factions []Faction `json:"factions"`
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
	Wars struct {
		Ranked    *War  `json:"ranked"`
		Raids     []War `json:"raids"`
		Territory []War `json:"territory"`
	} `json:"wars"`
}

// Attack represents an attack from the API
type Attack struct {
	ID                  int64                `json:"id"`
	Code                string               `json:"code"`
	Started             int64                `json:"started"`
	Ended               int64                `json:"ended"`
	Attacker            User                 `json:"attacker"`
	Defender            User                 `json:"defender"`
	Result              string               `json:"result"`
	RespectGain         float64              `json:"respect_gain"`
	RespectLoss         float64              `json:"respect_loss"`
	Chain               int                  `json:"chain"`
	IsInterrupted       bool                 `json:"is_interrupted"`
	IsStealthed         bool                 `json:"is_stealthed"`
	IsRaid              bool                 `json:"is_raid"`
	IsRankedWar         bool                 `json:"is_ranked_war"`
	Modifiers           AttackModifiers      `json:"modifiers"`
	FinishingHitEffects []FinishingHitEffect `json:"finishing_hit_effects"`
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
	WarID          int
	SummaryTabName string
	RecordsTabName string
	SpreadsheetID  string
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

// FactionInfoResponse represents response from /faction/?selections=basic (own faction)
type FactionInfoResponse struct {
	ID       int                      `json:"ID"`
	Name     string                   `json:"name"`
	Tag      string                   `json:"tag"`
	TagImage string                   `json:"tag_image"`
	Leader   int                      `json:"leader"`
	CoLeader int                      `json:"co-leader"`
	Respect  int                      `json:"respect"`
	Age      int                      `json:"age"`
	Members  map[string]FactionMember `json:"members"`
}

// FactionBasicResponse represents response from /faction/{id}?selections=basic
type FactionBasicResponse struct {
	ID       int                      `json:"ID"`
	Name     string                   `json:"name"`
	Tag      string                   `json:"tag"`
	TagImage string                   `json:"tag_image"`
	Leader   int                      `json:"leader"`
	CoLeader int                      `json:"co-leader"`
	Respect  int                      `json:"respect"`
	Age      int                      `json:"age"`
	Members  map[string]FactionMember `json:"members"`
}

// FactionMember represents a faction member's data
type FactionMember struct {
	Name          string       `json:"name"`
	Level         int          `json:"level"`
	DaysInFaction int          `json:"days_in_faction"`
	LastAction    LastAction   `json:"last_action"`
	Status        MemberStatus `json:"status"`
	Position      string       `json:"position"`
}

// LastAction represents a member's last action
type LastAction struct {
	Status    string `json:"status"`
	Timestamp int64  `json:"timestamp"`
	Relative  string `json:"relative"`
}

// MemberStatus represents a member's current status/location
type MemberStatus struct {
	Description    string `json:"description"`
	State          string `json:"state"`
	Color          string `json:"color"`
	Details        string `json:"details"`
	Until          *int64 `json:"until"`
	TravelType     string `json:"travel_type"`      // For traveling status
	PlaneImageType string `json:"plane_image_type"` // For traveling status
}

// TravelRecord represents a member's travel status record
type TravelRecord struct {
	Name            string `json:"name"`
	Level           int    `json:"level"`
	Location        string `json:"location"`
	State           string `json:"state"`
	Departure       string `json:"departure"`
	Countdown       string `json:"countdown"`
	Arrival         string `json:"arrival"`
	BusinessArrival string `json:"business_arrival"` // Alternative arrival time assuming business class
}

// StateChangeRecord represents a member's state change record
type StateChangeRecord struct {
	Timestamp            time.Time `json:"timestamp"`
	MemberID             int       `json:"member_id"`
	MemberName           string    `json:"member_name"`
	FactionName          string    `json:"faction_name"`
	FactionID            int       `json:"faction_id"`
	LastActionStatus     string    `json:"last_action_status"`
	StatusDescription    string    `json:"status_description"`
	StatusState          string    `json:"status_state"`
	StatusColor          string    `json:"status_color"`
	StatusDetails        string    `json:"status_details"`
	StatusUntil          string    `json:"status_until"`
	StatusTravelType     string    `json:"status_travel_type"`
	StatusPlaneImageType string    `json:"status_plane_image_type"`
	PreviousState        string    `json:"old_state"`
	CurrentState         string    `json:"new_state"`
	PreviousLastAction   string    `json:"old_last_action"`
	CurrentLastAction    string    `json:"new_last_action"`
}

// StateRecord represents a point-in-time snapshot of a member's state
type StateRecord struct {
	Timestamp         time.Time `json:"timestamp"`
	MemberName        string    `json:"member_name"`
	MemberID          string    `json:"member_id"`
	FactionName       string    `json:"faction_name"`
	FactionID         string    `json:"faction_id"`
	LastActionStatus  string    `json:"last_action_status"`
	StatusDescription string    `json:"status_description"`
	StatusState       string    `json:"status_state"`
	StatusUntil       time.Time `json:"status_until"`
	StatusTravelType  string    `json:"status_travel_type"`
}

// StatusV2Record represents a member's data for Status v2 sheets
type StatusV2Record struct {
	Name            string    `json:"name"`
	MemberID        string    `json:"member_id"`
	Level           int       `json:"level"`
	State           string    `json:"state"`            // LastActionStatus from StateRecord
	Status          string    `json:"status"`           // StatusDescription from StateRecord
	Location        string    `json:"location"`         // Destination for traveling, otherwise current location
	Countdown       string    `json:"countdown"`        // Calculated from StatusUntil field
	Departure       string    `json:"departure"`        // Manual adjustment preserved
	Arrival         string    `json:"arrival"`          // Manual adjustment preserved
	BusinessArrival string    `json:"business_arrival"` // Alternative arrival time assuming business class
	Until           time.Time `json:"until"`            // StatusUntil timestamp from StateRecord
}

// JSONMember represents a member in the JSON export format
type JSONMember struct {
	Name            string `json:"Name"`
	MemberID        string `json:"MemberID"`
	Level           int    `json:"Level"`
	State           string `json:"State"`
	Status          string `json:"Status,omitempty"`
	Countdown       string `json:"Countdown,omitempty"`
	Until           string `json:"Until,omitempty"`
	Arrival         string `json:"Arrival,omitempty"`
	BusinessArrival string `json:"BusinessArrival,omitempty"`
}

// LocationData represents the traveling and located members for a location
type LocationData struct {
	Traveling []JSONMember `json:"Traveling"`
	LocatedIn []JSONMember `json:"Located In"`
}

// StatusV2JSON represents the complete JSON export structure
type StatusV2JSON struct {
	Faction   string                  `json:"Faction"`
	Updated   string                  `json:"Updated"`
	Interval  int                     `json:"Interval"` // Update interval in seconds
	Locations map[string]LocationData `json:"Locations"`
}
