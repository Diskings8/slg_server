// Package gameconfigjson 内嵌 tabtoy 导出的 gameconfig.json（go:embed 同目录文件）。
//
// 单一数据源：game_conf 默认配置（InitDefault）与各域 New() 兜底共用同一份内嵌数据，
// 避免循环依赖（域包不能 import game_conf，只能依赖本叶包）。运行时文件加载
// （Init/ReLoad 走 config_path 指向的 gameconfig.json）与本内嵌同源，无双份数据漂移。
package gameconfigjson

import (
	_ "embed"
	"sync"

	"server.slg.com/api/protocol/pb/pb_gameconfig"
	"server.slg.com/common/utils/util_jsons"
)

//go:embed gameconfig.json
var JSON []byte

var (
	once    sync.Once
	table   *pb_gameconfig.Table
	loadErr error
)

// Table 内嵌 gameconfig.json 解析的 pb.Table（惰性一次解析，线程安全）。
//
// jsoniter 忽略 @Tool/@Version 等未知键，与运行时文件加载路径行为一致。
func Table() (*pb_gameconfig.Table, error) {
	once.Do(func() {
		t := &pb_gameconfig.Table{}
		if err := util_jsons.Unmarshal(JSON, t); err != nil {
			loadErr = err
			return
		}
		table = t
	})
	return table, loadErr
}

// Build 从内嵌 gameconfig.json 构建任意域配置（惰性解析一次 + 委托 build 构造）。
//
// 供各域 New() 兜底使用：避免每域重复「读内嵌 → 解析 → 构造」样板。
func Build[T any](build func(*pb_gameconfig.Table) (*T, error)) (*T, error) {
	t, err := Table()
	if err != nil {
		return nil, err
	}
	return build(t)
}
