package logging

import (
	"fmt"
	"math"
	"os"
	"strings"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

const (
	requestIDField = "reqId"
)

type RequestIDKey struct{}

type Params struct {
	PrettyLogging     bool
	Debug             bool
	Level             zapcore.Level
	EnableFileLogging bool
	LogFileName       string
	MaxSize           int
}

func New(params Params) *zap.Logger {
	var cores []zapcore.Core

	cores = append(cores, newZapCore(zapcore.AddSync(os.Stdout), params.PrettyLogging, params.Level))
	if params.EnableFileLogging {
		fileLoggerSync := zapcore.AddSync(&lumberjack.Logger{
			Filename: params.LogFileName,
			MaxSize:  params.MaxSize,
		})
		cores = append(cores, newZapCore(fileLoggerSync, false, params.Level))
	}

	return newZapLogger(zapcore.NewTee(cores...), params.PrettyLogging, params.Debug)
}

func zapBaseEncoderConfig() zapcore.EncoderConfig {
	ec := zap.NewProductionEncoderConfig()
	ec.EncodeDuration = zapcore.SecondsDurationEncoder
	ec.TimeKey = "time"
	return ec
}

func ZapJsonEncoder() zapcore.Encoder {
	ec := zapBaseEncoderConfig()
	ec.EncodeTime = func(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
		nanos := t.UnixNano()
		millis := int64(math.Trunc(float64(nanos) / float64(time.Millisecond)))
		enc.AppendInt64(millis)
	}
	return zapcore.NewJSONEncoder(ec)
}

func zapConsoleEncoder() zapcore.Encoder {
	ec := zapBaseEncoderConfig()
	ec.ConsoleSeparator = " "
	ec.EncodeTime = zapcore.TimeEncoderOfLayout("15:04:05 PM")
	ec.EncodeLevel = zapcore.CapitalColorLevelEncoder
	return zapcore.NewConsoleEncoder(ec)
}

func attachBaseFields(logger *zap.Logger) *zap.Logger {
	host, err := os.Hostname()
	if err != nil {
		host = "unknown"
	}

	logger = logger.With(
		zap.String("hostname", host),
		zap.Int("pid", os.Getpid()),
	)

	return logger
}

func newZapCore(syncer zapcore.WriteSyncer, prettyLogging bool, level zapcore.Level) zapcore.Core {
	var encoder zapcore.Encoder

	if prettyLogging {
		encoder = zapConsoleEncoder()
	} else {
		encoder = ZapJsonEncoder()
	}

	return zapcore.NewCore(encoder, syncer, level)
}

func newZapLogger(core zapcore.Core, prettyLogging bool, debug bool) *zap.Logger {
	var zapOpts []zap.Option

	if debug {
		zapOpts = append(zapOpts, zap.AddCaller())
	}

	zapOpts = append(zapOpts, zap.AddStacktrace(zap.ErrorLevel))

	logger := zap.New(core, zapOpts...)

	if !prettyLogging {
		logger = attachBaseFields(logger)
	}

	return logger
}

func ZapLogLevelFromString(logLevel string) (zapcore.Level, error) {
	switch strings.ToUpper(logLevel) {
	case "DEBUG":
		return zap.DebugLevel, nil
	case "INFO":
		return zap.InfoLevel, nil
	case "WARNING":
		return zap.WarnLevel, nil
	case "ERROR":
		return zap.ErrorLevel, nil
	case "FATAL":
		return zap.FatalLevel, nil
	case "PANIC":
		return zap.PanicLevel, nil
	default:
		return -1, fmt.Errorf("unknown log level: %s", logLevel)
	}
}

func WithRequestID(reqID string) zap.Field {
	return zap.String(requestIDField, reqID)
}
