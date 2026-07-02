package domain

// FactionInfo represents a faction's basic information and member roster
type FactionInfo struct {
	ID       int
	Name     string
	Tag      string
	TagImage string
	Leader   int
	CoLeader int
	Respect  int
	Age      int
	Members  map[string]FactionMember
}

// FactionMember represents a faction member's data
type FactionMember struct {
	Name          string
	Level         int
	DaysInFaction int
	LastAction    LastAction
	Status        MemberStatus
	Position      string
}

// LastAction represents a member's last action
type LastAction struct {
	Status    string
	Timestamp int64
	Relative  string
}

// MemberStatus represents a member's current status/location
type MemberStatus struct {
	Description    string
	State          string
	Color          string
	Details        string
	Until          *int64
	TravelType     string // For traveling status
	PlaneImageType string // For traveling status
}
