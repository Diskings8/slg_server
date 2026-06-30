package pools

// IHandler 池内数据必须实现的接口
type IHandler interface {
	ID() uint64
	Dirty()
	IsDirty() bool
	SaveCache() error
	SaveDB() error
}
