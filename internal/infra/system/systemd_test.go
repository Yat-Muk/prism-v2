package system

import (
	"context"
	"os"
	"testing"
	"time"

	"go.uber.org/zap"
)

func TestSystemdManager_Status(t *testing.T) {
	// 1. 檢查是否在 Linux Systemd 環境下運行
	if _, err := os.Stat("/run/systemd/system"); os.IsNotExist(err) {
		t.Skip("Skipping systemd test: /run/systemd/system not found (not a systemd environment)")
	}

	logger := zap.NewNop()
	mgr, err := NewSystemdManager(logger)
	if err != nil {
		t.Fatalf("Failed to connect to systemd bus: %v", err)
	}
	defer mgr.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 2. 選擇一個系統必定存在的服務進行測試 (dbus.service 通常一直運行)
	targetService := "dbus.service"

	// 3. 測試 IsActive
	active, err := mgr.IsActive(ctx, targetService)
	if err != nil {
		t.Errorf("IsActive failed: %v", err)
	}
	t.Logf("Service %s IsActive: %v", targetService, active)

	// 4. 測試 Status (獲取詳細信息)
	status, err := mgr.Status(ctx, targetService)
	if err != nil {
		t.Fatalf("Status failed: %v", err)
	}

	// 5. 驗證關鍵字段解析邏輯
	// systemd.go 中的解析邏輯非常複雜 (interface{} -> uint64 -> string)，這裡必須驗證
	t.Logf("Status: %+v", status)

	if status.Name != targetService {
		t.Errorf("Expected name %s, got %s", targetService, status.Name)
	}

	// dbus 服務應該是 Active 的
	if !status.Active {
		t.Logf("Warning: %s is not active, checking properties might be limited", targetService)
	} else {
		// 如果服務活躍，PID 應該存在
		if status.PID == "" || status.PID == "0" {
			t.Error("Active service should have a PID")
		}
		// 啟動時間應該已解析
		if status.Uptime == "" {
			t.Error("Active service should have Uptime string")
		}
		if status.UptimeDur == 0 {
			t.Error("Active service should have UptimeDur > 0")
		}
	}
}

func TestSystemdManager_EnableDisable(t *testing.T) {
	// 這個測試比較危險，因為它會修改系統狀態。
	// 我們只測試接口調用是否返回錯誤（針對一個不存在的服務），而不去動真的服務。

	if _, err := os.Stat("/run/systemd/system"); os.IsNotExist(err) {
		t.Skip("Skipping systemd test")
	}

	logger := zap.NewNop()
	mgr, err := NewSystemdManager(logger)
	if err != nil {
		t.Fatal(err)
	}
	defer mgr.Close()

	ctx := context.Background()
	dummyService := "prism-test-dummy-nonexistent.service"

	// 對不存在的服務操作應該返回錯誤，或者 Systemd 會忽略
	// 我們主要確保代碼不會 Panic
	err = mgr.Enable(ctx, dummyService)
	if err == nil {
		t.Log("Warning: Enabling non-existent service did not return error (systemd behavior might vary)")
	} else {
		t.Logf("Enable non-existent service correctly returned error: %v", err)
	}
}
