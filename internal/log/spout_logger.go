package log

import (
	"fmt"
	"log/slog"
	"os"
	"runtime"

	slogzap "github.com/samber/slog-zap"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

type LogType string

const (
	DEBUG LogType = "debug"
	INFO  LogType = "info"
	WARN  LogType = "warn"
	ERROR LogType = "error"
)

var globalLogger *zap.Logger
var zapLevel, slogLevel = getLogLevel(INFO) //Todo read this from Config

func GetLogger() *zap.Logger {
	if globalLogger == nil {
		globalLogger = CreateLogger()
	}
	return globalLogger
}

func GetSLogger() *slog.Logger {
	return slog.New(slogzap.Option{Level: slogLevel, Logger: GetLogger()}.NewZapHandler())
}

func CreateLogger() *zap.Logger {

	stdout := zapcore.AddSync(os.Stdout)
	file := zapcore.AddSync(&lumberjack.Logger{
		Filename:   "logs/spoutmc.log",
		MaxSize:    10, // megabytes
		MaxBackups: 5,
		MaxAge:     7, // days
	})
	level := zap.NewAtomicLevelAt(zapLevel)
	productionCfg := zap.NewProductionEncoderConfig()
	productionCfg.TimeKey = "timestamp"
	productionCfg.EncodeTime = zapcore.ISO8601TimeEncoder
	developmentCfg := zap.NewDevelopmentEncoderConfig()
	developmentCfg.EncodeLevel = zapcore.CapitalColorLevelEncoder
	consoleEncoder := zapcore.NewConsoleEncoder(developmentCfg)
	fileEncoder := zapcore.NewConsoleEncoder(productionCfg)

	core := zapcore.NewTee(
		zapcore.NewCore(consoleEncoder, stdout, level),
		zapcore.NewCore(fileEncoder, file, level),
	)

	spoutLogger := zap.New(core)
	return spoutLogger
}

func HandleError(err error) (b bool) {
	if err != nil {
		// notice that we're using 1, so it will actually log where
		// the error happened, 0 = this function, we don't want that.
		_, filename, line, _ := runtime.Caller(1)
		globalLogger.Error(fmt.Sprintf("⛔ %s:%d: %s %s", filename, line, err.Error()))
		b = true
	}
	return
}

func getLogLevel(level LogType) (zapcore.Level, slog.Level) {
	switch level {
	case "debug":
		return zap.DebugLevel, slog.LevelDebug
	case "warn":
		return zap.WarnLevel, slog.LevelWarn
	case "error":
		return zap.ErrorLevel, slog.LevelError
	default:
		return zap.InfoLevel, slog.LevelInfo
	}
}
