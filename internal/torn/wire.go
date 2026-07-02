package torn

import "torn_rw_stats/internal/domain"

// This file defines the wire-format types that mirror the Torn API JSON
// responses exactly, plus the mapping functions that convert them into
// domain types. Only this adapter knows the Torn API's shape; the rest of
// the application works with domain types.

// warDTO mirrors a war object from /v2/faction/wars
type warDTO struct {
	ID       int          `json:"war_id"`
	Start    int64        `json:"start"`
	End      *int64       `json:"end"`
	Target   int          `json:"target"`
	Winner   *int         `json:"winner"`
	Factions []factionDTO `json:"factions"`
}

// factionDTO mirrors a war-participant faction
type factionDTO struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Score int    `json:"score"`
	Chain int    `json:"chain"`
}

// warsResponseDTO mirrors the response from /v2/faction/wars
type warsResponseDTO struct {
	Wars struct {
		Ranked    *warDTO  `json:"ranked"`
		Raids     []warDTO `json:"raids"`
		Territory []warDTO `json:"territory"`
	} `json:"wars"`
}

// attackDTO mirrors an attack from /v2/faction/attacks
type attackDTO struct {
	ID                  int64                   `json:"id"`
	Code                string                  `json:"code"`
	Started             int64                   `json:"started"`
	Ended               int64                   `json:"ended"`
	Attacker            userDTO                 `json:"attacker"`
	Defender            userDTO                 `json:"defender"`
	Result              string                  `json:"result"`
	RespectGain         float64                 `json:"respect_gain"`
	RespectLoss         float64                 `json:"respect_loss"`
	Chain               int                     `json:"chain"`
	IsInterrupted       bool                    `json:"is_interrupted"`
	IsStealthed         bool                    `json:"is_stealthed"`
	IsRaid              bool                    `json:"is_raid"`
	IsRankedWar         bool                    `json:"is_ranked_war"`
	Modifiers           attackModifiersDTO      `json:"modifiers"`
	FinishingHitEffects []finishingHitEffectDTO `json:"finishing_hit_effects"`
}

// attackModifiersDTO mirrors the modifiers applied to an attack
type attackModifiersDTO struct {
	FairFight   float64 `json:"fair_fight"`
	War         float64 `json:"war"`
	Retaliation float64 `json:"retaliation"`
	Group       float64 `json:"group"`
	Overseas    float64 `json:"overseas"`
	Chain       float64 `json:"chain"`
	Warlord     float64 `json:"warlord"`
}

// finishingHitEffectDTO mirrors a finishing hit effect
type finishingHitEffectDTO struct {
	Name  string  `json:"name"`
	Value float64 `json:"value"`
}

// userDTO mirrors a user in an attack
type userDTO struct {
	ID      int         `json:"id"`
	Name    string      `json:"name"`
	Level   int         `json:"level"`
	Faction *factionDTO `json:"faction"`
}

// attacksResponseDTO mirrors the response from /v2/faction/attacks
type attacksResponseDTO struct {
	Attacks []attackDTO `json:"attacks"`
}

// factionInfoDTO mirrors the response from /faction/?selections=basic
// and /faction/{id}?selections=basic
type factionInfoDTO struct {
	ID       int                         `json:"ID"`
	Name     string                      `json:"name"`
	Tag      string                      `json:"tag"`
	TagImage string                      `json:"tag_image"`
	Leader   int                         `json:"leader"`
	CoLeader int                         `json:"co-leader"`
	Respect  int                         `json:"respect"`
	Age      int                         `json:"age"`
	Members  map[string]factionMemberDTO `json:"members"`
}

// factionMemberDTO mirrors a faction member's data
type factionMemberDTO struct {
	Name          string          `json:"name"`
	Level         int             `json:"level"`
	DaysInFaction int             `json:"days_in_faction"`
	LastAction    lastActionDTO   `json:"last_action"`
	Status        memberStatusDTO `json:"status"`
	Position      string          `json:"position"`
}

// lastActionDTO mirrors a member's last action
type lastActionDTO struct {
	Status    string `json:"status"`
	Timestamp int64  `json:"timestamp"`
	Relative  string `json:"relative"`
}

// memberStatusDTO mirrors a member's current status/location
type memberStatusDTO struct {
	Description    string `json:"description"`
	State          string `json:"state"`
	Color          string `json:"color"`
	Details        string `json:"details"`
	Until          *int64 `json:"until"`
	TravelType     string `json:"travel_type"`
	PlaneImageType string `json:"plane_image_type"`
}

func (d warDTO) toDomain() domain.War {
	return domain.War{
		ID:       d.ID,
		Start:    d.Start,
		End:      d.End,
		Target:   d.Target,
		Winner:   d.Winner,
		Factions: factionsToDomain(d.Factions),
	}
}

func warsToDomain(dtos []warDTO) []domain.War {
	if dtos == nil {
		return nil
	}
	wars := make([]domain.War, len(dtos))
	for i, d := range dtos {
		wars[i] = d.toDomain()
	}
	return wars
}

func (d factionDTO) toDomain() domain.Faction {
	return domain.Faction(d)
}

func factionsToDomain(dtos []factionDTO) []domain.Faction {
	if dtos == nil {
		return nil
	}
	factions := make([]domain.Faction, len(dtos))
	for i, d := range dtos {
		factions[i] = d.toDomain()
	}
	return factions
}

func (d warsResponseDTO) toDomain() *domain.FactionWars {
	wars := &domain.FactionWars{
		Raids:     warsToDomain(d.Wars.Raids),
		Territory: warsToDomain(d.Wars.Territory),
	}
	if d.Wars.Ranked != nil {
		ranked := d.Wars.Ranked.toDomain()
		wars.Ranked = &ranked
	}
	return wars
}

func (d attackDTO) toDomain() domain.Attack {
	a := domain.Attack{
		ID:            d.ID,
		Code:          d.Code,
		Started:       d.Started,
		Ended:         d.Ended,
		Attacker:      d.Attacker.toDomain(),
		Defender:      d.Defender.toDomain(),
		Result:        d.Result,
		RespectGain:   d.RespectGain,
		RespectLoss:   d.RespectLoss,
		Chain:         d.Chain,
		IsInterrupted: d.IsInterrupted,
		IsStealthed:   d.IsStealthed,
		IsRaid:        d.IsRaid,
		IsRankedWar:   d.IsRankedWar,
		Modifiers:     domain.AttackModifiers(d.Modifiers),
	}
	if d.FinishingHitEffects != nil {
		a.FinishingHitEffects = make([]domain.FinishingHitEffect, len(d.FinishingHitEffects))
		for i, e := range d.FinishingHitEffects {
			a.FinishingHitEffects[i] = domain.FinishingHitEffect(e)
		}
	}
	return a
}

func (d attacksResponseDTO) toDomain() []domain.Attack {
	attacks := make([]domain.Attack, len(d.Attacks))
	for i, a := range d.Attacks {
		attacks[i] = a.toDomain()
	}
	return attacks
}

func (d userDTO) toDomain() domain.User {
	u := domain.User{
		ID:    d.ID,
		Name:  d.Name,
		Level: d.Level,
	}
	if d.Faction != nil {
		f := d.Faction.toDomain()
		u.Faction = &f
	}
	return u
}

func (d factionInfoDTO) toDomain() *domain.FactionInfo {
	info := &domain.FactionInfo{
		ID:       d.ID,
		Name:     d.Name,
		Tag:      d.Tag,
		TagImage: d.TagImage,
		Leader:   d.Leader,
		CoLeader: d.CoLeader,
		Respect:  d.Respect,
		Age:      d.Age,
		Members:  make(map[string]domain.FactionMember, len(d.Members)),
	}
	for id, m := range d.Members {
		info.Members[id] = domain.FactionMember{
			Name:          m.Name,
			Level:         m.Level,
			DaysInFaction: m.DaysInFaction,
			LastAction:    domain.LastAction(m.LastAction),
			Status:        domain.MemberStatus(m.Status),
			Position:      m.Position,
		}
	}
	return info
}
