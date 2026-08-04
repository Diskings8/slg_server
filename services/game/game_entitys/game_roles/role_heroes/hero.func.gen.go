package role_heroes

//------------------------------- getters -------------------------------

func (hr *RoleHero) GetID() uint64            { return hr.RoleHero.ID }
func (hr *RoleHero) GetCreatedAt() int64      { return hr.RoleHero.CreatedAt }
func (hr *RoleHero) GetUpdatedAt() int64      { return hr.RoleHero.UpdatedAt }
func (hr *RoleHero) GetRoleID() uint64        { return hr.RoleHero.RoleID }
func (hr *RoleHero) GetHeroConfID() int32     { return hr.RoleHero.HeroConfID }
func (hr *RoleHero) GetLevel() uint32         { return hr.RoleHero.Level }
func (hr *RoleHero) GetExp() uint32           { return hr.RoleHero.Exp }
func (hr *RoleHero) GetAttrPoint() uint32     { return hr.RoleHero.AttrPoint }
func (hr *RoleHero) GetCurTroopTypeID() int32 { return hr.RoleHero.CurTroopTypeID }
func (hr *RoleHero) GetIsLocked() bool        { return hr.RoleHero.IsLocked }
func (hr *RoleHero) GetIsAwakened() bool      { return hr.RoleHero.IsAwakened }

//------------------------------- setters -------------------------------

func (hr *RoleHero) SetID(v uint64)           { hr.RoleHero.ID = v }
func (hr *RoleHero) SetCreatedAt(v int64)     { hr.RoleHero.CreatedAt = v }
func (hr *RoleHero) SetUpdatedAt(v int64)     { hr.RoleHero.UpdatedAt = v }
func (hr *RoleHero) SetRoleID(v uint64)       { hr.RoleHero.RoleID = v }
func (hr *RoleHero) SetHeroConfID(v int32)    { hr.RoleHero.HeroConfID = v }
func (hr *RoleHero) SetLevel(v uint32)        { hr.RoleHero.Level = v }
func (hr *RoleHero) SetExp(v uint32)          { hr.RoleHero.Exp = v }
func (hr *RoleHero) SetAttrPoint(v uint32)    { hr.RoleHero.AttrPoint = v }
func (hr *RoleHero) SetCurTroopTypeID(v int32) { hr.RoleHero.CurTroopTypeID = v }
func (hr *RoleHero) SetIsLocked(v bool)       { hr.RoleHero.IsLocked = v }
func (hr *RoleHero) SetIsAwakened(v bool)     { hr.RoleHero.IsAwakened = v }
