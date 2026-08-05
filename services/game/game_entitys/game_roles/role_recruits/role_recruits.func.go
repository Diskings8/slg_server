package role_recruits

import (
	"go.uber.org/zap"
	"server.slg.com/common/loggers"
	"server.slg.com/common/utils/util_jsons"
	"server.slg.com/services/game/game_models"
)

// Init 单行整包，无需二级索引
func (rrs *RoleRecruits) Init() {}

// Copy 深拷贝
func (rrs *RoleRecruits) Copy(src *RoleRecruits) {
	b, err := util_jsons.Marshal(src)
	if err != nil {
		loggers.Logger.Error("marshal failed", zap.Any("src", src), zap.Error(err))
	}

	err = util_jsons.Unmarshal(b, rrs)
	if err != nil {
		loggers.Logger.Error("unmarshal failed", zap.Any("src", src), zap.Error(err))
	}

	rrs.Init()
}

// GetPool 读取指定抽卡池状态（只读；不存在返回 nil）
func (rrs *RoleRecruits) GetPool(poolID uint32) *game_models.RecruitPool {
	if rrs.Data.Pools == nil {
		return nil
	}
	return rrs.Data.Pools[poolID]
}

// EnsureGetPool 获取指定抽卡池状态，不存在则创建默认零值（写路径）
func (rrs *RoleRecruits) EnsureGetPool(poolID uint32) *game_models.RecruitPool {
	if rrs.Data.Pools == nil {
		rrs.Data.Pools = make(map[uint32]*game_models.RecruitPool)
	}
	pool, ok := rrs.Data.Pools[poolID]
	if !ok {
		pool = &game_models.RecruitPool{ID: poolID}
		rrs.Data.Pools[poolID] = pool
	}
	return pool
}

// SetPool 写回池状态
func (rrs *RoleRecruits) SetPool(pool *game_models.RecruitPool) {
	if rrs.Data.Pools == nil {
		rrs.Data.Pools = make(map[uint32]*game_models.RecruitPool)
	}
	rrs.Data.Pools[pool.ID] = pool
}
