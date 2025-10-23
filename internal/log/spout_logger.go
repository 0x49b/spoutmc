package log

import (
	"fmt"
	"log/slog"
	"os"
	"runtime"
	"sync"

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

var (
	globalLogger *zap.Logger
	zapLevel     zap.AtomicLevel
	slogLevel    slog.Level
	initOnce     sync.Once
)

// ------------------------
// Logger Getters
// ------------------------

func GetLogger() *zap.Logger {

	initOnce.Do(func() {
		globalLogger = CreateLogger()
	})
	return globalLogger
}

func GetSLogger() *slog.Logger {
	return slog.New(slogzap.Option{Level: slogLevel, Logger: GetLogger()}.NewZapHandler())
}

// ------------------------
// Logger Creation
// ------------------------

func CreateLogger() *zap.Logger {
	stdout := zapcore.AddSync(os.Stdout)
	file := zapcore.AddSync(&lumberjack.Logger{
		Filename:   "logs/spoutmc.log",
		MaxSize:    10, // MB
		MaxBackups: 5,
		MaxAge:     7, // days
	})

	zapLevel = zap.NewAtomicLevelAt(zapcore.InfoLevel) // default level

	productionCfg := zap.NewProductionEncoderConfig()
	productionCfg.TimeKey = "timestamp"
	productionCfg.EncodeTime = zapcore.ISO8601TimeEncoder

	developmentCfg := zap.NewDevelopmentEncoderConfig()
	developmentCfg.EncodeLevel = zapcore.CapitalColorLevelEncoder

	consoleEncoder := zapcore.NewConsoleEncoder(developmentCfg)
	fileEncoder := zapcore.NewConsoleEncoder(productionCfg)

	core := zapcore.NewTee(
		zapcore.NewCore(consoleEncoder, stdout, zapLevel),
		zapcore.NewCore(fileEncoder, file, zapLevel),
	)

	logger := zap.New(core)
	return logger
}

// ------------------------
// Log Level Handling
// ------------------------

func SetLogLevel(level LogType) {
	zLevel, sLevel := getLogLevel(level)
	slogLevel = sLevel
	if zapLevel.Enabled(zLevel) || !zapLevel.Enabled(zLevel) {
		zapLevel.SetLevel(zLevel)
	}
	GetLogger().Info("Log level changed", zap.String("new_level", string(level)))
}

func getLogLevel(level LogType) (zapcore.Level, slog.Level) {
	switch level {
	case DEBUG:
		return zap.DebugLevel, slog.LevelDebug
	case WARN:
		return zap.WarnLevel, slog.LevelWarn
	case ERROR:
		return zap.ErrorLevel, slog.LevelError
	case INFO:
		return zap.InfoLevel, slog.LevelInfo
	default:
		return zap.InfoLevel, slog.LevelInfo
	}
}

// ------------------------
// Error Helper
// ------------------------

func HandleError(err error) (b bool) {
	if err != nil {
		_, filename, line, _ := runtime.Caller(1)
		GetLogger().Error(fmt.Sprintf("⛔ %s:%d: %s", filename, line, err.Error()))
		b = true
	}
	return
}
