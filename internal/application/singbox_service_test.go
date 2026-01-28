package application

import (
	"context"
	"testing"

	domainConfig "github.com/Yat-Muk/prism-v2/internal/domain/config"
	"github.com/Yat-Muk/prism-v2/internal/domain/singbox"
	infraFirewall "github.com/Yat-Muk/prism-v2/internal/infra/firewall"
	"go.uber.org/zap"
)

// MockFirewall 模擬防火牆管理器
type MockFirewall struct {
	openedPorts []struct {
		port  int
		proto string
	}
	flushed     bool
	saved       bool
	hoppingRule *struct {
		main  int
		start int
		end   int
	}
}

// ----------------------------------------------------------------------------
// 實作 firewall.Manager 接口的所有方法
// ----------------------------------------------------------------------------

func (m *MockFirewall) Type() string {
	return "mock"
}

func (m *MockFirewall) GetOpenedPorts() []int {
	var ports []int
	for _, p := range m.openedPorts {
		ports = append(ports, p.port)
	}
	return ports
}

func (m *MockFirewall) OpenPort(ctx context.Context, port int, protocol string) error {
	m.openedPorts = append(m.openedPorts, struct {
		port  int
		proto string
	}{port, protocol})
	return nil
}

// ✅ 修正：新增 OpenPortRange 方法
func (m *MockFirewall) OpenPortRange(ctx context.Context, startPort, endPort int, protocol string) error {
	return nil // 測試中暫不驗證範圍開放
}

func (m *MockFirewall) OpenHysteria2PortHopping(ctx context.Context, listenPort, startPort, endPort int) error {
	m.hoppingRule = &struct {
		main  int
		start int
		end   int
	}{listenPort, startPort, endPort}
	return nil
}

func (m *MockFirewall) SaveRules(ctx context.Context) error {
	m.saved = true
	return nil
}

func (m *MockFirewall) FlushRules(ctx context.Context) error {
	m.flushed = true
	m.openedPorts = nil
	return nil
}

func (m *MockFirewall) Capabilities() infraFirewall.Capabilities {
	return infraFirewall.Capabilities{
		SupportPortHopping: true,
	}
}

// ----------------------------------------------------------------------------
// 測試案例
// ----------------------------------------------------------------------------

func TestExtractPorts(t *testing.T) {
	svc := &SingboxService{
		log: zap.NewNop(),
	}

	sbCfg := &singbox.Config{
		Inbounds: []singbox.Inbound{
			{
				"type":        "vless",
				"listen_port": 443,
			},
			{
				"type":        "hysteria2",
				"listen_port": 8443,
			},
			{
				"type":        "socks",
				"listen_port": 1080.0,
			},
		},
	}

	ports := svc.extractPorts(sbCfg)

	if len(ports) != 3 {
		t.Fatalf("預期 3 個端口，實際獲得 %d", len(ports))
	}
}

func TestUpdateFirewallRules(t *testing.T) {
	mockFW := &MockFirewall{}
	logger := zap.NewNop()

	svc := NewSingboxService(nil, nil, mockFW, nil, logger)
	ctx := context.Background()

	sbCfg := &singbox.Config{
		Inbounds: []singbox.Inbound{
			{
				"type":        "vless",
				"listen_port": 443,
			},
		},
	}

	domCfg := domainConfig.DefaultConfig()
	domCfg.Protocols.Hysteria2.Enabled = true
	domCfg.Protocols.Hysteria2.Port = 8443
	domCfg.Protocols.Hysteria2.PortHopping = "20000-30000"

	err := svc.updateFirewallRules(ctx, sbCfg, domCfg)
	if err != nil {
		t.Fatalf("updateFirewallRules 失敗: %v", err)
	}

	if !mockFW.flushed {
		t.Error("應先清理舊規則 (Flush)")
	}
	if mockFW.hoppingRule == nil {
		t.Error("未應用端口跳躍規則")
	}
}
