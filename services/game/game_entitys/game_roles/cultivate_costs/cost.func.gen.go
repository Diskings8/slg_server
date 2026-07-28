package cultivate_costs

import "server.slg.com/api/protocol/pb/pb_cultivate"

//------------------------------- getters -------------------------------

func (cc *CultivateCost) GetID() uint64              { return cc.CultivateCost.ID }
func (cc *CultivateCost) GetCreatedAt() int64        { return cc.CultivateCost.CreatedAt }
func (cc *CultivateCost) GetUpdatedAt() int64        { return cc.CultivateCost.UpdatedAt }
func (cc *CultivateCost) GetRoleID() uint64          { return cc.CultivateCost.RoleID }
func (cc *CultivateCost) GetCultivateType() pb_cultivate.CultivateType { return cc.CultivateCost.CultivateType }

//------------------------------- setters -------------------------------

func (cc *CultivateCost) SetID(v uint64)             { cc.CultivateCost.ID = v }
func (cc *CultivateCost) SetCreatedAt(v int64)       { cc.CultivateCost.CreatedAt = v }
func (cc *CultivateCost) SetUpdatedAt(v int64)       { cc.CultivateCost.UpdatedAt = v }
func (cc *CultivateCost) SetRoleID(v uint64)         { cc.CultivateCost.RoleID = v }
func (cc *CultivateCost) SetCultivateType(v pb_cultivate.CultivateType) { cc.CultivateCost.CultivateType = v }
