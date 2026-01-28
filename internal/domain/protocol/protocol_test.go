package protocol

import (
	"testing"
)

// TestShadowTLS_DeepAnalysis 深入測試 ShadowTLS 的入站與出站跳轉邏輯
func TestShadowTLS_DeepAnalysis(t *testing.T) {
	s := NewShadowTLS(30000, "tls-password", "ss-password")
	s.SNI = "www.microsoft.com"
	s.DetourPort = 10001
	s.enabled = true

	// 1. 測試主 Inbound 結構
	t.Run("Inbound_Chain", func(t *testing.T) {
		inbound, err := s.ToSingboxInbound()
		if err != nil {
			t.Fatalf("ShadowTLS Inbound 轉換失敗: %v", err)
		}
		if inbound["type"] != "shadowtls" {
			t.Errorf("類型應為 shadowtls, 得到 %v", inbound["type"])
		}
		// 核心校驗：必須指向正確的 Shadowsocks 入站標籤
		if inbound["detour"] != "shadowtls-ss-in" {
			t.Errorf("Detour 標籤錯誤: %v", inbound["detour"])
		}
	})

	// 2. 測試 Detour Inbound (Shadowsocks)
	t.Run("Detour_Inbound_Security", func(t *testing.T) {
		detourIn := s.GetDetourInbound()
		if detourIn["listen"] != "127.0.0.1" {
			t.Error("安全性漏洞：Detour Shadowsocks 必須僅監聽 127.0.0.1")
		}
		if detourIn["listen_port"] != 10001 {
			t.Errorf("Detour 端口錯誤: %v", detourIn["listen_port"])
		}
	})

	// 3. 測試 Outbound 跳轉
	t.Run("Outbound_Chain", func(t *testing.T) {
		outbound, err := s.ToSingboxOutbound()
		if err != nil {
			t.Fatalf("ShadowTLS Outbound 轉換失敗: %v", err)
		}
		// 在 Sing-box 中，ShadowTLS 的 outbound 其實是一個開啟了 detour 的 shadowsocks
		if outbound["type"] != "shadowsocks" || outbound["detour"] != "shadowtls-detour" {
			t.Errorf("Outbound 跳轉鏈路錯誤: %v -> %v", outbound["type"], outbound["detour"])
		}
	})
}

// TestAnyTLSReality_DetailedValidation 測試 AnyTLS Reality 的填充與 Reality 配置
func TestAnyTLSReality_DetailedValidation(t *testing.T) {
	a := NewAnyTLSReality(40000, "example.com", "pass123")
	a.PublicKey = "my-public-key"
	a.PrivateKey = "my-private-key"
	a.ShortID = "sid01"
	a.PaddingMode = PaddingVideo // 測試影片優化填充模式
	a.enabled = true

	// 1. 驗證 Inbound 填充方案與 Reality 握手
	t.Run("Inbound_Padding_And_Handshake", func(t *testing.T) {
		inbound, err := a.ToSingboxInbound()
		if err != nil {
			t.Fatal(err)
		}

		// 驗證 PaddingScheme (Video 模式應包含 stop=9)
		padding := inbound["padding_scheme"].([]string)
		if padding[0] != "stop=9" {
			t.Errorf("Video 模式填充方案錯誤，首項應為 stop=9, 得到 %s", padding[0])
		}

		// 驗證 Reality 手機目標
		tls := inbound["tls"].(map[string]interface{})
		reality := tls["reality"].(map[string]interface{})
		handshake := reality["handshake"].(map[string]interface{})
		if handshake["server"] != "example.com" || handshake["server_port"] != 443 {
			t.Errorf("Reality 握手目標配置錯誤: %v", handshake)
		}
	})

	// 2. 驗證 Outbound uTLS 與 Reality 公鑰
	t.Run("Outbound_uTLS_Config", func(t *testing.T) {
		outbound, err := a.ToSingboxOutbound()
		if err != nil {
			t.Fatal(err)
		}

		tls := outbound["tls"].(map[string]interface{})
		utls := tls["utls"].(map[string]interface{})
		if utls["enabled"] != true || utls["fingerprint"] != "chrome" {
			t.Errorf("uTLS 配置錯誤: %v", utls)
		}

		reality := tls["reality"].(map[string]interface{})
		if reality["public_key"] != "my-public-key" {
			t.Errorf("Reality 公鑰未正確導出: %v", reality["public_key"])
		}
	})
}

// TestProtocol_ValidationErrors 測試驗證失敗的情況 (殘酷測試)
func TestProtocol_ValidationErrors(t *testing.T) {
	// 測試：當 ShadowTLS 主端口與 Detour 端口相同時應報錯
	s := NewShadowTLS(10000, "p", "p")
	s.DetourPort = 10000
	if err := s.Validate(); err == nil {
		t.Error("預期端口衝突時報錯，但未報錯")
	}

	// 測試：當 Reality 缺少私鑰且已啟用時應報錯
	a := NewAnyTLSReality(443, "sni", "pass")
	a.enabled = true
	a.PrivateKey = ""
	if err := a.Validate(); err == nil {
		t.Error("預期缺少私鑰時報錯，但未報錯")
	}
}
