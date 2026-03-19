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

// Module represents a component/package of the application
type Module string

const (
	ModuleMain           Module = "main"
	ModuleDocker         Module = "docker"
	ModuleWatchdog       Module = "watchdog"
	ModuleWebserver      Module = "webserver"
	ModuleGit            Module = "git"
	ModuleConfig         Module = "config"
	ModuleInfrastructure Module = "infrastructure"
	ModuleServer         Module = "server"
	ModuleContainer      Module = "container"
	ModuleFiles          Module = "files"
	ModuleStorage        Module = "storage"
	ModuleAPI            Module = "api"
	ModuleSetup          Module = "setup"
	ModuleUser           Module = "user"
	ModuleHost           Module = "host"
	ModuleServerCfg      Module = "servercfg"
	ModuleUnknown        Module = "unknown"
)

// moduleEmojis maps each module to its emoji prefix
var moduleEmojis = map[Module]string{
	ModuleMain:           "⚔️ ",
	ModuleDocker:         "🐳 ",
	ModuleWatchdog:       "🐺 ",
	ModuleWebserver:      "🤵🏻‍♂️ ",
	ModuleGit:            "🗄️ ",
	ModuleConfig:         "⚙️ ",
	ModuleInfrastructure: "🏗️ ",
	ModuleServer:         "🎮 ",
	ModuleContainer:      "📦 ",
	ModuleFiles:          "📁 ",
	ModuleStorage:        "💾 ",
	ModuleAPI:            "🔌 ",
	ModuleSetup:          "🔧 ",
	ModuleUser:           "👤 ",
	ModuleHost:           "🖥️ ",
	ModuleServerCfg:      "📝 ",
	ModuleUnknown:        "❓ ",
}

var (
	globalLogger *zap.Logger
	zapLevel     zap.AtomicLevel
	slogLevel    slog.Level
	initOnce     sync.Once
)

// ModuleLogger wraps zap.Logger with module-specific emoji prefix
type ModuleLogger struct {
	*zap.Logger
	module Module
	emoji  string
}

// ------------------------
// Logger Getters
// ------------------------

// GetLogger returns a logger for the specified module with automatic emoji prefix
func GetLogger(module Module) *ModuleLogger {
	initOnce.Do(func() {
		globalLogger = CreateLogger()
	})

	emoji, exists := moduleEmojis[module]
	if !exists {
		emoji = moduleEmojis[ModuleUnknown]
	}

	return &ModuleLogger{
		Logger: globalLogger,
		module: module,
		emoji:  emoji,
	}
}

// GetSLogger returns a structured logger
func GetSLogger() *slog.Logger {
	return slog.New(slogzap.Option{Level: slogLevel, Logger: globalLogger}.NewZapHandler())
}

// ------------------------
// ModuleLogger Methods with Emoji Prefix
// ------------------------

func (ml *ModuleLogger) Info(msg string, fields ...zap.Field) {
	ml.Logger.Info(fmt.Sprintf("%s %s", ml.emoji, msg), fields...)
}

func (ml *ModuleLogger) Debug(msg string, fields ...zap.Field) {
	ml.Logger.Debug(fmt.Sprintf("%s %s", ml.emoji, msg), fields...)
}

func (ml *ModuleLogger) Warn(msg string, fields ...zap.Field) {
	ml.Logger.Warn(fmt.Sprintf("%s %s", ml.emoji, msg), fields...)
}

func (ml *ModuleLogger) Error(msg string, fields ...zap.Field) {
	ml.Logger.Error(fmt.Sprintf("%s %s", ml.emoji, msg), fields...)
}

func (ml *ModuleLogger) Fatal(msg string, fields ...zap.Field) {
	ml.Logger.Fatal(fmt.Sprintf("%s %s", ml.emoji, msg), fields...)
}

// GetEmoji returns the emoji for this module logger
func (ml *ModuleLogger) GetEmoji() string {
	return ml.emoji
}

// GetModule returns the module for this logger
func (ml *ModuleLogger) GetModule() Module {
	return ml.module
}

// GetZapLogger returns the underlying zap.Logger (useful for passing to functions that expect *zap.Logger)
func (ml *ModuleLogger) GetZapLogger() *zap.Logger {
	return ml.Logger
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
	globalLogger.Info("⚙️ Log level changed", zap.String("new_level", string(level)))
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
		// Use globalLogger directly for generic error handling
		globalLogger.Error(fmt.Sprintf("⛔ %s:%d: %s", filename, line, err.Error()))
		b = true
	}
	return
}
