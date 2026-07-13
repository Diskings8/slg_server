package roles

import (
	"context"
	"errors"
	"time"

	"github.com/patrickmn/go-cache"
	"gorm.io/gorm"
	"server.slg.com/common/pollers"
	"server.slg.com/common/utils/crontabs"
)

var pollerManager *pollers.PollerManager[*Data]
var jsonCache = cache.New(10*time.Minute, 5*time.Minute)

func Init(ctx context.Context) {
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
