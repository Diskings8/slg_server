package roles

import "server.slg.com/common/pollers"

// GetPollerMgr 获取角色数据轮询管理器
func GetPollerMgr() *pollers.PollerManager[*Data] {
	return pollerManager
}

// GetPoller 获取角色数据轮询器
func GetPoller(id uint64) (*pollers.Poller[*Data], error) {
	return pollerManager.Get(id)
}

// Get 获取角色数据
func Get(id uint64) (data *Data, freeFunc func(), releaseFunc func(), err error) {
	p, err := pollerManager.Get(id)
	if err != nil {
		return nil, nil, nil, err
	}
	data, err = p.Get()
	if err != nil {
		return nil, nil, nil, err
	}
	return data, p.Release, p.Save, nil
}

// GetCopy 获取角色复制数据
func GetCopy(id uint64) (data *Data, err error) {
	p, err := pollerManager.Get(id)
	if err != nil {
		return nil, err
	}
	return p.GetCopy(), nil
}

// Close ..
func Close() error {
	if pollerManager != nil {
		return pollerManager.Close()
	}
	return nil
}
