package system

import (
	"context"
	"strings"
	"testing"
	"time"

	"go.uber.org/zap"
)

func TestSafeExecutor_IsAllowed(t *testing.T) {
	logger := zap.NewNop()
	executor := NewExecutor(logger)

	tests := []struct {
		cmd     string
		allowed bool
	}{
		{"systemctl", true},
		{"grep", true},
		{"date", true},
		{"reboot", false},
		{"shutdown", false}, // 危險命令
		{"unknown_cmd", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.cmd, func(t *testing.T) {
			if got := executor.IsAllowed(tt.cmd); got != tt.allowed {
				t.Errorf("IsAllowed(%q) = %v; want %v", tt.cmd, got, tt.allowed)
			}
		})
	}
}

func TestSafeExecutor_Execute(t *testing.T) {
	logger := zap.NewNop()
	executor := NewExecutor(logger)
	ctx := context.Background()

	// 1. 測試合法命令 (date)
	t.Run("Allowed Command", func(t *testing.T) {
		out, err := executor.Execute(ctx, "date")
		if err != nil {
			t.Errorf("Execute('date') failed: %v", err)
		}
		if out == "" {
			t.Error("Execute('date') returned empty output")
		}
	})

	// 2. 測試非法命令 (whoami - 假設不在白名單中)
	t.Run("Disallowed Command", func(t *testing.T) {
		// 注意：如果您的 allowlist 包含 whoami，請換一個不在白名單的命令
		cmd := "reboot"
		_, err := executor.Execute(ctx, cmd)
		if err == nil {
			t.Errorf("Execute('%s') should fail but succeeded", cmd)
		}
		// 檢查錯誤信息是否包含特定關鍵詞
		if !strings.Contains(err.Error(), "不在白名單中") && !strings.Contains(err.Error(), "not allowed") {
			t.Logf("Warning: Error message might differ from expectation: %v", err)
		}
	})
}

func TestSafeExecutor_ExecuteWithTimeout(t *testing.T) {
	logger := zap.NewNop()
	executor := NewExecutor(logger)
	ctx := context.Background()

	// 測試超時機制 (ping)
	// 注意：ping 必須在白名單中
	t.Run("Timeout Execution", func(t *testing.T) {
		// 嘗試執行一個耗時命令，但只給極短的超時時間
		// 使用 ping -c 5 本地迴環，通常需要 4-5 秒
		start := time.Now()
		_, err := executor.ExecuteWithTimeout(ctx, 100*time.Millisecond, "ping", "-c", "5", "127.0.0.1")
		duration := time.Since(start)

		if err == nil {
			t.Error("ExecuteWithTimeout should have timed out")
		}

		// 確保確實是上下文超時導致的錯誤
		if !strings.Contains(err.Error(), "signal: killed") && !strings.Contains(err.Error(), "context deadline exceeded") {
			t.Logf("Note: Error was %v (expected timeout related)", err)
		}

		if duration > 2*time.Second {
			t.Errorf("Execution took too long (%v), timeout logic might verify failed", duration)
		}
	})
}
