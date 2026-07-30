package worldmap_inits

import (
	"math"
	"sort"

	"server.slg.com/services/internal/cores/cores_declarations"
)

// DefaultMapConfig 默认大地图配置
// 1000×1000 格，每格宽 ScreenWeight=40 → 25×25 个 Screen
type DefaultMapConfig struct {
	scope int32
	count int32
}

func NewDefaultMapConfig() *DefaultMapConfig {
	scope := int32(1000)
	count := scope * scope
	return &DefaultMapConfig{
		scope: scope,
		count: count,
	}
}

func (c *DefaultMapConfig) MapCount() int32 {
	return c.count
}

func (c *DefaultMapConfig) MapScope() int32 {
	return c.scope
}

func (c *DefaultMapConfig) MapID2XY(id cores_declarations.MapID) (x, y int32) {
	if id < 0 {
		return -1, -1
	}
	idx := int32(id)
	return idx % c.scope, idx / c.scope
}

func (c *DefaultMapConfig) XY2MapID(x, y int32) cores_declarations.MapID {
	return cores_declarations.MapID(y*c.scope + x)
}

func (c *DefaultMapConfig) SortByDis(mapID cores_declarations.MapID, mapIDs []cores_declarations.MapID) {
	cx, cy := c.MapID2XY(mapID)
	sort.Slice(mapIDs, func(i, j int) bool {
		ix, iy := c.MapID2XY(mapIDs[i])
		jx, jy := c.MapID2XY(mapIDs[j])
		di := (ix-cx)*(ix-cx) + (iy-cy)*(iy-cy)
		dj := (jx-cx)*(jx-cx) + (jy-cy)*(jy-cy)
		return di < dj
	})
}

func (c *DefaultMapConfig) CoverMapIDs(baseMapID int32, landCover int, radius any) []cores_declarations.MapID {
	// radius 参数支持 uint32 或 int
	var r int32
	switch v := radius.(type) {
	case uint32:
		r = int32(v)
	case int:
		r = int32(v)
	default:
		r = 0
	}

	x := baseMapID % c.scope
	y := baseMapID / c.scope

	minX := int32(math.Max(float64(x-r), 0))
	maxX := int32(math.Min(float64(x+r), float64(c.scope-1)))
	minY := int32(math.Max(float64(y-r), 0))
	maxY := int32(math.Min(float64(y+r), float64(c.scope-1)))

	var ids []cores_declarations.MapID
	for ty := minY; ty <= maxY; ty++ {
		for tx := minX; tx <= maxX; tx++ {
			ids = append(ids, cores_declarations.MapID(ty*c.scope+tx))
		}
	}
	return ids
}
