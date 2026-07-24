package game_roles

import (
	"server.slg.com/common/common_declarations"
)

func Init(dbc common_declarations.DbcI) {
	err := dbc.AutoMigrate(nil)
	if err != nil {
		panic("AutoMigrate error" + err.Error())
	}
}
