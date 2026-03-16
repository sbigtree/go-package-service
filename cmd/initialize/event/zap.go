package event

import (
	"go-tmp/cmd/global"
	"go-tmp/core/tools"
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func InitZapLogger() {
	// 日志配置
	cfg := &tools.LoggerConfig{
		Mode:       "dev",
		LogDir:     "./logs",
		MaxSize:    100,
		MaxBackups: 100000,
		MaxAge:     90,
		Compress:   true,
	}

	encoder := tools.GetEncoder(cfg.Mode)

	// 设置日志等级
	level := zapcore.InfoLevel
	if cfg.Mode == "dev" {
		level = zapcore.DebugLevel
	}

	serverLogWriter := tools.GetLogWriterWithFilename(cfg, "server.log") // 所有日志
	errorLogWriter := tools.GetLogWriterWithFilename(cfg, "error.log")   // 仅 error
	panicLogWriter := tools.GetLogWriterWithFilename(cfg, "panic.log")   // 仅 panic

	consoleSyncer := zapcore.Lock(os.Stdout)

	// Core 1: 所有日志 -> server.log + 控制台
	allCore := zapcore.NewCore(
		encoder,
		zapcore.NewMultiWriteSyncer(serverLogWriter, consoleSyncer),
		level,
	)

	// Core 2: 仅 Error -> error.log
	errorCore := zapcore.NewCore(
		encoder,
		zapcore.AddSync(errorLogWriter),
		zap.LevelEnablerFunc(func(l zapcore.Level) bool {
			return l == zapcore.ErrorLevel
		}),
	)

	// Core 3: Panic 及以上 -> panic.log
	panicCore := zapcore.NewCore(
		encoder,
		zapcore.AddSync(panicLogWriter),
		zap.LevelEnablerFunc(func(l zapcore.Level) bool {
			return l >= zapcore.DPanicLevel
		}),
	)

	core := zapcore.NewTee(allCore, errorCore, panicCore)

	// 原始 Logger（不加 Skip）
	logger := zap.New(core,
		zap.AddCaller(),
		zap.AddStacktrace(zapcore.ErrorLevel),
	)

	// 仍然保留 Skip=1 的全局 Logger（用于封装函数调用）
	global.ZapLog = logger.WithOptions(zap.AddCallerSkip(1))

	// 设置 zap.S() 精确定位
	zap.ReplaceGlobals(logger)

	zap.S().Info("✅ Zap 彩色日志系统初始化完成!")
}
