package attack

import (
	"testing"
	"testing/quick"

	"torn_rw_stats/internal/app"
)

func TestAttackProcessingServiceProperties(t *testing.T) {
	service := NewAttackProcessingService()

	t.Run("records count equals attacks count", func(t *testing.T) {
		if err := quick.Check(func(n uint8) bool {
			attacks := makeTestAttacks(int(n) % 101)
			records := service.ProcessAttacksIntoRecords(attacks, makeTestWar(), 12345)
			return len(records) == len(attacks)
		}, nil); err != nil {
			t.Error(err)
		}
	})

	t.Run("attack IDs preserved", func(t *testing.T) {
		if err := quick.Check(func(n uint8) bool {
			attacks := makeTestAttacks((int(n) % 50) + 1)
			records := service.ProcessAttacksIntoRecords(attacks, makeTestWar(), 12345)
			if len(records) != len(attacks) {
				return false
			}
			ids := make(map[int64]bool, len(attacks))
			for _, a := range attacks {
				ids[a.ID] = true
			}
			for _, r := range records {
				if !ids[r.AttackID] {
					return false
				}
			}
			return true
		}, nil); err != nil {
			t.Error(err)
		}
	})

	t.Run("attack direction deterministic", func(t *testing.T) {
		if err := quick.Check(func(attackerFactionID, defenderFactionID, ourFactionID int) bool {
			attack := app.Attack{}
			attack.Attacker.Faction = &app.Faction{ID: attackerFactionID}
			attack.Defender.Faction = &app.Faction{ID: defenderFactionID}
			d1 := service.determineAttackDirection(attack, ourFactionID)
			d2 := service.determineAttackDirection(attack, ourFactionID)
			return d1 == d2
		}, nil); err != nil {
			t.Error(err)
		}
	})

	t.Run("outgoing when attacker is ours", func(t *testing.T) {
		if err := quick.Check(func(ourID uint32) bool {
			ours := int(ourID) + 1
			attack := app.Attack{}
			attack.Attacker.Faction = &app.Faction{ID: ours}
			attack.Defender.Faction = &app.Faction{ID: ours + 1}
			return service.determineAttackDirection(attack, ours) == "Outgoing"
		}, nil); err != nil {
			t.Error(err)
		}
	})

	t.Run("incoming when defender is ours", func(t *testing.T) {
		if err := quick.Check(func(ourID uint32) bool {
			ours := int(ourID) + 1
			attack := app.Attack{}
			attack.Attacker.Faction = &app.Faction{ID: ours + 1}
			attack.Defender.Faction = &app.Faction{ID: ours}
			return service.determineAttackDirection(attack, ours) == "Incoming"
		}, nil); err != nil {
			t.Error(err)
		}
	})

	t.Run("respect values non-negative and preserved", func(t *testing.T) {
		if err := quick.Check(func(n uint8) bool {
			attacks := makeTestAttacksWithRespect(int(n) % 20)
			records := service.ProcessAttacksIntoRecords(attacks, makeTestWar(), 12345)
			for i, r := range records {
				if r.RespectGain < 0 || r.RespectLoss < 0 {
					return false
				}
				if i < len(attacks) {
					if r.RespectGain != attacks[i].RespectGain || r.RespectLoss != attacks[i].RespectLoss {
						return false
					}
				}
			}
			return true
		}, nil); err != nil {
			t.Error(err)
		}
	})
}

func makeTestAttacks(n int) []app.Attack {
	attacks := make([]app.Attack, n)
	for i := range attacks {
		attacks[i] = app.Attack{ID: int64(i + 1)}
	}
	return attacks
}

func makeTestAttacksWithRespect(n int) []app.Attack {
	attacks := make([]app.Attack, n)
	for i := range attacks {
		attacks[i] = app.Attack{
			ID:          int64(i + 1),
			RespectGain: float64(i % 50),
			RespectLoss: float64(i % 25),
		}
	}
	return attacks
}

func makeTestWar() *app.War {
	return &app.War{
		ID:       1,
		Factions: []app.Faction{{ID: 1, Name: "F1"}, {ID: 2, Name: "F2"}},
	}
}
