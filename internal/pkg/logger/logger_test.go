package logger

import (
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"testing"
)

// TestLogger_Init 測試日誌初始化
func TestLogger_Init(t *testing.T) {
	t.Run("初始化成功", func(t *testing.T) {
		// 使用zap直接創建logger
		logger, err := zap.NewProduction()
		assert.NoError(t, err)
		assert.NotNil(t, logger)
		defer logger.Sync()
	})
}

// TestLogger_Levels 測試不同日誌級別
func TestLogger_Levels(t *testing.T) {
	logger, err := zap.NewDevelopment()
	assert.NoError(t, err)
	defer logger.Sync()

	// 這些不應該panic
	assert.NotPanics(t, func() {
		logger.Debug("debug message")
		logger.Info("info message")
		logger.Warn("warn message")
		logger.Error("error message")
	})
}

// TestLogger_WithFields 測試帶字段的日誌
func TestLogger_WithFields(t *testing.T) {
	logger, err := zap.NewProduction()
	assert.NoError(t, err)
	defer logger.Sync()

	assert.NotPanics(t, func() {
		logger.Info("test message",
			zap.String("key1", "value1"),
			zap.Int("key2", 123),
			zap.Bool("key3", true),
		)
	})
}

// TestLogger_Sugar 測試Sugar logger
func TestLogger_Sugar(t *testing.T) {
	logger, err := zap.NewProduction()
	assert.NoError(t, err)
	defer logger.Sync()

	sugar := logger.Sugar()
	assert.NotNil(t, sugar)

	assert.NotPanics(t, func() {
		sugar.Infow("test message",
			"key1", "value1",
			"key2", 123,
		)
	})
}

// TestLogger_Named 測試命名logger
func TestLogger_Named(t *testing.T) {
	logger, err := zap.NewProduction()
	assert.NoError(t, err)
	defer logger.Sync()

	namedLogger := logger.Named("test-module")
	assert.NotNil(t, namedLogger)

	assert.NotPanics(t, func() {
		namedLogger.Info("named logger message")
	})
}
