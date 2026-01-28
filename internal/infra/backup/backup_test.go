package backup

import (
	"crypto/rand"
	"encoding/hex" // ✅ 新增：引入 hex 包
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestBackupManager(t *testing.T) {
	// 1. 環境準備
	tempDir := t.TempDir()
	configDir := filepath.Join(tempDir, "config")
	backupDir := filepath.Join(tempDir, "backups")
	keyDir := filepath.Join(tempDir, "keys")

	os.MkdirAll(configDir, 0755)
	os.MkdirAll(backupDir, 0755)
	os.MkdirAll(keyDir, 0700)

	// 創建模擬配置文件
	configPath := filepath.Join(configDir, "config.yaml")
	originalContent := "uuid: 1234-5678\nport: 7890"
	if err := os.WriteFile(configPath, []byte(originalContent), 0644); err != nil {
		t.Fatal(err)
	}

	// 創建模擬密鑰文件
	// ✅ 修復：生成 32 字節隨機數 -> 轉換為 Hex 字符串 -> 寫入文件
	keyPath := filepath.Join(keyDir, "master.key")
	rawKey := make([]byte, 32)
	if _, err := rand.Read(rawKey); err != nil {
		t.Fatal(err)
	}

	// crypto 包通常要求密鑰文件內容為 Hex 字符串 (64字節長度)
	keyHex := hex.EncodeToString(rawKey)

	if err := os.WriteFile(keyPath, []byte(keyHex), 0600); err != nil {
		t.Fatal(err)
	}

	// 2. 初始化 Manager
	policy := RetentionPolicy{
		MaxFiles: 5,
		MaxAge:   30 * 24 * time.Hour,
	}

	mgr, err := NewManager(backupDir, keyPath, policy)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	// 3. 測試：創建備份
	backupTag := "test"
	t.Log("Creating backup...")
	if err := mgr.Backup(configPath, backupTag); err != nil {
		t.Fatalf("Backup failed: %v", err)
	}

	// 驗證列表
	list, err := mgr.List()
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(list) == 0 {
		t.Fatal("Backup list is empty")
	}
	t.Logf("Found %d backups", len(list))

	// 4. 測試：恢復備份
	// 模擬文件損壞/被修改
	if err := os.WriteFile(configPath, []byte("corrupted data"), 0644); err != nil {
		t.Fatal(err)
	}

	targetBackup := list[0].Name
	t.Logf("Restoring from %s...", targetBackup)

	if err := mgr.Restore(targetBackup, configPath); err != nil {
		t.Fatalf("Restore failed: %v", err)
	}

	// 驗證恢復後的內容
	restoredData, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(restoredData) != originalContent {
		t.Errorf("Content mismatch!\nWant: %s\nGot: %s", originalContent, string(restoredData))
	} else {
		t.Log("Content verified successfully")
	}

	// 5. 測試：輪替策略 (Rotation)
	// 修改內存中的策略為只保留 1 個文件
	mgr.retention.MaxFiles = 1

	// 為了確保時間戳差異，稍作等待
	time.Sleep(1 * time.Second)

	// 再次備份，觸發清理
	if err := mgr.Backup(configPath, "rotate"); err != nil {
		t.Fatalf("Rotation backup failed: %v", err)
	}

	list, _ = mgr.List()
	if len(list) != 1 {
		t.Errorf("Rotation failed: expected 1 backup, got %d", len(list))
	} else {
		t.Log("Rotation successful: old backups removed")
	}
}
