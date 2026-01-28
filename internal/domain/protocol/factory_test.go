package protocol

import (
	"testing"

	"github.com/Yat-Muk/prism-v2/internal/domain/config"
	"github.com/Yat-Muk/prism-v2/internal/pkg/appctx"
)

// TestGetSNI 測試 SNI 的優先級邏輯
func TestGetSNI(t *testing.T) {
	tests := []struct {
		name            string
		certMode        string
		certDomain      string
		configSNI       string
		globalTLSDomain string
		want            string
	}{
		{
			name:       "ACME 模式優先使用 CertDomain",
			certMode:   "acme",
			certDomain: "acme.com",
			configSNI:  "manual.com",
			want:       "acme.com",
		},
		{
			name:      "自簽名模式優先使用配置 SNI",
			certMode:  "self_signed",
			configSNI: "manual.com",
			want:      "manual.com",
		},
		{
			name:            "最終 Fallback 到全局域名",
			certMode:        "self_signed",
			configSNI:       "",
			globalTLSDomain: "global.com",
			want:            "global.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getSNI(tt.certMode, tt.certDomain, tt.configSNI, tt.globalTLSDomain)
			if got != tt.want {
				t.Errorf("getSNI() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestFactory_FromConfig 測試從配置生成協議列表
func TestFactory_FromConfig(t *testing.T) {
	paths := &appctx.Paths{CertDir: "/etc/prism/certs"}
	factory := NewFactory(paths)

	cfg := config.DefaultConfig()
	cfg.UUID = "test-uuid"
	cfg.Password = "test-pass"

	// 1. 測試 Reality Vision
	cfg.Protocols.RealityVision.Enabled = true
	cfg.Protocols.RealityVision.Port = 12345

	// 2. 測試 Hysteria2 (帶寬默認值)
	cfg.Protocols.Hysteria2.Enabled = true
	cfg.Protocols.Hysteria2.UpMbps = 0 // 應觸發默認值 100

	protocols := factory.FromConfig(cfg)

	// 驗證數量 (DefaultConfig 默認可能開啟了多個，根據你的 Default 邏輯判斷)
	if len(protocols) == 0 {
		t.Fatal("應生成至少一個協議實例")
	}

	var foundRV, foundHy2 bool
	for _, p := range protocols {
		switch p.Type() {
		case TypeRealityVision:
			foundRV = true
			if p.Port() != 12345 {
				t.Errorf("RealityVision 端口錯誤: %d", p.Port())
			}
		case TypeHysteria2:
			foundHy2 = true
			hy2 := p.(*Hysteria2)
			if hy2.UpMbps != 100 {
				t.Errorf("Hysteria2 帶寬默認值處理失敗: %d", hy2.UpMbps)
			}
		}
	}

	if !foundRV {
		t.Error("未生成 RealityVision 協議")
	}
	if !foundHy2 {
		t.Error("未生成 Hysteria2 協議")
	}
}

// TestGetTLSDomain 測試全局 TLS 域名推導
func TestGetTLSDomain(t *testing.T) {
	p := &config.ProtocolsConfig{}

	// 初始應返回默認
	if d := getTLSDomain(p); d != "www.bing.com" {
		t.Errorf("預期默認域名，得到 %s", d)
	}

	// 設置 TUIC 為 ACME 模式
	p.TUIC.CertMode = "acme"
	p.TUIC.CertDomain = "tuic-cert.com"

	if d := getTLSDomain(p); d != "tuic-cert.com" {
		t.Errorf("應從 TUIC 推導域名，得到 %s", d)
	}
}
