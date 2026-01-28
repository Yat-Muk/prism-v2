package logger

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

// TestMaskSensitive_BasicPatterns 測試基本脫敏模式
func TestMaskSensitive_BasicPatterns(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		shouldMask    bool
		maskIndicator string // 脫敏後應該包含的標記
	}{
		{
			name:          "GitHub Token",
			input:         "ghp_1234567890abcdefghijklmnopqrstuvwxyz",
			shouldMask:    true,
			maskIndicator: "MASKED",
		},
		{
			name:          "EAB Key",
			input:         "kid_1234567890abcdef",
			shouldMask:    true,
			maskIndicator: "MASKED",
		},
		{
			name:          "UUID",
			input:         "user_id: 550e8400-e29b-41d4-a716-446655440000",
			shouldMask:    true,
			maskIndicator: "-UUID-",
		},
		{
			name:          "Password",
			input:         "password: mySecretPass123",
			shouldMask:    true,
			maskIndicator: "MASKED",
		},
		{
			name:          "Email - 部分脫敏",
			input:         "user@example.com",
			shouldMask:    true,
			maskIndicator: "@", // Email保留@符號
		},
		{
			name:          "Normal text",
			input:         "This is a normal log message",
			shouldMask:    false,
			maskIndicator: "This is a normal log message",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := MaskSensitive(tt.input)

			if tt.shouldMask {
				// 驗證已經被脫敏（結果應該不同於輸入）
				assert.NotEqual(t, tt.input, result, "應該進行脫敏")
			}

			// 驗證包含脫敏標記
			assert.Contains(t, result, tt.maskIndicator, "應該包含脫敏標記")
		})
	}
}

// TestMaskSensitive_MultiplePatterns 測試多個敏感信息
func TestMaskSensitive_MultiplePatterns(t *testing.T) {
	input := `
		Password: secret123
		UUID: 550e8400-e29b-41d4-a716-446655440000
	`

	result := MaskSensitive(input)

	// 驗證已經脫敏
	assert.NotEqual(t, input, result, "應該進行脫敏")

	// 驗證包含脫敏標記
	assert.Contains(t, result, "MASKED")
	assert.Contains(t, result, "-UUID-")
}

// TestMaskSensitive_NoSensitiveData 測試無敏感信息
func TestMaskSensitive_NoSensitiveData(t *testing.T) {
	input := "This is a normal log message"
	result := MaskSensitive(input)
	assert.Equal(t, input, result, "無敏感信息時應該保持不變")
}

// TestMaskSensitive_EmptyString 測試空字符串
func TestMaskSensitive_EmptyString(t *testing.T) {
	result := MaskSensitive("")
	assert.Equal(t, "", result)
}

// TestNewSafeLogger 測試SafeLogger創建
func TestNewSafeLogger(t *testing.T) {
	logger, err := zap.NewDevelopment()
	assert.NoError(t, err)

	safeLogger := NewSafeLogger(logger)
	assert.NotNil(t, safeLogger)
}

// TestSafeLoggerImpl_BasicLogging 測試基本日誌功能
func TestSafeLoggerImpl_BasicLogging(t *testing.T) {
	logger, err := zap.NewDevelopment()
	assert.NoError(t, err)

	safeLogger := NewSafeLogger(logger)

	// 這些方法不應該panic
	t.Run("Debug", func(t *testing.T) {
		assert.NotPanics(t, func() {
			safeLogger.Debug("debug message")
		})
	})

	t.Run("Info", func(t *testing.T) {
		assert.NotPanics(t, func() {
			safeLogger.Info("info message")
		})
	})

	t.Run("Warn", func(t *testing.T) {
		assert.NotPanics(t, func() {
			safeLogger.Warn("warn message")
		})
	})

	t.Run("Error", func(t *testing.T) {
		assert.NotPanics(t, func() {
			safeLogger.Error("error message")
		})
	})
}

// TestSafeLoggerImpl_FormattedLogging 測試格式化日誌
func TestSafeLoggerImpl_FormattedLogging(t *testing.T) {
	logger, err := zap.NewDevelopment()
	assert.NoError(t, err)

	safeLogger := NewSafeLogger(logger)

	t.Run("Debugf", func(t *testing.T) {
		assert.NotPanics(t, func() {
			safeLogger.Debugf("debug: %s", "test")
		})
	})

	t.Run("Infof", func(t *testing.T) {
		assert.NotPanics(t, func() {
			safeLogger.Infof("info: %s", "test")
		})
	})

	t.Run("Warnf", func(t *testing.T) {
		assert.NotPanics(t, func() {
			safeLogger.Warnf("warn: %s", "test")
		})
	})

	t.Run("Errorf", func(t *testing.T) {
		assert.NotPanics(t, func() {
			safeLogger.Errorf("error: %s", "test")
		})
	})
}

// TestSafeLoggerImpl_WithSensitiveData 測試帶敏感數據的日誌
func TestSafeLoggerImpl_WithSensitiveData(t *testing.T) {
	logger, err := zap.NewDevelopment()
	assert.NoError(t, err)

	safeLogger := NewSafeLogger(logger)

	// 記錄包含敏感信息的日誌（不應該panic）
	assert.NotPanics(t, func() {
		safeLogger.Infow("User login",
			"password", "secret123",
			"token", "bearer_abc123",
			"uuid", "550e8400-e29b-41d4-a716-446655440000",
		)
	})
}

// TestMaskSensitive_RealWorldExamples 測試真實場景
func TestMaskSensitive_RealWorldExamples(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "API請求日誌",
			input: "API request: POST /api/login password=secret123",
		},
		{
			name:  "數據庫連接",
			input: "Connecting to DB: user=admin password=dbpass123",
		},
		{
			name:  "Token刷新",
			input: "Token refresh: uuid=550e8400-e29b-41d4-a716-446655440000",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := MaskSensitive(tt.input)
			// 驗證已脫敏（結果不同）
			assert.NotEqual(t, tt.input, result, "應該進行脫敏")
			// 結果不應該為空
			assert.NotEmpty(t, result)
		})
	}
}
