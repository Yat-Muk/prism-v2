package config

import (
	"strings"
	"testing"
)

// TestDefaultConfig 測試默認配置生成
func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	// 1. 基礎驗證
	if cfg == nil {
		t.Fatal("DefaultConfig returned nil")
	}
	if cfg.Version < 2 {
		t.Errorf("Expected Version >= 2, got %d", cfg.Version)
	}

	// 2. 驗證服務器基礎配置
	if cfg.Server.Host == "" {
		t.Error("Server.Host should not be empty")
	}
	if cfg.Server.Port <= 0 {
		t.Errorf("Server.Port should be positive, got %d", cfg.Server.Port)
	}

	// 3. 驗證日誌配置
	if cfg.Log.Level != "info" {
		t.Errorf("Expected default Log.Level 'info', got '%s'", cfg.Log.Level)
	}
	if cfg.Log.OutputPath == "" {
		t.Error("Log.OutputPath should not be empty")
	}

	// 4. 驗證關鍵協議默認值
	// Reality Vision (默認啟用)
	if !cfg.Protocols.RealityVision.Enabled {
		t.Error("RealityVision should be enabled by default")
	}
	if cfg.Protocols.RealityVision.PrivateKey == "" {
		t.Error("RealityVision PrivateKey should be generated")
	}
	if cfg.Protocols.RealityVision.ShortID == "" {
		t.Error("RealityVision ShortID should be generated")
	}

	// 5. 驗證全局設置
	if cfg.UUID == "" {
		t.Error("Global UUID should be generated")
	}
	if cfg.Password == "" {
		t.Error("Global Password should be generated")
	}

	// 6. 驗證備份配置
	if !cfg.Backup.Enabled {
		t.Error("Backup should be enabled by default")
	}
	if cfg.Backup.MaxFiles <= 0 {
		t.Error("Backup.MaxFiles should be positive")
	}

	// 7. 驗證證書配置默認值
	if cfg.Certificate.ACMEProvider != "letsencrypt" {
		t.Errorf("Default ACMEProvider should be letsencrypt, got %s", cfg.Certificate.ACMEProvider)
	}
}

// TestFillDefaults 測試默認值填充邏輯
func TestFillDefaults(t *testing.T) {
	cfg := &Config{
		UUID:     "global-uuid",
		Password: "global-password",
		Protocols: ProtocolsConfig{
			Hysteria2: Hysteria2Config{},
			TUIC:      TUICConfig{},
			AnyTLS:    AnyTLSConfig{},
			ShadowTLS: ShadowTLSConfig{},
		},
	}

	// 執行填充
	cfg.FillDefaults()

	// 驗證 Hysteria2 是否繼承了全局密碼
	if cfg.Protocols.Hysteria2.Password != "global-password" {
		t.Errorf("Hysteria2 password should inherit global password, got '%s'", cfg.Protocols.Hysteria2.Password)
	}

	// 驗證 TUIC 是否繼承了全局 UUID 和密碼
	if cfg.Protocols.TUIC.UUID != "global-uuid" {
		t.Errorf("TUIC UUID should inherit global UUID, got '%s'", cfg.Protocols.TUIC.UUID)
	}
	if cfg.Protocols.TUIC.Password != "global-password" {
		t.Errorf("TUIC password should inherit global password")
	}

	// 驗證 AnyTLS 默認用戶名
	if cfg.Protocols.AnyTLS.Username != "prism" {
		t.Errorf("AnyTLS username should default to 'prism', got '%s'", cfg.Protocols.AnyTLS.Username)
	}
}

// TestDeepCopy 測試深拷貝邏輯
func TestDeepCopy(t *testing.T) {
	original := DefaultConfig()
	original.Protocols.RealityVision.SNI = "original.com"
	original.Routing.WARP.Domains = []string{"site1.com", "site2.com"}

	// 執行拷貝
	copied := original.DeepCopy()

	// 驗證內容一致性
	if copied.Protocols.RealityVision.SNI != "original.com" {
		t.Error("DeepCopy failed to copy basic field")
	}
	if len(copied.Routing.WARP.Domains) != 2 {
		t.Error("DeepCopy failed to copy slice")
	}

	// 驗證內存獨立性 (修改副本不應影響原件)
	copied.Protocols.RealityVision.SNI = "modified.com"
	copied.Routing.WARP.Domains[0] = "hacked.com"

	if original.Protocols.RealityVision.SNI == "modified.com" {
		t.Error("DeepCopy is shallow: modifying copy affected original struct")
	}
	if original.Routing.WARP.Domains[0] == "hacked.com" {
		t.Error("DeepCopy is shallow: modifying copy slice affected original slice")
	}
}

// TestPortHoppingValidation 測試端口跳躍格式驗證
func TestPortHoppingValidation(t *testing.T) {
	tests := []struct {
		input   string
		wantErr bool
	}{
		{"20000-30000", false}, // 合法
		{"1000-2000", true},    // 起始端口太小 (<1024)
		{"60000-70000", true},  // 結束端口太大 (>65535)
		{"30000-20000", true},  // 起始 > 結束
		{"invalid", true},      // 格式錯誤
		{"", false},            // 空字符串視為不啟用，合法
	}

	for _, tt := range tests {
		h := Hysteria2Config{PortHopping: tt.input}
		err := h.ValidatePortHopping()
		if (err != nil) != tt.wantErr {
			t.Errorf("ValidatePortHopping(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
		}
	}
}

// TestGetACMEURL 測試 ACME URL 獲取邏輯
func TestGetACMEURL(t *testing.T) {
	c := &CertificateConfig{}

	// 默認 Let's Encrypt
	c.ACMEProvider = "letsencrypt"
	if url := c.GetACMEURL(); !strings.Contains(url, "letsencrypt.org") {
		t.Errorf("Expected LE URL, got %s", url)
	}

	// ZeroSSL
	c.ACMEProvider = "zerossl"
	if url := c.GetACMEURL(); !strings.Contains(url, "zerossl.com") {
		t.Errorf("Expected ZeroSSL URL, got %s", url)
	}

	// 自定義 URL
	c.ACMEURL = "https://custom-ca.com"
	if url := c.GetACMEURL(); url != "https://custom-ca.com" {
		t.Errorf("Expected custom URL, got %s", url)
	}
}
