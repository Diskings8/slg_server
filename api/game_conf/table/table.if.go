// Package table 配置表通用接口与帮助函数。
//
// 各配置域（hero/skill/item/...）的 Conf 实现本接口，由 game_conf 的注册中心
// 统一加载 JSON → 校验 → 建索引，并支持热更与版本追踪。
package table

// Table 配置表通用接口（各域 Conf 实现）
type Table interface {
	// FileName 表对应的 JSON 文件名（不含扩展名），如 "hero"。
	FileName() string
	// Load 用 JSON 字节构建本表（覆盖占位）。失败必须返回 err 且保持本表数据不变（局部构建后提交）。
	Load(data []byte) error
	// Validate 校验本表数据完整性（主键唯一/数值范围/枚举合法性/表内引用）。失败返回 err。
	Validate() error
	// Version 本表内容版本（JSON 加载后为内容 hash 短串；Go 内嵌为 ""）。
	Version() string
}
