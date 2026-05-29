package loggers

import (
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var Log *zap.Logger

func Init() {
	encoder := zapcore.NewConsoleEncoder(zapcore.EncoderConfig{
		TimeKey:      "time",
		LevelKey:     "level",
		MessageKey:   "msg",
		CallerKey:    "caller", // 调用者信息
		EncodeLevel:  zapcore.CapitalColorLevelEncoder,
		EncodeTime:   zapcore.TimeEncoderOfLayout("2006-01-02 15:04:05"),
		EncodeCaller: zapcore.ShortCallerEncoder, // 短格式：main.go:123
		// EncodeCaller: zapcore.FullCallerEncoder, // 长格式：/path/to/main.go:123
	})

	core := zapcore.NewCore(encoder, zapcore.NewMultiWriteSyncer(zapcore.AddSync(os.Stdout)), zap.DebugLevel)
	Log = zap.New(core, zap.AddCaller(), zap.AddCallerSkip(0))
	zap.ReplaceGlobals(Log)
}
