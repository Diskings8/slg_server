package building

import (
	"fmt"

	"server.slg.com/api/protocol/pb/pb_city"
)

// Validate 校验建筑配置完整性
func (c *Conf) Validate() error {
	if len(c.buildingByType) == 0 {
		return fmt.Errorf("buildings must not be empty")
	}
	for t, b := range c.buildingByType {
		if t <= 0 {
			return fmt.Errorf("building type must be > 0, got %d", t)
		}
		if b.Footprint != 4 && b.Footprint != 9 {
			return fmt.Errorf("building %d footprint must be 4 or 9, got %d", t, b.Footprint)
		}
		if b.MaxLevel == 0 {
			return fmt.Errorf("building %d max_level must be > 0", t)
		}
		if b.BuildTimeUx < 0 {
			return fmt.Errorf("building %d build_time_ux must be >= 0", t)
		}
		if b.UpgradeCostGrowth <= 0 || b.UpgradeTimeGrowth <= 0 {
			return fmt.Errorf("building %d upgrade growth must be > 0", t)
		}
		for _, cost := range b.BuildCost {
			if cost.Count < 0 {
				return fmt.Errorf("building %d build_cost count must be >= 0", t)
			}
		}
		for _, cost := range b.UpgradeCostBase {
			if cost.Count < 0 {
				return fmt.Errorf("building %d upgrade_cost_base count must be >= 0", t)
			}
		}
		// 校场：queue_nums 非空、level 升序、首项 level==1
		if t == pb_city.BuildingType_RoleDrill {
			if len(b.QueueNums) == 0 {
				return fmt.Errorf("drill queue_nums must not be empty")
			}
			if b.QueueNums[0].Level != 1 {
				return fmt.Errorf("drill queue_nums first level must be 1, got %d", b.QueueNums[0].Level)
			}
			for i := 1; i < len(b.QueueNums); i++ {
				if b.QueueNums[i].Level <= b.QueueNums[i-1].Level {
					return fmt.Errorf("drill queue_nums level must be ascending, got %d after %d",
						b.QueueNums[i].Level, b.QueueNums[i-1].Level)
				}
			}
		}
		if b.DefensePerLevel < 0 || b.CapPerLevel < 0 || b.ProducePerHourL < 0 {
			return fmt.Errorf("building %d unlock params must be >= 0", t)
		}
	}
	return nil
}
