package util_randoms

import (
	"math/rand/v2"
	"sync"
	"time"
)

type randST int64

func (r randST) Uint64() uint64 {
	return uint64(time.Now().Nanosecond())
}

var randSeed randST

var randomPool = sync.Pool{
	New: func() any { return rand.New(randSeed) },
}

// Rand 获取随机值
// op: rand.Rand 需要使用Release
func Rand() *rand.Rand {
	return randomPool.Get().(*rand.Rand)
}

func Release(r *rand.Rand) {
	randomPool.Put(r)
}
