package application

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"

	"go.uber.org/zap"

	domainConfig "github.com/Yat-Muk/prism-v2/internal/domain/config"
	"github.com/Yat-Muk/prism-v2/internal/domain/singbox"
	infraFirewall "github.com/Yat-Muk/prism-v2/internal/infra/firewall"
	infraSingbox "github.com/Yat-Muk/prism-v2/internal/infra/singbox"
	"github.com/Yat-Muk/prism-v2/internal/pkg/appctx"
)

type SingboxService struct {
	generator       singbox.Generator
	service         *infraSingbox.Service
	firewallManager infraFirewall.Manager
	paths           *appctx.Paths
	log             *zap.Logger
}

func NewSingboxService(
	generator singbox.Generator,
	service *infraSingbox.Service,
	firewallManager infraFirewall.Manager,
	paths *appctx.Paths,
	log *zap.Logger,
) *SingboxService {
	return &SingboxService{
		generator:       generator,
		service:         service,
		firewallManager: firewallManager,
		paths:           paths,
		log:             log,
	}
}

func (s *SingboxService) ApplyConfig(ctx context.Context, cfg *domainConfig.Config) error {
	s.log.Info("開始應用配置到 Sing-box")

	// 1. 生成 Sing-box 配置
	singboxCfg, err := s.generator.Generate(ctx, cfg)
	if err != nil {
		return fmt.Errorf("生成配置失敗: %w", err)
	}
	s.log.Info("✅ 配置結構已生成")

	// 2. 創建臨時文件驗證
	tempFile, err := os.CreateTemp("", "singbox-check-*.json")
	if err != nil {
		return fmt.Errorf("創建臨時文件失敗: %w", err)
	}
	tempPath := tempFile.Name()
	defer os.Remove(tempPath) // 確保清理

	encoder := json.NewEncoder(tempFile)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(singboxCfg); err != nil {
		tempFile.Close()
		return fmt.Errorf("序列化配置失敗: %w", err)
	}
	tempFile.Close()

	// 4. 驗證配置
	s.log.Info("正在驗證配置...")
	checkCmd := exec.CommandContext(ctx, "sing-box", "check", "-c", tempPath)
	checkCmd.Env = append(os.Environ(), "ENABLE_DEPRECATED_SPECIAL_OUTBOUNDS=true")
	if output, err := checkCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("配置無效，拒絕應用:\n%s", string(output))
	}
	s.log.Info("✅ 配置驗證通過")

	// 5. 寫入正式配置文件
	if err := s.service.UpdateConfig(ctx, singboxCfg); err != nil {
		return fmt.Errorf("寫入配置文件失敗: %w", err)
	}

	// 6. 更新防火牆規則 (優化：直接傳入 singboxCfg 對象)
	if err := s.updateFirewallRules(ctx, singboxCfg, cfg); err != nil {
		s.log.Warn("更新防火牆規則失敗", zap.Error(err))
	}

	// 7. 重載服務
	s.log.Info("重載 Sing-box 服務...")
	if err := s.service.Reload(ctx); err != nil {
		s.log.Warn("熱重載失敗，嘗試重啟服務", zap.Error(err))
		if err := s.service.Restart(ctx); err != nil {
			return fmt.Errorf("重啟服務失敗: %w", err)
		}
	} else {
		s.log.Info("✅ Sing-box 服務已熱重載")
	}

	return nil
}

func (s *SingboxService) UpdateConfig(ctx context.Context, sbCfg *singbox.Config) error {
	return s.service.UpdateConfig(ctx, sbCfg)
}

// updateFirewallRules 更新防火牆規則
func (s *SingboxService) updateFirewallRules(ctx context.Context, sbCfg *singbox.Config, domCfg *domainConfig.Config) error {
	if s.firewallManager == nil {
		return nil
	}

	s.log.Info("🔄 正在同步防火牆規則...")

	// 1. Flush 舊規則
	if err := s.firewallManager.FlushRules(ctx); err != nil {
		s.log.Warn("清理舊規則失敗", zap.Error(err))
	}

	// 2. 提取並開放端口 (直接使用內存對象)
	ports := s.extractPorts(sbCfg)
	for _, portInfo := range ports {
		if err := s.firewallManager.OpenPort(ctx, portInfo.Port, portInfo.Protocol); err != nil {
			s.log.Error("開放端口失敗", zap.Int("port", portInfo.Port), zap.Error(err))
		}
	}

	// 3. 處理 Hysteria 2 跳躍端口 (依賴 domain config)
	if domCfg != nil && domCfg.Protocols.Hysteria2.Enabled && domCfg.Protocols.Hysteria2.PortHopping != "" {
		hopping := domCfg.Protocols.Hysteria2.PortHopping
		var start, end int
		if _, err := fmt.Sscanf(hopping, "%d-%d", &start, &end); err == nil {
			mainPort := domCfg.Protocols.Hysteria2.Port
			s.log.Info("配置 Hy2 跳躍端口防火牆", zap.String("range", hopping))
			if err := s.firewallManager.OpenHysteria2PortHopping(ctx, mainPort, start, end); err != nil {
				s.log.Error("跳躍端口設置失敗", zap.Error(err))
			}
		}
	}

	// 4. 保存規則
	_ = s.firewallManager.SaveRules(ctx)

	return nil
}

// portInfo 內部輔助結構
type portInfo struct {
	Port     int
	Protocol string
}

// extractPorts 從 Sing-box 配置中提取需要開放的端口
func (s *SingboxService) extractPorts(sbCfg *singbox.Config) []portInfo {
	var ports []portInfo

	if sbCfg == nil || len(sbCfg.Inbounds) == 0 {
		return ports
	}

	for _, inbound := range sbCfg.Inbounds {
		// 獲取端口
		portVal, ok := inbound["listen_port"]
		if !ok {
			continue
		}

		var port int
		switch v := portVal.(type) {
		case int:
			port = v
		case float64:
			port = int(v)
		default:
			continue
		}

		if port <= 0 || port > 65535 {
			continue
		}

		// 獲取協議類型
		typeVal, _ := inbound["type"].(string)
		protocol := "tcp"

		switch typeVal {
		// UDP 為主的協議，以及需要 UDP 轉發的協議，都建議開啟 both
		case "hysteria2", "tuic", "shadowtls", "naive", "trojan":
			protocol = "both"
		case "vless", "vmess":
			// 檢查是否開啟了 quic 或 grpc 傳輸，這些通常也需要 UDP
			if transport, ok := inbound["transport"].(map[string]interface{}); ok {
				if tType, ok := transport["type"].(string); ok && (tType == "quic" || tType == "grpc") {
					protocol = "both"
				}
			}
		}

		ports = append(ports, portInfo{
			Port:     port,
			Protocol: protocol,
		})
	}

	return ports
}

func (s *SingboxService) Restart(ctx context.Context) error {
	return s.service.Restart(ctx)
}

func (s *SingboxService) Stop(ctx context.Context) error {
	return s.service.Stop(ctx)
}

func (s *SingboxService) Start(ctx context.Context) error {
	return s.service.Start(ctx)
}
