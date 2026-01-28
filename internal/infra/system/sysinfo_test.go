package system

import (
	"testing"
	"time"

	"go.uber.org/zap"
)

// Mock Logger for testing
func newTestLogger() *zap.Logger {
	return zap.NewNop()
}

func TestNewSystemInfo(t *testing.T) {
	logger := newTestLogger()
	sysInfo := NewSystemInfo(logger)

	if sysInfo == nil {
		t.Fatal("NewSystemInfo returned nil")
	}
	if sysInfo.startTime.IsZero() {
		t.Error("startTime is zero")
	}
}

func TestGetStats(t *testing.T) {
	logger := newTestLogger()
	sysInfo := NewSystemInfo(logger)

	// 模擬一點時間流逝，確保速率計算不為 0
	time.Sleep(100 * time.Millisecond)

	stats, err := sysInfo.GetStats()
	if err != nil {
		t.Fatalf("GetStats returned error: %v", err)
	}
	if stats == nil {
		t.Fatal("GetStats returned nil stats")
	}

	// 1. 驗證基礎信息 (這些字段剛被修復，必須存在)
	t.Logf("Detected OS: %s", stats.OS)
	if stats.OS == "" {
		t.Error("OS is empty")
	}

	t.Logf("Detected Hostname: %s", stats.Hostname)
	if stats.Hostname == "" {
		t.Error("Hostname is empty (check sysinfo.go GetStats implementation)")
	}

	t.Logf("Detected Kernel: %s", stats.Kernel)
	if stats.Kernel == "" || stats.Kernel == "Unknown" {
		t.Log("Warning: Kernel version could not be detected (might be expected in some envs)")
	}

	// 2. 驗證資源數據
	t.Logf("CPU Model: %s", stats.CPUModel)
	// 注意：某些極簡 Docker 容器可能讀不到 CPU 型號，這裡只做警告
	if stats.CPUModel == "" {
		t.Log("Warning: CPUModel is empty")
	}

	t.Logf("Memory Total: %d bytes", stats.MemoryTotal)
	if stats.MemoryTotal == 0 {
		t.Error("MemoryTotal is 0")
	}

	t.Logf("Disk Total: %d bytes", stats.DiskTotal)
	if stats.DiskTotal == 0 {
		t.Log("Warning: DiskTotal is 0 (might happen in read-only containers)")
	}

	// 3. 驗證網絡數據 (剛修復的字段)
	t.Logf("NetSentTotal: %d, NetRecvTotal: %d", stats.NetSentTotal, stats.NetRecvTotal)
	// 只要不是負數即可，剛啟動可能為 0
	if stats.NetSentTotal < 0 || stats.NetRecvTotal < 0 {
		t.Error("Network total counters are negative")
	}

	// 4. 驗證負載
	t.Logf("LoadAvg: %s", stats.LoadAvg)
	if stats.LoadAvg == "" {
		t.Error("LoadAvg is empty")
	}
}

func TestFormatBytes(t *testing.T) {
	tests := []struct {
		input    uint64
		expected string
	}{
		{0, "0 B"},
		{500, "500 B"},
		{1024, "1.00 KB"},
		{1536, "1.50 KB"},
		{1048576, "1.00 MB"},
		{1073741824, "1.00 GB"},
	}

	for _, tt := range tests {
		result := formatBytes(tt.input)
		if result != tt.expected {
			t.Errorf("formatBytes(%d) = %s; want %s", tt.input, result, tt.expected)
		}
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		input    time.Duration
		expected string
	}{
		{30 * time.Second, "30秒"},
		{2 * time.Minute, "2分鐘"},
		{1*time.Hour + 30*time.Minute, "1小時30分鐘"},
		{25*time.Hour + 10*time.Minute, "1天1小時"},
	}

	for _, tt := range tests {
		result := formatDuration(tt.input)
		if result != tt.expected {
			t.Errorf("formatDuration(%v) = %s; want %s", tt.input, result, tt.expected)
		}
	}
}
