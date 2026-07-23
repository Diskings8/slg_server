package marchdos

import (
	"server.slg.com/services/internal/cores/map_datas"
	"server.slg.com/services/internal/cores/map_managers"
	"server.slg.com/services/internal/cores/marchs"
)

// BaseMarch 行军执行器基类，封装行军锁定和解锁逻辑，提供来源和目标地图格子的并发安全访问
type BaseMarch struct {
	marchManage   *marchs.MarchInfoManager // 行军管理
	mgr           *map_managers.MapManager // MarchDoFuncHandleI.Do() 使用的管理器
	fromMapInfo   *map_datas.MapInfo       // 来源地图信息
	toMapInfo     *map_datas.MapInfo       // 目标地图信息
	marchLockOk   bool
	fromMapLockOk bool
	toMapLockOk   bool
	hadInit       bool
	err           error
	prepareOpts   []func(*map_managers.MapManager) // 先进先出
	doOpts        []func(*map_managers.MapManager) // 先进先出
	finishOpts    []func(*map_managers.MapManager) // 先进先出

	// ---- 召回（CallBack）操作链 ----
	prepareCallBackOpts    []func(*map_managers.MapManager) // 召回前置
	callBackOpts           []func(*map_managers.MapManager) // 召回核心
	finishCallBackOpts     []func(*map_managers.MapManager) // 召回后置
	prepareCallBackNowOpts []func(*map_managers.MapManager) // 立即召回前置
	callBackNowOpts        []func(*map_managers.MapManager) // 立即召回核心
	finishCallBackNowOpts  []func(*map_managers.MapManager) // 立即召回后置

	// ---- 召回到达（BackArrive）操作链 ----
	prepareBackArriveOpts []func(*map_managers.MapManager) // 召回到达前置
	backArriveOpts        []func(*map_managers.MapManager) // 召回到达核心
	finishBackArriveOpts  []func(*map_managers.MapManager) // 召回到达后置
}

func (m *BaseMarch) Init() {
	m.AddPrepareOpt(func(manager *map_managers.MapManager) {})
	m.AddDoOpt(func(manager *map_managers.MapManager) {})
	m.AddFinishOpt(func(manager *map_managers.MapManager) {})
	m.hadInit = true
}

// SetManager 设置地图管理器，供 Do() error 使用
func (m *BaseMarch) SetManager(mm *map_managers.MapManager) {
	m.mgr = mm
}

// SetBase 通过 MapManager 统一设置管理器依赖（mgr + marchManage）
func (m *BaseMarch) SetBase(mm *map_managers.MapManager) {
	m.mgr = mm
	m.marchManage = mm.GetMarchManage()
}

// Do 执行行军流程（MarchDoFuncHandleI 接口实现）
//
// 按顺序执行 prepareOpts → doOpts → finishOpts。
// 需要先通过 SetManager 设置地图管理器，否则 panic。
func (m *BaseMarch) Do() error {
	if !m.hadInit {
		panic("marchdos: Do called before Init, missing BaseMarch.Init()?")
	}
	if m.mgr == nil {
		panic("marchdos: Do called before SetManager")
	}
	for _, v := range m.prepareOpts {
		v(m.mgr)
	}
	for _, v := range m.doOpts {
		v(m.mgr)
	}
	for _, v := range m.finishOpts {
		v(m.mgr)
	}
	return nil
}

// DoWithManager 手动指定管理器执行行军流程（兼容内部调用）
func (m *BaseMarch) DoWithManager(mapManager *map_managers.MapManager) {
	if !m.hadInit {
		panic("marchdos: DoWithManager called before Init")
	}
	for _, v := range m.prepareOpts {
		v(mapManager)
	}
	for _, v := range m.doOpts {
		v(mapManager)
	}
	for _, v := range m.finishOpts {
		v(mapManager)
	}
}

func (m *BaseMarch) AddPrepareOpt(f func(*map_managers.MapManager)) {
	m.prepareOpts = append(m.prepareOpts, f)
}

func (m *BaseMarch) AddDoOpt(f func(*map_managers.MapManager)) {
	m.doOpts = append(m.doOpts, f)
}

func (m *BaseMarch) AddFinishOpt(f func(*map_managers.MapManager)) {
	m.finishOpts = append(m.finishOpts, f)
}

// ---- CallBack 操作链注册 ----

func (m *BaseMarch) AddPrepareCallBackOpt(f func(*map_managers.MapManager)) {
	m.prepareCallBackOpts = append(m.prepareCallBackOpts, f)
}

func (m *BaseMarch) AddCallBackOpt(f func(*map_managers.MapManager)) {
	m.callBackOpts = append(m.callBackOpts, f)
}

func (m *BaseMarch) AddFinishCallBackOpt(f func(*map_managers.MapManager)) {
	m.finishCallBackOpts = append(m.finishCallBackOpts, f)
}

// ---- CallBackNow 操作链注册 ----

func (m *BaseMarch) AddPrepareCallBackNowOpt(f func(*map_managers.MapManager)) {
	m.prepareCallBackNowOpts = append(m.prepareCallBackNowOpts, f)
}

func (m *BaseMarch) AddCallBackNowOpt(f func(*map_managers.MapManager)) {
	m.callBackNowOpts = append(m.callBackNowOpts, f)
}

func (m *BaseMarch) AddFinishCallBackNowOpt(f func(*map_managers.MapManager)) {
	m.finishCallBackNowOpts = append(m.finishCallBackNowOpts, f)
}

// ---- BackArrive 操作链注册 ----

func (m *BaseMarch) AddPrepareBackArriveOpt(f func(*map_managers.MapManager)) {
	m.prepareBackArriveOpts = append(m.prepareBackArriveOpts, f)
}

func (m *BaseMarch) AddBackArriveOpt(f func(*map_managers.MapManager)) {
	m.backArriveOpts = append(m.backArriveOpts, f)
}

func (m *BaseMarch) AddFinishBackArriveOpt(f func(*map_managers.MapManager)) {
	m.finishBackArriveOpts = append(m.finishBackArriveOpts, f)
}

// ----------------------------------------------------------------
// CallBack / CallBackNow 模板方法
// ----------------------------------------------------------------

// CallBack 召回行军（模板方法）
//
// 按顺序执行 prepareCallBackOpts → callBackOpts → finishCallBackOpts。
// 子类可通过 AddPrepareCallBackOpt/AddCallBackOpt/AddFinishCallBackOpt 注册自定义逻辑。
func (m *BaseMarch) CallBack() error {
	if !m.hadInit {
		panic("marchdos: CallBack called before Init")
	}
	for _, v := range m.prepareCallBackOpts {
		v(m.mgr)
	}
	for _, v := range m.callBackOpts {
		v(m.mgr)
	}
	for _, v := range m.finishCallBackOpts {
		v(m.mgr)
	}
	return nil
}

// CallBackNow 立即召回行军（模板方法）
//
// 按顺序执行 prepareCallBackNowOpts → callBackNowOpts → finishCallBackNowOpts。
func (m *BaseMarch) CallBackNow() error {
	if !m.hadInit {
		panic("marchdos: CallBackNow called before Init")
	}
	for _, v := range m.prepareCallBackNowOpts {
		v(m.mgr)
	}
	for _, v := range m.callBackNowOpts {
		v(m.mgr)
	}
	for _, v := range m.finishCallBackNowOpts {
		v(m.mgr)
	}
	return nil
}

// BackArrive 召回到达处理（模板方法）
//
// 当行军在 Back 状态下到达（ticker 触发）时调用。
// 按顺序执行 prepareBackArriveOpts → backArriveOpts → finishBackArriveOpts。
// 子类可通过 AddPrepareBackArriveOpt/AddBackArriveOpt/AddFinishBackArriveOpt 注册自定义逻辑。
func (m *BaseMarch) BackArrive() error {
	if !m.hadInit {
		panic("marchdos: BackArrive called before Init")
	}
	for _, v := range m.prepareBackArriveOpts {
		v(m.mgr)
	}
	for _, v := range m.backArriveOpts {
		v(m.mgr)
	}
	for _, v := range m.finishBackArriveOpts {
		v(m.mgr)
	}
	return nil
}

// ReTry 召回重试（模板方法）
//
// 当召回操作失败时进行重试，默认实现为直接委托 CallBack。
// 子类可覆写以加入退避重试、延迟重试等逻辑。
func (m *BaseMarch) ReTry() error {
	if !m.hadInit {
		panic("marchdos: ReTry called before Init")
	}
	return m.CallBack()
}
