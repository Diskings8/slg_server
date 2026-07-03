package map_managers

import "server.slg.com/services/internal/cores/cores_declarations"

var defaultOptions = &options{
	stopChan: make(chan struct{}),
	cutNum:   cores_declarations.ServerMapBlockCutNum,
}

type options struct {
	stopChan           chan struct{}
	startTime, endTime int64
	cutNum             int32 // 地图行切块数量
}

type Option func(*options)

func evaluateOptions(opts []Option) *options {
	optCopy := &options{}
	*optCopy = *defaultOptions
	for _, o := range opts {
		o(optCopy)
	}
	return optCopy
}

// WithStopChan stopChan
func WithStopChan(c chan struct{}) Option {
	return func(o *options) {
		o.stopChan = c
	}
}

// WithStartTime WithStartTime
func WithStartTime(startTime int64) Option {
	return func(o *options) {
		o.startTime = startTime
	}
}

// WithEndTime WithEndTime
func WithEndTime(endTime int64) Option {
	return func(o *options) {
		o.endTime = endTime
	}
}

// WithCutNum WithCutNum
func WithCutNum(cutNum int32) Option {
	return func(o *options) {
		o.cutNum = cutNum
	}
}
