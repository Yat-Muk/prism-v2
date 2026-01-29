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
	// 智能錯誤處理
	// 在 CI 環境中，寫入文件會成功，但 systemctl daemon-reload 會失敗
	// 我們需要忽略 Systemd 相關的錯誤，只關注文件是否正確生成
	if err != nil {
		errStr := err.Error()
		// 如果是權限不足或無法連接 Systemd，我們視為測試通過（因為我們只測文件生成）
		if strings.Contains(errStr, "daemon-reload") ||
			strings.Contains(errStr, "exit status 1") ||
			strings.Contains(errStr, "permission denied") ||
			strings.Contains(errStr, "no such file or directory") { // 找不到 systemctl 命令
			t.Logf("⚠️ Systemd 命令執行失敗 (在測試環境中屬預期行為): %v", err)
		} else {
			// 如果是其他錯誤（如無法寫入文件），則報錯
			t.Fatalf("Install() failed with unexpected error: %v", err)
		}
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
		{"Capability", "CapabilityBoundingSet="},
		{"NET_ADMIN", "CAP_NET_ADMIN"},
	}

	for _, check := range checks {
		if !strings.Contains(content, check.expected) {
			t.Errorf("Service file missing %s: expected to contain %q", check.name, check.expected)
		}
	}

	t.Logf("Generated Service File Content Verified Successfully")
}
