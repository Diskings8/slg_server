package asyncsave_entity

import (
	"errors"

	"github.com/go4org/hashtriemap"
	"server.slg.com/common/common_declarations"
	"server.slg.com/common/utils/crontabs"
)

var asyncSaveManager hashtriemap.HashTrieMap[string, *AsyncSaveEntity]

func NewAsyncSaveEntity(spec, tag string) (*AsyncSaveEntity, error) {
	en := &AsyncSaveEntity{
		changeChan: make(chan common_declarations.AsyncSaveEntityI, 1000),
		changeList: make(map[common_declarations.AsyncSaveEntityI]struct{}, 100),
	}
	var err error
	en.entryID, err = crontabs.AddFunc(spec, en.SaveChangeData)
	if err != nil {
		return nil, err
	}

	_, loaded := asyncSaveManager.LoadOrStore(tag, en)
	if loaded {
		return nil, errors.New("已经初始化了")
	}

	en.start()
	return en, nil
}

func RemoveAsyncSave(tag string) {

}

func SaveEntity(entityI common_declarations.AsyncSaveEntityI) {
	tmp, ok := asyncSaveManager.Load(entityI.Tag())
	if !ok {
		panic(entityI.Tag() + "未初始化！")
	}
	tmp.Save(entityI)
}
