package logger

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestDefaultConfig 測試默認配置
func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	assert.Equal(t, "info", cfg.Level)
	assert.Equal(t, "/var/log/prism/prism.log", cfg.OutputPath)
	assert.Equal(t, 10, cfg.MaxSize)
	assert.Equal(t, 5, cfg.MaxBackups)
	assert.Equal(t, 30, cfg.MaxAge)
	assert.True(t, cfg.Compress)
	assert.True(t, cfg.Console)
}

// TestNew 測試自定義配置創建logger
func TestNew(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := tmpDir + "/test.log"

	cfg := Config{
		Level:      "debug",
		OutputPath: logPath,
		MaxSize:    5,
		MaxBackups: 3,
		MaxAge:     7,
		Compress:   false,
		Console:    false,
	}

	logger, err := New(cfg)
	assert.NoError(t, err)
	assert.NotNil(t, logger)

	// 測試日誌記錄
	logger.Info("test message")

	// 驗證日誌文件創建
	_, err = os.Stat(logPath)
	assert.NoError(t, err, "日誌文件應該被創建")
}

// TestNew_InvalidLevel 測試無效日誌級別
func TestNew_InvalidLevel(t *testing.T) {
	cfg := Config{
		Level:      "invalid",
		OutputPath: "/tmp/test.log",
		Console:    true,
	}

	logger, err := New(cfg)
	assert.Error(t, err)
	assert.Nil(t, logger)
}

// TestNew_ConsoleOnly 測試僅控制台輸出
func TestNew_ConsoleOnly(t *testing.T) {
	cfg := Config{
		Level:      "info",
		OutputPath: "", // 空路徑，只輸出到控制台
		Console:    true,
	}

	logger, err := New(cfg)
	assert.NoError(t, err)
	assert.NotNil(t, logger)

	logger.Info("console only message")
}

// TestNew_FileAndConsole 測試文件和控制台同時輸出
func TestNew_FileAndConsole(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := tmpDir + "/test.log"

	cfg := Config{
		Level:      "debug",
		OutputPath: logPath,
		Console:    true,
	}

	logger, err := New(cfg)
	assert.NoError(t, err)
	assert.NotNil(t, logger)

	logger.Info("file and console message")
}

// TestNewDevelopment 測試開發環境logger
func TestNewDevelopment(t *testing.T) {
	logger, err := NewDevelopment()
	assert.NoError(t, err)
	assert.NotNil(t, logger)

	// 測試日誌記錄
	logger.Info("development logger test")
}

// TestNewProduction 測試生產環境logger
func TestNewProduction(t *testing.T) {
	logger, err := NewProduction()
	assert.NoError(t, err)
	assert.NotNil(t, logger)

	// 測試日誌記錄
	logger.Info("production logger test")
}

// TestLoggerHelpers 測試日誌輔助函數
func TestLoggerHelpers(t *testing.T) {
	t.Run("String", func(t *testing.T) {
		field := String("key", "value")
		assert.NotNil(t, field)
		assert.Equal(t, "key", field.Key)
		assert.Equal(t, "value", field.String)
	})

	t.Run("Int", func(t *testing.T) {
		field := Int("count", 42)
		assert.NotNil(t, field)
		assert.Equal(t, "count", field.Key)
		assert.Equal(t, int64(42), field.Integer)
	})

	t.Run("Error", func(t *testing.T) {
		err := assert.AnError
		field := Error(err)
		assert.NotNil(t, field)
		assert.Equal(t, "error", field.Key)
	})
}

// TestSanitizedHelpers 測試脫敏輔助函數
func TestSanitizedHelpers(t *testing.T) {
	t.Run("SanitizedString", func(t *testing.T) {
		field := SanitizedString("api_key", "sk_test_1234567890abcdef")
		assert.NotNil(t, field)
		assert.Equal(t, "api_key", field.Key)
		// 應該包含脫敏後的值
		assert.NotContains(t, field.String, "1234567890abcdef")
	})

	t.Run("SanitizedPassword", func(t *testing.T) {
		field := SanitizedPassword("password", "mySecretPass123")
		assert.NotNil(t, field)
		assert.Equal(t, "password", field.Key)
		assert.NotContains(t, field.String, "mySecretPass123")
	})

	t.Run("SanitizedAPIKey", func(t *testing.T) {
		field := SanitizedAPIKey("api_key", "sk_live_abcdefghijklmnop")
		assert.NotNil(t, field)
		assert.Equal(t, "api_key", field.Key)
		assert.NotContains(t, field.String, "abcdefghijklmnop")
	})

	t.Run("SanitizedEmail", func(t *testing.T) {
		field := SanitizedEmail("email", "user@example.com")
		assert.NotNil(t, field)
		assert.Equal(t, "email", field.Key)
		// Email應該被部分脫敏
		assert.Contains(t, field.String, "@")
	})
}

// TestLoggerWrapper 測試LoggerWrapper
func TestLoggerWrapper(t *testing.T) {
	cfg := Config{
		Level:   "info",
		Console: true,
	}

	wrapper, err := NewWithSafe(cfg)
	assert.NoError(t, err)
	assert.NotNil(t, wrapper)

	// 測試獲取SafeLogger
	safeLogger := wrapper.GetSafe()
	assert.NotNil(t, safeLogger)

	// 測試SafeLogger功能
	safeLogger.Info("test message")
}

// TestNewDevelopmentWithSafe 測試帶SafeLogger的開發環境logger
func TestNewDevelopmentWithSafe(t *testing.T) {
	wrapper, err := NewDevelopmentWithSafe()
	assert.NoError(t, err)
	assert.NotNil(t, wrapper)
	assert.NotNil(t, wrapper.Logger)

	safeLogger := wrapper.GetSafe()
	assert.NotNil(t, safeLogger)

	safeLogger.Info("development with safe logger")
}

// TestNewProductionWithSafe 測試帶SafeLogger的生產環境logger
func TestNewProductionWithSafe(t *testing.T) {
	wrapper, err := NewProductionWithSafe()
	assert.NoError(t, err)
	assert.NotNil(t, wrapper)
	assert.NotNil(t, wrapper.Logger)

	safeLogger := wrapper.GetSafe()
	assert.NotNil(t, safeLogger)

	safeLogger.Info("production with safe logger")
}

// TestNew_DifferentLevels 測試不同日誌級別
func TestNew_DifferentLevels(t *testing.T) {
	levels := []string{"debug", "info", "warn", "error"}

	for _, level := range levels {
		t.Run(level, func(t *testing.T) {
			cfg := Config{
				Level:   level,
				Console: true,
			}

			logger, err := New(cfg)
			assert.NoError(t, err)
			assert.NotNil(t, logger)
		})
	}
}
