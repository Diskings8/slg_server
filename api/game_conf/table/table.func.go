package table

import (
	"fmt"
	"hash/fnv"
)

// ContentHash 计算配置原始字节的内容 hash（FNV-32a，8 位十六进制短串）。
//
// 用作每表内容版本：热更时对比新旧 hash，内容未变则跳过原子替换。
func ContentHash(data []byte) string {
	h := fnv.New32a()
	_, _ = h.Write(data)
	return fmt.Sprintf("%08x", h.Sum32())
}
