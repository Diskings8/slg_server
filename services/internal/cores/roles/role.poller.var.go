package roles

import (
	"context"
	"errors"
	"time"

	"github.com/patrickmn/go-cache"
	"gorm.io/gorm"
	"server.slg.com/common/conns/dbconn"
	"server.slg.com/common/loggers"
	"server.slg.com/common/pollers"
	"server.slg.com/common/utils/crontabs"
)

var pollerManager *pollers.PollerManager[*Data]
var jsonCache = cache.New(10*time.Minute, 5*time.Minute)

// Init 初始化角色数据 poller
//
// 须在数据库初始化后调用：内部会迁移 role_data 表。
func Init(ctx context.Context) {
	if dbc := dbconn.GetWriteDbConn(); dbc != nil {
		if err := dbc.Table("role_data").AutoMigrate(&Data{}); err != nil {
			loggers.Logger.Error("role_data auto_migrate failed: " + err.Error())
		}
	}

	pollerManager = pollers.New(ctx, loader, func() *Data { return &Data{} }, crontabs.Pre30Seconds, crontabs.Pre1Minutes, crontabs.AHalfDay)
}

func loader(id uint64) (*Data, error) {
	r := NewRoleDataInfo(id)
	if err := r.DBGet(); err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
		r.RoleID = id
	}
	return r, nil
}
