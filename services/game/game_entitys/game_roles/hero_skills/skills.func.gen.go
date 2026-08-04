package hero_skills

//------------------------------- getters -------------------------------

func (hs *HeroSkill) GetID() uint64           { return hs.HeroSkill.ID }
func (hs *HeroSkill) GetCreatedAt() int64     { return hs.HeroSkill.CreatedAt }
func (hs *HeroSkill) GetUpdatedAt() int64     { return hs.HeroSkill.UpdatedAt }
func (hs *HeroSkill) GetRoleID() uint64       { return hs.HeroSkill.RoleID }
func (hs *HeroSkill) GetSkillConfID() int32   { return hs.HeroSkill.SkillConfID }
func (hs *HeroSkill) GetLevel() int32         { return hs.HeroSkill.Level }
func (hs *HeroSkill) GetIsAwakened() bool     { return hs.HeroSkill.IsAwakened }
func (hs *HeroSkill) GetIsUnlocked() bool     { return hs.HeroSkill.IsUnlocked }
func (hs *HeroSkill) GetResearchLevel() int32 { return hs.HeroSkill.ResearchLevel }
func (hs *HeroSkill) GetEquipHeroID() uint64  { return hs.HeroSkill.EquipHeroID }
func (hs *HeroSkill) GetUseCountLimit() int32 { return hs.HeroSkill.UseCountLimit }
func (hs *HeroSkill) GetUsedCount() int32     { return hs.HeroSkill.UsedCount }

//------------------------------- setters -------------------------------

func (hs *HeroSkill) SetID(v uint64)           { hs.HeroSkill.ID = v }
func (hs *HeroSkill) SetCreatedAt(v int64)     { hs.HeroSkill.CreatedAt = v }
func (hs *HeroSkill) SetUpdatedAt(v int64)     { hs.HeroSkill.UpdatedAt = v }
func (hs *HeroSkill) SetRoleID(v uint64)       { hs.HeroSkill.RoleID = v }
func (hs *HeroSkill) SetSkillConfID(v int32)   { hs.HeroSkill.SkillConfID = v }
func (hs *HeroSkill) SetLevel(v int32)         { hs.HeroSkill.Level = v }
func (hs *HeroSkill) SetIsAwakened(v bool)     { hs.HeroSkill.IsAwakened = v }
func (hs *HeroSkill) SetIsUnlocked(v bool)     { hs.HeroSkill.IsUnlocked = v }
func (hs *HeroSkill) SetResearchLevel(v int32) { hs.HeroSkill.ResearchLevel = v }
func (hs *HeroSkill) SetEquipHeroID(v uint64)  { hs.HeroSkill.EquipHeroID = v }
func (hs *HeroSkill) SetUseCountLimit(v int32) { hs.HeroSkill.UseCountLimit = v }
func (hs *HeroSkill) SetUsedCount(v int32)     { hs.HeroSkill.UsedCount = v }
