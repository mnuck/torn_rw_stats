package attack

// IsSuccessfulAttack determines if an attack result represents a successful attack.
// Successful attacks include hospitalization, mugging, or the attacker leaving.
//
// Pure function: No I/O operations, fully testable with direct inputs.
func IsSuccessfulAttack(result string) bool {
	successfulResults := []string{"Hospitalized", "Mugged", "Left"}
	for _, successResult := range successfulResults {
		if result == successResult {
			return true
		}
	}
	return false
}

// IsSuccessfulDefense determines if an attack result represents a successful defense.
// Successful defenses include stalemate, escape, or assisted defense.
//
// Pure function: No I/O operations, fully testable with direct inputs.
func IsSuccessfulDefense(result string) bool {
	successfulDefenseResults := []string{"Stalemate", "Escape", "Assist"}
	for _, defenseResult := range successfulDefenseResults {
		if result == defenseResult {
			return true
		}
	}
	return false
}
