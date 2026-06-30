package pools

type entry[T IHandler] struct {
	handler  T
	refCount int32
}
