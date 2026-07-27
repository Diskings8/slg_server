package game_roles

import "server.slg.com/common/common_declarations"

// Init 初始化 game_roles 模块
func Init(dbc common_declarations.DbcI) {
	// 自动迁移角色表
	if err := dbc.AutoMigrate(&RoleDbModel{}); err != nil {
		panic("AutoMigrate role failed: " + err.Error())
	}

	// 初始化轮询管理器
	initPoller()
}

// Close 关闭轮询管理器
func Close() error {
	if pollerManager != nil {
		return pollerManager.Close()
	}
	return nil
}
