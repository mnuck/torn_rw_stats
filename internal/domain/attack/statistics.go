package attack

import "torn_rw_stats/internal/app"

// AttackStatistics holds calculated attack statistics including total attacks,
// win/loss counts, and respect gained/lost for a faction.
type AttackStatistics struct {
	TotalAttacks  int
	AttacksWon    int
	AttacksLost   int
	RespectGained float64
	RespectLost   float64
}

// CalculateAttackStatistics computes comprehensive attack statistics for a faction.
// It processes all attacks, determining which are offensive vs defensive from the
// faction's perspective, and accumulates wins, losses, and respect changes.
//
// Pure function: No I/O operations, fully testable with direct inputs.
func CalculateAttackStatistics(attacks []app.Attack, ourFactionID int) AttackStatistics {
	var stats AttackStatistics

	for _, attack := range attacks {
		if IsOurAttack(attack, ourFactionID) {
			stats = processOffensiveAttack(stats, attack)
		} else if IsAttackAgainstUs(attack, ourFactionID) {
			stats = processDefensiveAttack(stats, attack)
		}
	}

	return stats
}

// IsOurAttack determines if an attack was performed by our faction
func IsOurAttack(attack app.Attack, ourFactionID int) bool {
	return attack.Attacker.Faction != nil && attack.Attacker.Faction.ID == ourFactionID
}

// IsAttackAgainstUs determines if an attack was performed against our faction
func IsAttackAgainstUs(attack app.Attack, ourFactionID int) bool {
	return attack.Defender.Faction != nil && attack.Defender.Faction.ID == ourFactionID
}

// processOffensiveAttack processes statistics for an attack we performed
func processOffensiveAttack(stats AttackStatistics, attack app.Attack) AttackStatistics {
	stats.TotalAttacks++
	stats.RespectGained += attack.RespectGain
	stats.RespectLost += attack.RespectLoss

	if IsSuccessfulAttack(attack.Result) {
		stats.AttacksWon++
	} else {
		stats.AttacksLost++
	}

	return stats
}

// processDefensiveAttack processes statistics for an attack against us
func processDefensiveAttack(stats AttackStatistics, attack app.Attack) AttackStatistics {
	stats.TotalAttacks++

	// For defensive stats, respect gain/loss is inverted from attacker's perspective
	stats.RespectLost += attack.RespectGain
	stats.RespectGained += attack.RespectLoss

	// We "won" if we defended successfully
	if IsSuccessfulDefense(attack.Result) {
		stats.AttacksWon++
	} else {
		stats.AttacksLost++
	}

	return stats
}
