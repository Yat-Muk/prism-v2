package system

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestServiceInstaller_Install(t *testing.T) {
	// 1. 準備模擬數據
	mockBinPath := "/usr/local/bin/sing-box"
	mockConfigPath := "/etc/sing-box/config.json"

	installer := NewServiceInstaller(mockBinPath, mockConfigPath)

	// 2. 創建臨時目錄模擬系統目錄
	tempDir := t.TempDir()
	servicePath := filepath.Join(tempDir, "sing-box.service")

	// 3. 執行安裝
	err := installer.Install(servicePath)
	if err != nil {
		t.Fatalf("Install() failed: %v", err)
	}

	// 4. 驗證文件是否生成
	if _, err := os.Stat(servicePath); os.IsNotExist(err) {
		t.Fatal("Service file was not created")
	}

	// 5. 驗證文件內容正確性
	contentBytes, err := os.ReadFile(servicePath)
	if err != nil {
		t.Fatalf("Failed to read generated service file: %v", err)
	}
	content := string(contentBytes)

	// 檢查關鍵字段是否存在且正確替換
	checks := []struct {
		name     string
		expected string
	}{
		{"Description", "Description=Sing-box service"},
		{"ExecStart Path", mockBinPath},
		{"Config Path", mockConfigPath},
		{"Restart Policy", "Restart=on-failure"},
		{"Capability", "CapabilityBoundingSet=CAP_NET_ADMIN"},
	}

	for _, check := range checks {
		if !strings.Contains(content, check.expected) {
			t.Errorf("Service file missing %s: expected to contain %q", check.name, check.expected)
		}
	}

	t.Logf("Generated Service File Content:\n%s", content)
}
