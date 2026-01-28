package application

import (
	"context"
	"testing"

	"github.com/Yat-Muk/prism-v2/internal/domain/config"
	"go.uber.org/zap"
)

// MockRepo 模擬倉庫，用於測試 Service 邏輯
type MockRepo struct {
	cfg *config.Config
}

func (m *MockRepo) Load(ctx context.Context) (*config.Config, error) {
	if m.cfg == nil {
		return config.DefaultConfig(), nil
	}
	// 模擬從磁盤讀取（返回副本）
	return m.cfg.DeepCopy(), nil
}

func (m *MockRepo) Save(ctx context.Context, c *config.Config) error {
	// 模擬寫入磁盤
	m.cfg = c.DeepCopy()
	return nil
}

func TestConfigService_UpdateConfig(t *testing.T) {
	// 1. 初始化
	mockRepo := &MockRepo{}
	logger := zap.NewNop()

	// ✅ 修正：使用正確的構造函數參數 (repo, logger)
	svc := NewConfigService(mockRepo, logger)
	ctx := context.Background()

	// 2. 測試：獲取初始配置
	initialCfg, err := svc.GetConfig(ctx)
	if err != nil {
		t.Fatalf("GetConfig failed: %v", err)
	}
	if initialCfg.Version < 2 {
		t.Error("Initial config version invalid")
	}

	// 3. 測試：原子更新
	newPort := 9999
	newUUID := "updated-uuid-123"

	updateErr := svc.UpdateConfig(ctx, func(c *config.Config) error {
		// 修改副本
		c.Server.Port = newPort
		c.UUID = newUUID
		return nil
	})

	if updateErr != nil {
		t.Fatalf("UpdateConfig failed: %v", updateErr)
	}

	// 4. 驗證：Repository 是否被更新
	if mockRepo.cfg.Server.Port != newPort {
		t.Errorf("Port not updated in repo. Want %d, Got %d", newPort, mockRepo.cfg.Server.Port)
	}
	if mockRepo.cfg.UUID != newUUID {
		t.Errorf("UUID not updated in repo. Want %s, Got %s", newUUID, mockRepo.cfg.UUID)
	}

	// 5. 驗證：再次讀取
	reloadedCfg, _ := svc.GetConfig(ctx)
	if reloadedCfg.Server.Port != newPort {
		t.Error("GetConfig returned stale data after update")
	}
}

func TestConfigService_SaveWithDefaults(t *testing.T) {
	mockRepo := &MockRepo{}
	logger := zap.NewNop()
	svc := NewConfigService(mockRepo, logger)
	ctx := context.Background()

	// 創建一個缺省配置 (Hysteria2 密碼為空)
	cfg := config.DefaultConfig()
	cfg.Password = "root-password"
	cfg.Protocols.Hysteria2.Password = ""

	// 保存並觸發 FillDefaults
	if err := svc.SaveWithDefaults(ctx, cfg); err != nil {
		t.Fatalf("SaveWithDefaults failed: %v", err)
	}

	// 驗證是否自動填充
	savedCfg, _ := svc.GetConfig(ctx)
	if savedCfg.Protocols.Hysteria2.Password != "root-password" {
		t.Errorf("FillDefaults failed to inherit password. Got: '%s'", savedCfg.Protocols.Hysteria2.Password)
	}
}

// 測試並發安全性 (防止競態條件)
func TestConfigService_Concurrency(t *testing.T) {
	mockRepo := &MockRepo{}
	logger := zap.NewNop()
	svc := NewConfigService(mockRepo, logger)
	ctx := context.Background()

	// 模擬並發讀寫
	// 如果沒有鎖，可能會導致 map 並發讀寫 panic 或數據不一致
	done := make(chan bool)
	concurrency := 10

	for i := 0; i < concurrency; i++ {
		go func() {
			// 讀取
			svc.GetConfig(ctx)

			// 寫入
			svc.UpdateConfig(ctx, func(c *config.Config) error {
				c.Server.Port++
				return nil
			})

			done <- true
		}()
	}

	// 等待所有 goroutine 完成
	for i := 0; i < concurrency; i++ {
		<-done
	}

	// 驗證最終狀態
	finalCfg, _ := svc.GetConfig(ctx)
	t.Logf("Final Port after concurrent updates: %d", finalCfg.Server.Port)
}
