package config

import (
	"fmt"

	"github.com/Yat-Muk/prism-v2/internal/domain/validator"
	"github.com/google/uuid"
)

const (
	// ConfigVersionLatest 最新配置版本
	ConfigVersionLatest = 2

	// ConfigVersionV1 V1 版本（舊版）
	ConfigVersionV1 = 1
)

// Migrator 配置遷移器
type Migrator struct{}

// NewMigrator 創建遷移器
func NewMigrator() *Migrator {
	return &Migrator{}
}

// MigrateToLatest 自動遷移到最新版本
func (m *Migrator) MigrateToLatest(cfg *Config) (*Config, error) {
	if cfg == nil {
		return nil, fmt.Errorf("配置為空，無法遷移")
	}

	// 已經是最新版本
	if cfg.Version == ConfigVersionLatest {
		return cfg, nil
	}

	// V1 -> V2
	if cfg.Version == ConfigVersionV1 || cfg.Version == 0 {
		return m.migrateV1ToV2(cfg)
	}

	// 未來版本降級（例如從 V3 回退到 V2）
	if cfg.Version > ConfigVersionLatest {
		return nil, fmt.Errorf("配置版本過高 (v%d)，當前程序僅支持 v%d", cfg.Version, ConfigVersionLatest)
	}

	return cfg, nil
}

// migrateV1ToV2 V1 -> V2 遷移邏輯
func (m *Migrator) migrateV1ToV2(oldCfg *Config) (*Config, error) {
	newCfg := *oldCfg // 淺拷貝
	newCfg.Version = ConfigVersionLatest

	// 1. 驗證 UUID
	if oldCfg.UUID == "" || !validator.ValidateUUID(oldCfg.UUID) {
		newCfg.UUID = uuid.New().String()
	}

	// 2. 遷移 AnyTLS PaddingScheme -> PaddingMode
	if len(oldCfg.Protocols.AnyTLS.PaddingScheme) > 0 && oldCfg.Protocols.AnyTLS.PaddingMode == "" {
		newCfg.Protocols.AnyTLS.PaddingMode = "official" // V2 默認值
		newCfg.Protocols.AnyTLS.PaddingScheme = nil      // 清空舊字段
	}

	if len(oldCfg.Protocols.AnyTLSReality.PaddingScheme) > 0 && oldCfg.Protocols.AnyTLSReality.PaddingMode == "" {
		newCfg.Protocols.AnyTLSReality.PaddingMode = "official"
		newCfg.Protocols.AnyTLSReality.PaddingScheme = nil
	}

	// 3. 驗證並修復 SNI 域名
	protocols := []struct {
		sni *string
	}{
		{&newCfg.Protocols.RealityVision.SNI},
		{&newCfg.Protocols.RealityGRPC.SNI},
		{&newCfg.Protocols.AnyTLSReality.SNI},
		{&newCfg.Protocols.ShadowTLS.SNI},
	}

	for _, p := range protocols {
		if *p.sni != "" && !validator.ValidateDomain(*p.sni) {
			*p.sni = "www.microsoft.com" // 回退到默認值
		}
	}

	// 4. 驗證端口範圍
	ports := []struct {
		port *int
	}{
		{&newCfg.Protocols.RealityVision.Port},
		{&newCfg.Protocols.RealityGRPC.Port},
		{&newCfg.Protocols.Hysteria2.Port},
		{&newCfg.Protocols.TUIC.Port},
		{&newCfg.Protocols.AnyTLS.Port},
		{&newCfg.Protocols.AnyTLSReality.Port},
		{&newCfg.Protocols.ShadowTLS.Port},
	}

	for _, p := range ports {
		if *p.port < 1024 || *p.port > 65535 {
			*p.port = 0 // 標記為未設置，後續可由系統重新分配
		}
	}

	return &newCfg, nil
}

// NeedsMigration 檢查是否需要遷移
func (m *Migrator) NeedsMigration(cfg *Config) bool {
	if cfg == nil {
		return false
	}
	return cfg.Version < ConfigVersionLatest
}

// GetMigrationPath 獲取遷移路徑描述
func (m *Migrator) GetMigrationPath(fromVersion int) string {
	if fromVersion >= ConfigVersionLatest {
		return "無需遷移"
	}

	if fromVersion == ConfigVersionV1 || fromVersion == 0 {
		return "V1 -> V2: UUID驗證, AnyTLS PaddingScheme遷移, SNI/端口驗證"
	}

	return fmt.Sprintf("未知遷移路徑 (v%d -> v%d)", fromVersion, ConfigVersionLatest)
}
