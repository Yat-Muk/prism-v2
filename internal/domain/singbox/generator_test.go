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
	mockProto := &MockProtocol{NameStr: "vless", PortInt: 443}
	factory := &MockFactory{
		protocols: []protocol.Protocol{mockProto},
	}

	t.Run("測試 v1.12+ 生成邏輯", func(t *testing.T) {
		g := NewGenerator("1.12.0", factory)
		sbCfg, err := g.Generate(context.Background(), cfg)
		if err != nil {
			t.Fatalf("Generate 失敗: %v", err)
		}

		// 驗證 1.12+ 移除 Strategy 字段
		if sbCfg.DNS.Strategy != "" {
			t.Error("1.12+ DNS 不應包含 Strategy 字段")
		}

		// 驗證 1.12+ 路由使用 Action
		foundAction := false
		for _, rule := range sbCfg.Route.Rules {
			if rule.Action == "reject" {
				foundAction = true
				break
			}
		}
		if !foundAction {
			t.Error("1.12+ 路由應使用 Action 字段")
		}
	})

	t.Run("測試舊版 (<1.12) 生成邏輯", func(t *testing.T) {
		g := NewGenerator("1.11.0", factory)
		sbCfg, err := g.Generate(context.Background(), cfg)
		if err != nil {
			t.Fatalf("Generate 失敗: %v", err)
		}

		// 驗證舊版保留 Strategy
		if sbCfg.DNS.Strategy == "" {
			t.Error("舊版 DNS 應包含 Strategy 字段")
		}

		// 驗證舊版 Outbound 包含 block
		foundBlock := false
		for _, out := range sbCfg.Outbounds {
			if out["tag"] == "block" {
				foundBlock = true
				break
			}
		}
		if !foundBlock {
			t.Error("舊版 Outbound 應包含 block 標籤")
		}
	})
}

func TestVersionLogic(t *testing.T) {
	tests := []struct {
		version  string
		isLegacy bool
	}{
		{"1.11.0", true},
		{"1.12.0", false},
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
