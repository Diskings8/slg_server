package battle

import (
	"fmt"

	"server.slg.com/api/game_conf/table"
	"server.slg.com/common/utils/util_jsons"
)

// battleJSON 战斗规则配置表 JSON 结构（磁盘格式，snake_case）
type battleJSON struct {
	Rounds             uint32 `json:"rounds"`
	InjuryRateStart    uint32 `json:"injury_rate_start"`
	InjuryRateDecay    uint32 `json:"injury_rate_decay"`
	SettlementDeadRate uint32 `json:"settlement_dead_rate"`
	PhysConverge       uint32 `json:"phys_converge"`
	MagicConverge      uint32 `json:"magic_converge"`
	BattleExpCoeff     uint32 `json:"battle_exp_coeff"`
}

// FileName 表名（JSON 文件名，不含扩展名）
func (c *Conf) FileName() string { return "battle" }

// Version 内容版本（JSON 加载后为内容 hash；Go 内嵌为 ""）
func (c *Conf) Version() string { return c.version }

// Load 从 JSON 字节构建战斗规则（覆盖占位）。公式类方法（InjuryRate/Coeff）保留在代码层，不进表。
func (c *Conf) Load(data []byte) error {
	var j battleJSON
	if err := util_jsons.Unmarshal(data, &j); err != nil {
		return fmt.Errorf("unmarshal: %w", err)
	}
	c.Rounds = j.Rounds
	c.InjuryRateStart = j.InjuryRateStart
	c.InjuryRateDecay = j.InjuryRateDecay
	c.SettlementDeadRate = j.SettlementDeadRate
	c.PhysConverge = j.PhysConverge
	c.MagicConverge = j.MagicConverge
	c.BattleExpCoeff = j.BattleExpCoeff
	c.version = table.ContentHash(data)
	return nil
}

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
