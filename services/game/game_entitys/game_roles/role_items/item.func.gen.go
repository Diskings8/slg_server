package role_items

import "server.slg.com/api/protocol/pb_confs"

//------------------------------- getters -------------------------------

func (ri *RoleItem) GetID() uint64                { return ri.RoleItem.ID }
func (ri *RoleItem) GetCreatedAt() int64          { return ri.RoleItem.CreatedAt }
func (ri *RoleItem) GetUpdatedAt() int64          { return ri.RoleItem.UpdatedAt }
func (ri *RoleItem) GetRoleID() uint64            { return ri.RoleItem.RoleID }
func (ri *RoleItem) GetConfigID() int32           { return ri.RoleItem.ConfigID }
func (ri *RoleItem) GetItemType() pb_confs.ItemType    { return ri.RoleItem.ItemType }
func (ri *RoleItem) GetItemSubType() pb_confs.ItemSubType { return ri.RoleItem.ItemSubType }
func (ri *RoleItem) GetCount() int64              { return ri.RoleItem.Count }

//------------------------------- setters -------------------------------

func (ri *RoleItem) SetID(v uint64)               { ri.RoleItem.ID = v }
func (ri *RoleItem) SetCreatedAt(v int64)         { ri.RoleItem.CreatedAt = v }
func (ri *RoleItem) SetUpdatedAt(v int64)         { ri.RoleItem.UpdatedAt = v }
func (ri *RoleItem) SetRoleID(v uint64)           { ri.RoleItem.RoleID = v }
func (ri *RoleItem) SetConfigID(v int32)          { ri.RoleItem.ConfigID = v }
func (ri *RoleItem) SetItemType(v pb_confs.ItemType)    { ri.RoleItem.ItemType = v }
func (ri *RoleItem) SetItemSubType(v pb_confs.ItemSubType) { ri.RoleItem.ItemSubType = v }
func (ri *RoleItem) SetCount(v int64)             { ri.RoleItem.Count = v }
