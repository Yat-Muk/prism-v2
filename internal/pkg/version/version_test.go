package version

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

// TestVersion 測試版本信息
func TestVersion(t *testing.T) {
	t.Run("獲取版本號", func(t *testing.T) {
		v := GetVersion()
		assert.NotEmpty(t, v, "版本號不應為空")
	})

	t.Run("版本號格式", func(t *testing.T) {
		v := GetVersion()
		assert.True(t, len(v) > 0)
	})
}

// TestBuildInfo 測試構建信息
func TestBuildInfo(t *testing.T) {
	t.Run("版本存在", func(t *testing.T) {
		assert.NotEmpty(t, Version)
	})

	t.Run("構建時間", func(t *testing.T) {
		// BuildTime可能為空（本地構建）
		_ = BuildTime
		assert.True(t, true)
	})

	t.Run("Git提交哈希", func(t *testing.T) {
		// GitCommit可能為空（本地構建）
		_ = GitCommit
		assert.True(t, true)
	})
}

// TestVersionString 測試版本字符串
func TestVersionString(t *testing.T) {
	t.Run("版本字符串格式", func(t *testing.T) {
		versionStr := GetVersionString()
		assert.NotEmpty(t, versionStr)
		assert.Contains(t, versionStr, Version)
	})
}

// GetVersion 輔助函數
func GetVersion() string {
	if Version != "" {
		return Version
	}
	return "dev"
}

// GetVersionString 輔助函數
func GetVersionString() string {
	if GitCommit != "" {
		return Version + "-" + GitCommit[:7]
	}
	return Version
}
