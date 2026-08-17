package troop

import (
	"fmt"

	"server.slg.com/api/game_conf/table"
	"server.slg.com/api/protocol/pb_confs"
	"server.slg.com/common/common_declarations"
	"server.slg.com/common/utils/util_jsons"
)

// itemUseJSON 域内 ItemUse 的 JSON 镜像（common_declarations.ItemUse 无 json tag，需中间结构转换）
type itemUseJSON struct {
	ItemID   int32 `json:"item_id"`
	ItemType int32 `json:"item_type,omitempty"`
	Count    int64 `json:"count"`
}

func (j itemUseJSON) toItemUse() common_declarations.ItemUse {
	return common_declarations.ItemUse{
		ItemID:   pb_confs.ItemID(j.ItemID),
		ItemType: pb_confs.ItemType(j.ItemType),
		Count:    j.Count,
	}
}

func toCosts(rows []itemUseJSON) []common_declarations.ItemUse {
	if len(rows) == 0 {
		return nil
	}
	out := make([]common_declarations.ItemUse, 0, len(rows))
	for _, r := range rows {
		out = append(out, r.toItemUse())
	}
	return out
}

// troopJSON 兵种配置表 JSON 结构（磁盘格式，snake_case）
type troopJSON struct {
	TransformLevel uint32        `json:"transform_level"`
	DefaultTroopID int32         `json:"default_troop_id"`
	UnlockItemConf int32         `json:"unlock_item_conf"`
	TransformCost  []itemUseJSON `json:"transform_cost"`
}

// FileName 表名（JSON 文件名，不含扩展名）
func (c *Conf) FileName() string { return "troop" }

// Version 内容版本（JSON 加载后为内容 hash；Go 内嵌为 ""）
func (c *Conf) Version() string { return c.version }

// Load 从 JSON 字节构建兵种配置（覆盖占位）。
func (c *Conf) Load(data []byte) error {
	var j troopJSON
	if err := util_jsons.Unmarshal(data, &j); err != nil {
		return fmt.Errorf("unmarshal: %w", err)
	}
	c.TransformLevel = j.TransformLevel
	c.DefaultTroopID = j.DefaultTroopID
	c.UnlockItemConf = j.UnlockItemConf
	c.TransformCost = toCosts(j.TransformCost)
	c.version = table.ContentHash(data)
	return nil
}

// Validate 校验兵种配置完整性。
func (c *Conf) Validate() error {
	if c.TransformLevel == 0 {
		return fmt.Errorf("transform_level must be > 0")
	}
	if c.DefaultTroopID <= 0 {
		return fmt.Errorf("default_troop_id must be > 0, got %d", c.DefaultTroopID)
	}
	if c.UnlockItemConf <= 0 {
		return fmt.Errorf("unlock_item_conf must be > 0, got %d", c.UnlockItemConf)
	}
	if c.DefaultTroopID == c.UnlockItemConf {
		return fmt.Errorf("default_troop_id %d == unlock_item_conf %d", c.DefaultTroopID, c.UnlockItemConf)
	}
	return nil
}
