package tools

import (
	"os"
	"path/filepath"
	"time"

	"github.com/natefinch/lumberjack"
	"go.uber.org/zap/zapcore"
)

type LoggerConfig struct {
	Mode       string // dev/test/prod
	LogDir     string
	MaxSize    int
	MaxBackups int
	MaxAge     int
	Compress   bool
}

func GetLogWriterWithFilename(cfg *LoggerConfig, filename string) zapcore.WriteSyncer {
	date := time.Now().Format("2006-01-02")
	logPath := filepath.Join(cfg.LogDir, date)
	_ = os.MkdirAll(logPath, os.ModePerm)

	fullPath := filepath.Join(logPath, filename)

	lumberJackLogger := &lumberjack.Logger{
		Filename:   fullPath,
		MaxSize:    cfg.MaxSize,
		MaxBackups: cfg.MaxBackups,
		MaxAge:     cfg.MaxAge,
		Compress:   cfg.Compress,
	}
	return zapcore.AddSync(lumberJackLogger)
}

// 返回编码器
func GetEncoder(mode string) zapcore.Encoder {
	return zapcore.NewConsoleEncoder(InitEncoderConfig(mode))
}

func InitEncoderConfig(mode string) zapcore.EncoderConfig {
	if mode == "dev" {
		return zapcore.EncoderConfig{
			TimeKey:        "🕒",
			LevelKey:       "🔔",
			NameKey:        "模块",
			CallerKey:      "📍",
			MessageKey:     "💬",
			StacktraceKey:  "🧵",
			LineEnding:     zapcore.DefaultLineEnding,
			EncodeLevel:    ColorLevelEncoder,
			EncodeTime:     zapcore.ISO8601TimeEncoder,
			EncodeDuration: zapcore.StringDurationEncoder,
			EncodeCaller:   zapcore.FullCallerEncoder, // ← 改这里
		}
	}

	return zapcore.EncoderConfig{
		TimeKey:        "time",
		LevelKey:       "level",
		NameKey:        "module",
		CallerKey:      "caller",
		MessageKey:     "msg",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.CapitalLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.SecondsDurationEncoder,
		EncodeCaller:   zapcore.FullCallerEncoder, // ← 生产模式也改
	}
}

// 彩色日志等级输出（仅 dev 模式用）
func ColorLevelEncoder(level zapcore.Level, enc zapcore.PrimitiveArrayEncoder) {
	switch level {
	case zapcore.DebugLevel:
		enc.AppendString("\033[36m🐛 DEBUG\033[0m")
	case zapcore.InfoLevel:
		enc.AppendString("\033[32m✅ INFO\033[0m")
	case zapcore.WarnLevel:
		enc.AppendString("\033[33m⚠️ WARN\033[0m")
	case zapcore.ErrorLevel:
		enc.AppendString("\033[31m❌ ERROR\033[0m")
	case zapcore.DPanicLevel, zapcore.PanicLevel, zapcore.FatalLevel:
		enc.AppendString("\033[35m🔥 PANIC\033[0m")
	default:
		enc.AppendString(level.String())
	}
}
