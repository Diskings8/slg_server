package asyncsave_entity

import (
	"sync"
	"sync/atomic"

	"github.com/robfig/cron/v3"
	"server.slg.com/common/common_declarations"
)

type AsyncSaveEntity struct {
	changeChan   chan common_declarations.AsyncSaveEntityI
	changeLocker sync.RWMutex
	changeList   map[common_declarations.AsyncSaveEntityI]struct{} // 这里不能用slice，会因为扩容丢失要保存的数据
	count        atomic.Int32
	entryID      cron.EntryID
}

func (a *AsyncSaveEntity) start() {
	go a.listen()
}

// Save 保存数据
func (a *AsyncSaveEntity) Save(e common_declarations.AsyncSaveEntityI) {
	a.count.Add(1)
	a.changeChan <- e
}

// SaveLen 列表待保存
func (a *AsyncSaveEntity) SaveLen() int {
	a.changeLocker.RLock()
	defer a.changeLocker.RUnlock()
	return len(a.changeList)
}

// SaveSync 立刻保存数据,不保证后续还是否会再次保存
func (a *AsyncSaveEntity) SaveSync(e common_declarations.AsyncSaveEntityI) {
	a.SaveStop(e)
	if !e.IsDelete() {
		e.SaveDo()
	}
}

// SaveChangeData 保存数据
func (a *AsyncSaveEntity) SaveChangeData() {
	a.saveChangeData()
}

// SaveStop 停止保存
func (a *AsyncSaveEntity) SaveStop(e common_declarations.AsyncSaveEntityI) {
	a.changeLocker.Lock()
	defer a.changeLocker.Unlock()
	delete(a.changeList, e)
}

func (a *AsyncSaveEntity) listen() {
	for s := range a.changeChan {
		a.changeLocker.Lock()
		a.changeList[s] = struct{}{}
		a.changeLocker.Unlock()
		a.count.Add(-1)
	}
}

func (a *AsyncSaveEntity) wait() {
	for {
		// 等待
		if a.count.Load() < 1 {
			break
		}
	}
}

func (a *AsyncSaveEntity) saveAll() {
	// 等待队列保存
	a.wait()

	for a.SaveLen() > 0 {
		// 入库保存
		a.saveChangeData()
	}
}

// saveChangeData 保存数据
func (a *AsyncSaveEntity) saveChangeData() {
	a.changeLocker.Lock()
	toSave := make(map[common_declarations.AsyncSaveEntityI]struct{}, len(a.changeList))
	for v := range a.changeList {
		if v.Saving().CompareAndSwap(false, true) {
			delete(a.changeList, v)
			toSave[v] = struct{}{}
		}
	}
	a.changeLocker.Unlock()

	for v := range toSave {
		if !v.IsDelete() {
			v.SaveDo()
		}

		v.Saving().Store(false)
	}
}
