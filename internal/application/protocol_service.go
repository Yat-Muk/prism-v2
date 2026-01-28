package application

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"

	domainConfig "github.com/Yat-Muk/prism-v2/internal/domain/config"
	"github.com/Yat-Muk/prism-v2/internal/domain/protocol"
	"go.uber.org/zap"
)

// ProtocolService 協議管理服務接口
type ProtocolService interface {
	ToggleProtocolsFromInput(ctx context.Context, currentEnabled []int, input string) ([]int, error)
	// UpdateConfigWithEnabledProtocols 僅更新配置結構體，不負責保存到磁盤
	UpdateConfigWithEnabledProtocols(cfg *domainConfig.Config, enabledProtocols []int) error
	// 批量更新 SNI
	UpdateAllSNI(cfg *domainConfig.Config, sni string) error
}

// protocolService 協議管理服務實現
type protocolService struct {
	log *zap.Logger
}

func NewProtocolService(log *zap.Logger) ProtocolService {
	return &protocolService{
		log: log,
	}
}

// ---------------------------------------------------------
// 協議開關策略映射表
// 如果未來增加新協議，只需在這裡添加一行映射，無需修改業務邏輯
// ---------------------------------------------------------
type protocolEnabler func(cfg *domainConfig.Config, enabled bool)

var enableStrategy = map[protocol.ID]protocolEnabler{
	protocol.IDRealityVision: func(c *domainConfig.Config, e bool) { c.Protocols.RealityVision.Enabled = e },
	protocol.IDRealityGRPC:   func(c *domainConfig.Config, e bool) { c.Protocols.RealityGRPC.Enabled = e },
	protocol.IDHysteria2:     func(c *domainConfig.Config, e bool) { c.Protocols.Hysteria2.Enabled = e },
	protocol.IDTUIC:          func(c *domainConfig.Config, e bool) { c.Protocols.TUIC.Enabled = e },
	protocol.IDAnyTLS:        func(c *domainConfig.Config, e bool) { c.Protocols.AnyTLS.Enabled = e },
	protocol.IDAnyTLSReality: func(c *domainConfig.Config, e bool) { c.Protocols.AnyTLSReality.Enabled = e },
	protocol.IDShadowTLS:     func(c *domainConfig.Config, e bool) { c.Protocols.ShadowTLS.Enabled = e },
}

// ---------------------------------------------------------
// SNI 更新策略映射表
// ---------------------------------------------------------
type sniSetter func(cfg *domainConfig.Config, sni string)

var sniStrategy = map[protocol.ID]sniSetter{
	protocol.IDRealityVision: func(c *domainConfig.Config, s string) { c.Protocols.RealityVision.SNI = s },
	protocol.IDRealityGRPC:   func(c *domainConfig.Config, s string) { c.Protocols.RealityGRPC.SNI = s },
	protocol.IDHysteria2:     func(c *domainConfig.Config, s string) { c.Protocols.Hysteria2.SNI = s },
	protocol.IDTUIC:          func(c *domainConfig.Config, s string) { c.Protocols.TUIC.SNI = s },
	protocol.IDAnyTLS:        func(c *domainConfig.Config, s string) { c.Protocols.AnyTLS.SNI = s },
	protocol.IDAnyTLSReality: func(c *domainConfig.Config, s string) { c.Protocols.AnyTLSReality.SNI = s },
	protocol.IDShadowTLS:     func(c *domainConfig.Config, s string) { c.Protocols.ShadowTLS.SNI = s },
}

// UpdateConfigWithEnabledProtocols 更新配置對象中的協議開關狀態
func (s *protocolService) UpdateConfigWithEnabledProtocols(cfg *domainConfig.Config, enabledProtocols []int) error {
	if cfg == nil {
		return fmt.Errorf("配置不能為空")
	}

	// 1. 先將所有已知協議設置為 False (重置狀態)
	// 使用 ids.go 中的 AllIDs() 進行遍歷，確保不漏掉任何一個
	for _, id := range protocol.AllIDs() {
		if setter, ok := enableStrategy[id]; ok {
			setter(cfg, false)
		}
	}

	// 2. 根據傳入的列表開啟協議
	for _, idInt := range enabledProtocols {
		id := protocol.ID(idInt)
		if setter, ok := enableStrategy[id]; ok {
			setter(cfg, true)
		} else {
			s.log.Warn("嘗試啟用未知的協議 ID", zap.Int("id", idInt))
		}
	}

	return nil
}

// ToggleProtocolsFromInput 保持不變 (邏輯已經很清晰)
func (s *protocolService) ToggleProtocolsFromInput(ctx context.Context, currentEnabled []int, input string) ([]int, error) {
	selected := make(map[int]bool)
	for _, n := range currentEnabled {
		selected[n] = true
	}

	parts := strings.Split(input, ",")
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		n, err := strconv.Atoi(p)
		if err != nil {
			continue
		}

		if !protocol.ID(n).IsValid() {
			return nil, fmt.Errorf("無效的協議編號: %d", n)
		}

		if selected[n] {
			delete(selected, n)
		} else {
			selected[n] = true
		}
	}

	var result []int
	for n := range selected {
		result = append(result, n)
	}
	sort.Ints(result)

	s.log.Info("協議開關狀態已變更", zap.String("input", input), zap.Ints("result", result))
	return result, nil
}

// UpdateAllSNI 批量更新所有協議的 SNI
func (s *protocolService) UpdateAllSNI(cfg *domainConfig.Config, sni string) error {
	if cfg == nil {
		return fmt.Errorf("配置不能為空")
	}

	for _, id := range protocol.AllIDs() {
		if setter, ok := sniStrategy[id]; ok {
			setter(cfg, sni)
		}
	}

	s.log.Info("已批量更新 SNI", zap.String("new_sni", sni))
	return nil
}
