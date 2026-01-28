package logger

import (
	"os"

	"github.com/Yat-Muk/prism-v2/internal/pkg/sanitizer"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

// Config 日誌配置
type Config struct {
	Level      string // debug, info, warn, error
	OutputPath string // 日誌文件路徑
	MaxSize    int    // 單個文件最大大小（MB）
	MaxBackups int    // 保留的舊日誌文件數量
	MaxAge     int    // 保留的天數
	Compress   bool   // 是否壓縮
	Console    bool   // 是否輸出到控制台
}

// DefaultConfig 返回默認配置
func DefaultConfig() Config {
	return Config{
		Level:      "info",
		OutputPath: "/var/log/prism/prism.log",
		MaxSize:    10,
		MaxBackups: 5,
		MaxAge:     30,
		Compress:   true,
		Console:    true,
	}
}

// New 創建新的日誌記錄器
func New(cfg Config) (*zap.Logger, error) {
	// 解析日誌級別
	level, err := zapcore.ParseLevel(cfg.Level)
	if err != nil {
		return nil, err
	}

	// 編碼器配置
	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	encoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder

	// 創建核心
	var cores []zapcore.Core

	// 文件輸出
	if cfg.OutputPath != "" {
		fileWriter := zapcore.AddSync(&lumberjack.Logger{
			Filename:   cfg.OutputPath,
			MaxSize:    cfg.MaxSize,
			MaxBackups: cfg.MaxBackups,
			MaxAge:     cfg.MaxAge,
			Compress:   cfg.Compress,
		})

		fileCore := zapcore.NewCore(
			zapcore.NewJSONEncoder(encoderConfig),
			fileWriter,
			level,
		)
		cores = append(cores, fileCore)
	}

	// 控制台輸出
	if cfg.Console {
		consoleEncoder := encoderConfig
		consoleEncoder.EncodeLevel = zapcore.CapitalColorLevelEncoder

		consoleCore := zapcore.NewCore(
			zapcore.NewConsoleEncoder(consoleEncoder),
			zapcore.AddSync(os.Stdout),
			level,
		)
		cores = append(cores, consoleCore)
	}

	// 組合核心
	core := zapcore.NewTee(cores...)

	// 創建 logger
	logger := zap.New(
		core,
		zap.AddCaller(),
		zap.AddStacktrace(zapcore.ErrorLevel),
	)

	return logger, nil
}

// NewDevelopment 創建開發環境日誌記錄器
func NewDevelopment() (*zap.Logger, error) {
	return zap.NewDevelopment()
}

// NewProduction 創建生產環境日誌記錄器
func NewProduction() (*zap.Logger, error) {
	return zap.NewProduction()
}

// String 創建字符串字段
func String(key, val string) zap.Field {
	return zap.String(key, val)
}

// Int 創建整數字段
func Int(key string, val int) zap.Field {
	return zap.Int(key, val)
}

// Error 創建錯誤字段
func Error(err error) zap.Field {
	return zap.Error(err)
}

// — 脫敏日誌字段 —

// SanitizedString 脫敏字符串字段
func SanitizedString(key, val string) zap.Field {
	return zap.String(key, sanitizer.String(val, 4, 4))
}

// SanitizedPassword 脫敏密碼字段
func SanitizedPassword(key, val string) zap.Field {
	return zap.String(key, sanitizer.Password(val))
}

// SanitizedAPIKey 脫敏 API 密鑰字段
func SanitizedAPIKey(key, val string) zap.Field {
	return zap.String(key, sanitizer.APIKey(val))
}

// SanitizedEmail 脫敏郵箱字段
func SanitizedEmail(key, val string) zap.Field {
	return zap.String(key, sanitizer.Email(val))
}

// LoggerWrapper 增強包裝，兼容原有 zap.Logger
type LoggerWrapper struct {
	*zap.Logger
	SafeLogger SafeLogger
}

// GetSafe 返回安全日誌接口
func (lw *LoggerWrapper) GetSafe() SafeLogger {
	return lw.SafeLogger
}

// NewWithSafe 創建帶安全日誌的增強 Logger
func NewWithSafe(cfg Config) (*LoggerWrapper, error) {
	zapLogger, err := New(cfg)
	if err != nil {
		return nil, err
	}

	return &LoggerWrapper{
		Logger:     zapLogger,
		SafeLogger: NewSafeLogger(zapLogger),
	}, nil
}

// NewDevelopmentWithSafe 開發環境
func NewDevelopmentWithSafe() (*LoggerWrapper, error) {
	zapLogger, _ := zap.NewDevelopment() // 忽略第二個返回值
	return &LoggerWrapper{
		Logger:     zapLogger,
		SafeLogger: NewSafeLogger(zapLogger),
	}, nil
}

// NewProductionWithSafe 生產環境
func NewProductionWithSafe() (*LoggerWrapper, error) {
	zapLogger, _ := zap.NewProduction() // 忽略第二個返回值
	return &LoggerWrapper{
		Logger:     zapLogger,
		SafeLogger: NewSafeLogger(zapLogger),
	}, nil
}
