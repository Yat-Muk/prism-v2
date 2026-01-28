package application

import (
	"context"
	"fmt"
	"math/rand"
	"strconv"

	domainConfig "github.com/Yat-Muk/prism-v2/internal/domain/config"
	"github.com/Yat-Muk/prism-v2/internal/domain/protocol"
	"go.uber.org/zap"
)

// PortService 端口管理服務接口
type PortService interface {
	// ResetAllPorts 為所有協議隨機生成端口 (10000-60000)
	ResetAllPorts(ctx context.Context, cfg *domainConfig.Config) (*domainConfig.Config, error)

	// UpdateSinglePort 更新單個協議端口，支持數字輸入或 "random"
	UpdateSinglePort(ctx context.Context, cfg *domainConfig.Config, protoID int, portInput string) (*domainConfig.Config, error)

	// UpdateHy2Hopping 更新 Hysteria2 跳躍端口範圍
	UpdateHy2Hopping(ctx context.Context, cfg *domainConfig.Config, startPort, endPort int) (*domainConfig.Config, error)

	// ClearHy2Hopping 清除 Hysteria2 跳躍端口
	ClearHy2Hopping(ctx context.Context, cfg *domainConfig.Config) (*domainConfig.Config, error)

	// GetPort 根據協議 ID 獲取當前配置中的端口
	GetPort(cfg *domainConfig.Config, protoID int) int
}

// portService 端口管理服務實現
type portService struct {
	log *zap.Logger
}

// NewPortService 創建端口管理服務
func NewPortService(log *zap.Logger) PortService {
	return &portService{
		log: log,
	}
}

// 定義端口設置函數類型
type portSetter func(cfg *domainConfig.Config, port int)

// 核心映射表：將協議 ID 映射到具體的 Config 修改邏輯
// 這消除了巨大的 switch 語句，符合開閉原則
var portStrategy = map[protocol.ID]portSetter{
	protocol.IDRealityVision: func(c *domainConfig.Config, p int) { c.Protocols.RealityVision.Port = p },
	protocol.IDRealityGRPC:   func(c *domainConfig.Config, p int) { c.Protocols.RealityGRPC.Port = p },
	protocol.IDHysteria2:     func(c *domainConfig.Config, p int) { c.Protocols.Hysteria2.Port = p },
	protocol.IDTUIC:          func(c *domainConfig.Config, p int) { c.Protocols.TUIC.Port = p },
	protocol.IDAnyTLS:        func(c *domainConfig.Config, p int) { c.Protocols.AnyTLS.Port = p },
	protocol.IDAnyTLSReality: func(c *domainConfig.Config, p int) { c.Protocols.AnyTLSReality.Port = p },
	protocol.IDShadowTLS:     func(c *domainConfig.Config, p int) { c.Protocols.ShadowTLS.Port = p },
}

// ResetAllPorts 為所有協議隨機生成不衝突的端口
func (s *portService) ResetAllPorts(ctx context.Context, cfg *domainConfig.Config) (*domainConfig.Config, error) {
	s.log.Info("正在重置所有協議端口...")

	newCfg := cfg.DeepCopy()

	// 用於記錄已分配的端口，防止衝突
	usedPorts := make(map[int]bool)
	// 保留系統常用端口
	usedPorts[80] = true
	usedPorts[443] = true
	usedPorts[22] = true

	// 定義隨機端口生成函數
	generateUniquePort := func() int {
		for {
			// 生成 10000 - 60000 之間的端口
			p := rand.Intn(50000) + 10000
			if !usedPorts[p] {
				usedPorts[p] = true
				return p
			}
		}
	}

	// 1. Reality Vision
	newCfg.Protocols.RealityVision.Port = generateUniquePort()

	// 2. Reality gRPC
	newCfg.Protocols.RealityGRPC.Port = generateUniquePort()

	// 3. Hysteria 2
	newCfg.Protocols.Hysteria2.Port = generateUniquePort()
	newCfg.Protocols.Hysteria2.PortHopping = ""

	// 4. TUIC
	newCfg.Protocols.TUIC.Port = generateUniquePort()

	// 5. AnyTLS
	newCfg.Protocols.AnyTLS.Port = generateUniquePort()

	// 6. AnyTLS Reality
	newCfg.Protocols.AnyTLSReality.Port = generateUniquePort()

	// 7. ShadowTLS (需要兩個端口: 監聽端口 + Handshake端口)
	newCfg.Protocols.ShadowTLS.Port = generateUniquePort()
	// ShadowTLS Detour 端口也需要唯一
	// 注意：這裡假設 ShadowTLS 實現中 DetourPort 是導出的並需要配置
	// newCfg.Protocols.ShadowTLS.DetourPort = generateUniquePort()

	s.log.Info("端口重置完成", zap.Int("count", len(usedPorts)-3))
	return newCfg, nil
}

// UpdateSinglePort 重構版：使用 Map 查找策略
func (s *portService) UpdateSinglePort(ctx context.Context, cfg *domainConfig.Config, protoID int, portInput string) (*domainConfig.Config, error) {
	// 1. 驗證 ID
	pID := protocol.ID(protoID)
	setter, exists := portStrategy[pID]
	if !exists {
		return nil, fmt.Errorf("不支持的協議 ID: %d", protoID)
	}

	// 2. 解析端口 (邏輯保持不變，但代碼更乾淨)
	var p int
	if portInput == "random" {
		p = rand.Intn(50000) + 10000
	} else {
		var err error
		p, err = strconv.Atoi(portInput)
		if err != nil {
			return nil, fmt.Errorf("端口必須是數字或 'random'")
		}
		if p < 1024 || p > 65535 {
			return nil, fmt.Errorf("端口範圍必須在 1024-65535 之間")
		}
	}

	// 3. 應用修改
	newCfg := cfg.DeepCopy()
	setter(newCfg, p)

	s.log.Info("已更新協議端口",
		zap.String("protocol", pID.String()), // 使用 String() 獲取可讀名稱
		zap.Int("port", p),
	)

	return newCfg, nil
}

// UpdateHy2Hopping 設置 Hysteria2 端口跳躍範圍
func (s *portService) UpdateHy2Hopping(ctx context.Context, cfg *domainConfig.Config, startPort, endPort int) (*domainConfig.Config, error) {
	if cfg == nil {
		return nil, fmt.Errorf("配置不能為空")
	}

	if startPort < 1024 || endPort > 65535 || endPort <= startPort {
		return nil, fmt.Errorf("跳躍端口範圍必須在 1024-65535 且 start < end")
	}

	// 使用深拷貝
	newCfg := cfg.DeepCopy()
	newCfg.Protocols.Hysteria2.PortHopping = fmt.Sprintf("%d-%d", startPort, endPort)

	s.log.Info("已設置 Hysteria2 端口跳躍",
		zap.Int("start_port", startPort),
		zap.Int("end_port", endPort),
	)

	return newCfg, nil
}

// ClearHy2Hopping 清除 Hysteria2 端口跳躍設置
func (s *portService) ClearHy2Hopping(ctx context.Context, cfg *domainConfig.Config) (*domainConfig.Config, error) {
	if cfg == nil {
		return nil, fmt.Errorf("配置不能為空")
	}

	// 使用深拷貝
	newCfg := cfg.DeepCopy()
	newCfg.Protocols.Hysteria2.PortHopping = ""

	s.log.Info("已清除 Hysteria2 端口跳躍")

	return newCfg, nil
}

// GetPort 實現
func (s *portService) GetPort(cfg *domainConfig.Config, protoID int) int {
	if cfg == nil {
		return 0
	}
	// 使用 switch 進行映射，清晰直觀
	switch protocol.ID(protoID) {
	case protocol.IDRealityVision:
		return cfg.Protocols.RealityVision.Port
	case protocol.IDRealityGRPC:
		return cfg.Protocols.RealityGRPC.Port
	case protocol.IDHysteria2:
		return cfg.Protocols.Hysteria2.Port
	case protocol.IDTUIC:
		return cfg.Protocols.TUIC.Port
	case protocol.IDAnyTLS:
		return cfg.Protocols.AnyTLS.Port
	case protocol.IDAnyTLSReality:
		return cfg.Protocols.AnyTLSReality.Port
	case protocol.IDShadowTLS:
		return cfg.Protocols.ShadowTLS.Port
	default:
		return 0
	}
}
