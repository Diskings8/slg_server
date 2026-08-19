package battle

import (
	"fmt"
)

// Validate 校验战斗规则完整性。
func (c *Conf) Validate() error {
	if c.Rounds == 0 {
		return fmt.Errorf("rounds must be > 0")
	}
	if c.InjuryRateStart > 100 {
		return fmt.Errorf("injury_rate_start must be <= 100, got %d", c.InjuryRateStart)
	}
	if c.InjuryRateDecay > c.InjuryRateStart {
		return fmt.Errorf("injury_rate_decay %d > injury_rate_start %d", c.InjuryRateDecay, c.InjuryRateStart)
	}
	if c.SettlementDeadRate > 100 {
		return fmt.Errorf("settlement_dead_rate must be <= 100, got %d", c.SettlementDeadRate)
	}
	if c.PhysConverge == 0 || c.PhysConverge > 1000 {
		return fmt.Errorf("phys_converge must be in (0, 1000], got %d", c.PhysConverge)
	}
	if c.MagicConverge == 0 || c.MagicConverge > 1000 {
		return fmt.Errorf("magic_converge must be in (0, 1000], got %d", c.MagicConverge)
	}
	if c.BattleExpCoeff == 0 {
		return fmt.Errorf("battle_exp_coeff must be > 0")
	}
	return nil
}
