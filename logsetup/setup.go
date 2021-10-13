package logsetup

import (
	"os"
	"strconv"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// SetupLogging sets up logging.
func SetupLogging() (*zap.Logger, error) {
	debug, _ := strconv.ParseBool(os.Getenv("DEBUG_LOGGING"))

	zcfg := zap.NewProductionConfig()
	zcfg.Encoding = "console"
	zcfg.EncoderConfig.EncodeTime = timeEncoder
	zcfg.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	zcfg.EncoderConfig.EncodeCaller = zapcore.ShortCallerEncoder
	zcfg.EncoderConfig.EncodeDuration = zapcore.StringDurationEncoder

	if debug {
		zcfg.Level.SetLevel(zapcore.DebugLevel)
	} else {
		zcfg.Level.SetLevel(zapcore.InfoLevel)
	}

	log, err := zcfg.Build(zap.AddStacktrace(zapcore.ErrorLevel))
	if err != nil {
		return nil, err
	}

	zap.RedirectStdLog(log)
	return log, err
}

const layout = "15:04:05.000"

func timeEncoder(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
	type appendTimeEncoder interface {
		AppendTimeLayout(time.Time, string)
	}

	if enc, ok := enc.(appendTimeEncoder); ok {
		enc.AppendTimeLayout(t, layout)
		return
	}

	enc.AppendString(t.Format(layout))
}
