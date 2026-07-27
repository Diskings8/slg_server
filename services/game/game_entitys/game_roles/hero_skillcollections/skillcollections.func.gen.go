package hero_skillcollections

//------------------------------- getters -------------------------------

func (e *HeroSkillCollection) GetID() uint64             { return e.HeroSkillCollection.ID }
func (e *HeroSkillCollection) GetCreatedAt() int64       { return e.HeroSkillCollection.CreatedAt }
func (e *HeroSkillCollection) GetUpdatedAt() int64       { return e.HeroSkillCollection.UpdatedAt }
func (e *HeroSkillCollection) GetRoleID() uint64         { return e.HeroSkillCollection.RoleID }
func (e *HeroSkillCollection) GetSkillConfID() int32     { return e.HeroSkillCollection.SkillConfID }
func (e *HeroSkillCollection) GetIsUnlocked() bool       { return e.HeroSkillCollection.IsUnlocked }
func (e *HeroSkillCollection) GetCollectionLevel() []int32 { return e.HeroSkillCollection.CollectionLevel }

//------------------------------- setters -------------------------------

func (e *HeroSkillCollection) SetID(v uint64)              { e.HeroSkillCollection.ID = v }
func (e *HeroSkillCollection) SetCreatedAt(v int64)        { e.HeroSkillCollection.CreatedAt = v }
func (e *HeroSkillCollection) SetUpdatedAt(v int64)        { e.HeroSkillCollection.UpdatedAt = v }
func (e *HeroSkillCollection) SetRoleID(v uint64)          { e.HeroSkillCollection.RoleID = v }
func (e *HeroSkillCollection) SetSkillConfID(v int32)      { e.HeroSkillCollection.SkillConfID = v }
func (e *HeroSkillCollection) SetIsUnlocked(v bool)        { e.HeroSkillCollection.IsUnlocked = v }
func (e *HeroSkillCollection) SetCollectionLevel(v []int32) { e.HeroSkillCollection.CollectionLevel = v }
