package singbox

import (
	"context"
	"testing"

	domainConfig "github.com/Yat-Muk/prism-v2/internal/domain/config"
	"github.com/Yat-Muk/prism-v2/internal/domain/protocol"
)

// ----------------------------------------------------------------------------
// Mock 對象：必須完全匹配 protocol.Protocol 接口
// ----------------------------------------------------------------------------

type MockProtocol struct {
	NameStr string
	PortInt int
}

func (m *MockProtocol) Name() string        { return m.NameStr }
func (m *MockProtocol) Port() int           { return m.PortInt }
func (m *MockProtocol) Type() protocol.Type { return protocol.Type(m.NameStr) }
func (m *MockProtocol) IsEnabled() bool     { return true }
func (m *MockProtocol) Validate() error     { return nil } // 補全 Validate 方法

func (m *MockProtocol) ToSingboxInbound() (map[string]interface{}, error) {
	return map[string]interface{}{
		"type":        m.NameStr,
		"listen_port": m.PortInt,
	}, nil
}

func (m *MockProtocol) ToSingboxOutbound() (map[string]interface{}, error) {
	return map[string]interface{}{
		"type": m.NameStr,
		"tag":  m.NameStr + "-out",
	}, nil
}

// MockFactory 滿足 protocol.Factory 接口
type MockFactory struct {
	protocols []protocol.Protocol
}

func (f *MockFactory) FromConfig(cfg *domainConfig.Config) []protocol.Protocol {
	return f.protocols
}

// ----------------------------------------------------------------------------
// 測試核心生成流程
// ----------------------------------------------------------------------------

func TestGenerate(t *testing.T) {
	cfg := domainConfig.DefaultConfig()
	cfg.Routing.IPv6Split.Enabled = true
	cfg.Routing.IPv6Split.Domains = []string{"google.com"}
	mockProto := &MockProtocol{NameStr: "vless", PortInt: 443}
	factory := &MockFactory{
		protocols: []protocol.Protocol{mockProto},
	}

	t.Run("測試 v1.12+ 生成邏輯", func(t *testing.T) {
		g := NewGenerator("1.10.0", factory)
		sbCfg, err := g.Generate(context.Background(), cfg)
		if err != nil {
			t.Fatalf("Generate 失敗: %v", err)
		}

		// 安全檢查：確保 DNS 已生成
		if sbCfg.DNS == nil {
			t.Fatal("預期生成 DNS 配置，但得到 nil (請檢查 needDNS 邏輯)")
		}

		// 驗證 1.8+ 移除 Strategy 字段 (結構體中該字段可能存在但為空，或者 DNS Server 定義不同)
		if sbCfg.DNS.Strategy != "" {
			t.Errorf("新版 DNS 不應設置 Strategy 字段，當前值: %s", sbCfg.DNS.Strategy)
		}

		// 驗證 1.8+ 使用 DefaultDomainResolver 而不是 DNS Outbound
		if sbCfg.Route.DefaultDomainResolver == nil {
			t.Error("新版路由應包含 DefaultDomainResolver")
		}
	})

	t.Run("測試舊版 (<1.8) 生成邏輯", func(t *testing.T) {
		g := NewGenerator("1.7.9", factory)
		sbCfg, err := g.Generate(context.Background(), cfg)
		if err != nil {
			t.Fatalf("Generate 失敗: %v", err)
		}

		if sbCfg.DNS == nil {
			t.Fatal("預期生成 DNS 配置，但得到 nil")
		}

		// 驗證舊版保留 Strategy
		if sbCfg.DNS.Strategy == "" {
			t.Error("舊版 DNS 應包含 Strategy 字段")
		}

		// 驗證舊版 Outbound 包含 block 和 dns-out
		foundBlock := false
		foundDNS := false
		for _, out := range sbCfg.Outbounds {
			if out["tag"] == "block" {
				foundBlock = true
			}
			if out["tag"] == "dns-out" {
				foundDNS = true
			}
		}
		if !foundBlock {
			t.Error("舊版 Outbound 應包含 block 標籤")
		}
		if !foundDNS {
			t.Error("舊版 Outbound 應包含 dns-out 標籤")
		}
	})
}

func TestVersionLogic(t *testing.T) {
	tests := []struct {
		version  string
		isLegacy bool
	}{
		{"1.7.9", true},   // 舊版
		{"1.8.0", false},  // 新版邊界
		{"1.11.0", false}, // 新版 (之前是 true，現在改為 false)
		{"1.12.0", false}, // 新版
		{"v1.12.5", false},
		{"unknown", false},
	}

	for _, tt := range tests {
		g := &generator{version: tt.version}
		if got := g.isLegacyCore(); got != tt.isLegacy {
			t.Errorf("版本 %s: isLegacyCore() = %v, 想要 %v", tt.version, got, tt.isLegacy)
		}
	}
}
