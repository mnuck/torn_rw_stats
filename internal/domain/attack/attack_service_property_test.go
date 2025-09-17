package attack

import (
	"reflect"
	"testing"

	"torn_rw_stats/internal/app"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
)

// TestAttackProcessingServiceProperties uses property-based testing to verify invariants
func TestAttackProcessingServiceProperties(t *testing.T) {
	service := NewAttackProcessingService()

	properties := gopter.NewProperties(nil)

	// Property: Number of records should equal number of input attacks
	properties.Property("records count equals attacks count", prop.ForAll(
		func(attacks []app.Attack, war *app.War) bool {
			records := service.ProcessAttacksIntoRecords(attacks, war, 12345)
			return len(records) == len(attacks)
		},
		genAttacks().SuchThat(func(attacks []app.Attack) bool {
			return len(attacks) <= 100 // Reasonable upper bound
		}),
		genWar(),
	))

	// Property: All attack IDs should be preserved in records
	properties.Property("attack IDs preserved", prop.ForAll(
		func(attacks []app.Attack, war *app.War) bool {
			records := service.ProcessAttacksIntoRecords(attacks, war, 12345)

			if len(records) != len(attacks) {
				return false
			}

			attackIDs := make(map[int64]bool)
			for _, attack := range attacks {
				attackIDs[attack.ID] = true
			}

			for _, record := range records {
				if !attackIDs[record.AttackID] {
					return false
				}
			}
			return true
		},
		genAttacks().SuchThat(func(attacks []app.Attack) bool {
			return len(attacks) <= 50 && len(attacks) > 0
		}),
		genWar(),
	))

	// Property: Attack direction should be deterministic for same faction configuration
	properties.Property("attack direction deterministic", prop.ForAll(
		func(attack app.Attack, ourFactionID int) bool {
			direction1 := service.determineAttackDirection(attack, ourFactionID)
			direction2 := service.determineAttackDirection(attack, ourFactionID)
			return direction1 == direction2
		},
		genAttack(),
		gen.IntRange(1, 999999),
	))

	// Property: Attack direction should be "Outgoing" when attacker faction matches ours
	properties.Property("outgoing when attacker is ours", prop.ForAll(
		func(attack app.Attack, ourFactionID int) bool {
			// Ensure attacker faction matches our faction
			attack.Attacker.Faction = &app.Faction{ID: ourFactionID, Name: "Our Faction"}
			attack.Defender.Faction = &app.Faction{ID: ourFactionID + 1, Name: "Enemy Faction"}

			direction := service.determineAttackDirection(attack, ourFactionID)
			return direction == "Outgoing"
		},
		genAttackWithFactions(),
		gen.IntRange(1, 999999),
	))

	// Property: Attack direction should be "Incoming" when defender faction matches ours
	properties.Property("incoming when defender is ours", prop.ForAll(
		func(attack app.Attack, ourFactionID int) bool {
			// Ensure defender faction matches our faction, attacker doesn't
			attack.Defender.Faction = &app.Faction{ID: ourFactionID, Name: "Our Faction"}
			attack.Attacker.Faction = &app.Faction{ID: ourFactionID + 1, Name: "Enemy Faction"}

			direction := service.determineAttackDirection(attack, ourFactionID)
			return direction == "Incoming"
		},
		genAttackWithFactions(),
		gen.IntRange(1, 999999),
	))

	// Property: Respect values should be non-negative and preserved
	properties.Property("respect values non-negative and preserved", prop.ForAll(
		func(attacks []app.Attack, war *app.War) bool {
			records := service.ProcessAttacksIntoRecords(attacks, war, 12345)

			for i, record := range records {
				if record.RespectGain < 0 || record.RespectLoss < 0 {
					return false
				}
				if i < len(attacks) {
					if record.RespectGain != attacks[i].RespectGain ||
						record.RespectLoss != attacks[i].RespectLoss {
						return false
					}
				}
			}
			return true
		},
		genAttacksWithRespect(),
		genWar(),
	))

	properties.TestingRun(t)
}

// genAttack generates a single attack
func genAttack() gopter.Gen {
	return gen.Struct(reflect.TypeOf(app.Attack{}), map[string]gopter.Gen{
		"ID":          gen.Int64Range(1, 999999),
		"Code":        gen.RegexMatch("[a-z0-9]{8}"),
		"Started":     gen.Int64Range(1640995200, 1740995200), // 2022-2025 range
		"Ended":       gen.Int64Range(1640995200, 1740995200),
		"Result":      gen.OneConstOf("Hospitalized", "Mugged", "Left", "Escape", "Stalemate"),
		"RespectGain": gen.Float64Range(0, 100),
		"RespectLoss": gen.Float64Range(0, 100),
		"Chain":       gen.IntRange(0, 250),
	})
}

// genAttackWithFactions generates an attack with faction information
func genAttackWithFactions() gopter.Gen {
	return gen.Struct(reflect.TypeOf(app.Attack{}), map[string]gopter.Gen{
		"ID":      gen.Int64Range(1, 999999),
		"Code":    gen.RegexMatch("[a-z0-9]{8}"),
		"Started": gen.Int64Range(1640995200, 1740995200),
		"Ended":   gen.Int64Range(1640995200, 1740995200),
		"Attacker": gen.Struct(reflect.TypeOf(app.User{}), map[string]gopter.Gen{
			"ID":      gen.IntRange(1, 999999),
			"Name":    gen.AlphaString(),
			"Level":   gen.IntRange(1, 100),
			"Faction": gen.PtrOf(genFaction()),
		}),
		"Defender": gen.Struct(reflect.TypeOf(app.User{}), map[string]gopter.Gen{
			"ID":      gen.IntRange(1, 999999),
			"Name":    gen.AlphaString(),
			"Level":   gen.IntRange(1, 100),
			"Faction": gen.PtrOf(genFaction()),
		}),
		"Result":      gen.OneConstOf("Hospitalized", "Mugged", "Left", "Escape", "Stalemate"),
		"RespectGain": gen.Float64Range(0, 100),
		"RespectLoss": gen.Float64Range(0, 100),
	})
}

// genAttacks generates a slice of attacks
func genAttacks() gopter.Gen {
	return gen.SliceOf(genAttack())
}

// genAttacksWithRespect generates attacks with valid respect values
func genAttacksWithRespect() gopter.Gen {
	return gen.SliceOf(gen.Struct(reflect.TypeOf(app.Attack{}), map[string]gopter.Gen{
		"ID":          gen.Int64Range(1, 999999),
		"Code":        gen.RegexMatch("[a-z0-9]{8}"),
		"RespectGain": gen.Float64Range(0, 50), // Reasonable respect range
		"RespectLoss": gen.Float64Range(0, 50),
	}))
}

// genFaction generates a faction
func genFaction() gopter.Gen {
	return gen.Struct(reflect.TypeOf(app.Faction{}), map[string]gopter.Gen{
		"ID":   gen.IntRange(1, 999999),
		"Name": gen.AlphaString(),
	})
}

// genWar generates a war with factions
func genWar() gopter.Gen {
	return gen.Struct(reflect.TypeOf(app.War{}), map[string]gopter.Gen{
		"ID":       gen.IntRange(1, 999999),
		"Start":    gen.Int64Range(1640995200, 1740995200),
		"Factions": gen.SliceOfN(2, genFaction()), // Always 2 factions for wars
	}).Map(func(w app.War) *app.War {
		return &w
	})
}
