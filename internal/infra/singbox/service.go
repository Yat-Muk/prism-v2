package singbox

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"go.uber.org/zap"

	"github.com/Yat-Muk/prism-v2/internal/domain/singbox"
	infraFirewall "github.com/Yat-Muk/prism-v2/internal/infra/firewall"
	"github.com/Yat-Muk/prism-v2/internal/infra/system"
	"github.com/Yat-Muk/prism-v2/internal/pkg/appctx"
)

const ServiceName = "sing-box"

type Service struct {
	systemd         system.SystemdManager
	log             *zap.Logger
	firewallManager infraFirewall.Manager
	paths           *appctx.Paths
}

func NewService(
	systemd system.SystemdManager,
	log *zap.Logger,
	firewallManager infraFirewall.Manager,
	paths *appctx.Paths,
) *Service {
	return &Service{
		systemd:         systemd,
		log:             log,
		firewallManager: firewallManager,
		paths:           paths,
	}
}

// UpdateConfig 將生成的配置寫入磁盤，並執行安全驗證
func (s *Service) UpdateConfig(ctx context.Context, sbCfg *singbox.Config) error {

	// 1. 確保配置目錄存在
	if err := os.MkdirAll(s.paths.ConfigDir, 0755); err != nil {
		return fmt.Errorf("無法創建配置目錄: %w", err)
	}

	configPath := filepath.Join(s.paths.ConfigDir, "config.json")
	s.log.Info("寫入 Sing-box 配置文件", zap.String("path", configPath))

	data, err := json.MarshalIndent(sbCfg, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化配置失敗: %w", err)
	}

	// 2. 寫入文件 (0644 權限)
	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("寫入文件失敗: %w", err)
	}

	// 3. [關鍵修復] 執行帶環境變量的配置驗證
	// 這一步會攔截 sing-box 的日誌輸出，防止汙染 TUI 界面
	if err := s.validateConfigSafe(configPath); err != nil {
		return err
	}

	return nil
}

// validateConfigSafe 安全地驗證配置
// 1. 注入環境變量解決 Deprecated 報錯
// 2. 捕獲 Stderr 防止界面崩壞
func (s *Service) validateConfigSafe(configPath string) error {
	s.log.Info("正在驗證配置有效性...")

	// 查找 sing-box 執行路徑
	binPath, err := exec.LookPath("sing-box")
	if err != nil {
		s.log.Warn("未找到 sing-box 二進制文件，跳過驗證")
		return nil
	}

	cmd := exec.Command(binPath, "check", "-c", configPath)

	// [Fix 1] 注入環境變量，允許舊版配置格式
	// 解決 FATAL[0000] to continuing using this feature... 報錯
	newEnv := append(os.Environ(), "ENABLE_DEPRECATED_SPECIAL_OUTBOUNDS=true")
	cmd.Env = newEnv

	// [Fix 2] 捕獲輸出，嚴禁直接打印到屏幕
	// CombinedOutput 會把 stdout 和 stderr 都抓回來，不會漏到終端上
	output, err := cmd.CombinedOutput()

	if err != nil {
		// 記錄詳細日誌到文件 (供 debug)
		s.log.Error("配置驗證失敗",
			zap.Error(err),
			zap.String("output", string(output)),
		)

		// 返回給 UI 的錯誤信息要簡潔，不要包含大段日誌
		// 可以只截取 output 的最後一行，或者提示查看日誌
		return fmt.Errorf("配置無效，sing-box 拒絕應用。\n(請檢查日誌獲取詳情)")
	}

	s.log.Info("配置驗證通過")
	return nil
}

// Start 啟動服務
func (s *Service) Start(ctx context.Context) error {
	s.log.Info("啟動 Sing-box 服務")
	return s.systemd.Start(ctx, ServiceName)
}

// Stop 停止服務
func (s *Service) Stop(ctx context.Context) error {
	s.log.Info("停止 Sing-box 服務")
	return s.systemd.Stop(ctx, ServiceName)
}

// Restart 重啟服務
func (s *Service) Restart(ctx context.Context) error {
	s.log.Info("重啟 Sing-box 服務")
	// 注意：如果 Systemd 服務文件沒有配置 Environment="ENABLE_DEPRECATED_SPECIAL_OUTBOUNDS=true"
	// 重啟可能仍然會失敗。建議提示用戶更新 systemd unit 文件。
	return s.systemd.Restart(ctx, ServiceName)
}

// Reload 熱重載服務
func (s *Service) Reload(ctx context.Context) error {
	s.log.Info("熱重載 Sing-box 配置")
	return s.systemd.Reload(ctx, ServiceName)
}

// Status 獲取服務狀態
func (s *Service) Status(ctx context.Context) (*system.ServiceStatus, error) {
	return s.systemd.Status(ctx, ServiceName)
}
