package state

import (
	domainConfig "github.com/Yat-Muk/prism-v2/internal/domain/config"
)

// RoutingState 路由狀態
type RoutingState struct {
	// WARP 狀態
	WarpEnabled bool
	WarpGlobal  bool
	WarpDomains []string

	// Socks5 狀態
	Socks5InboundEnabled  bool
	Socks5OutboundEnabled bool

	// IPv6 狀態
	IPv6Enabled bool
	IPv6Global  bool

	// DNS 狀態
	DNSEnabled bool
	DNSServer  string

	// SNI 代理狀態
	SNIEnabled  bool
	SNITargetIP string

	// 編輯狀態
	EditingType   string // "warp", "socks5_inbound", "socks5_outbound", "ipv6", "dns", "sni"
	EditingField  string // "domains", "port", "auth", "ip", etc.
	ConfirmDelete bool
}

// NewRoutingState 創建新的路由狀態
func NewRoutingState() *RoutingState {
	return &RoutingState{
		WarpEnabled:           false,
		WarpGlobal:            false,
		WarpDomains:           []string{},
		Socks5InboundEnabled:  false,
		Socks5OutboundEnabled: false,
		IPv6Enabled:           false,
		IPv6Global:            false,
		DNSEnabled:            false,
		DNSServer:             "",
		SNIEnabled:            false,
		SNITargetIP:           "",
		EditingType:           "",
		EditingField:          "",
		ConfirmDelete:         false,
	}
}

// LoadFromConfig 方法保留，因為它們包含對象映射邏輯
func (s *RoutingState) LoadWARPConfig(cfg *domainConfig.WARPConfig) {
	if cfg == nil {
		return
	}
	s.WarpEnabled = cfg.Enabled
	s.WarpGlobal = cfg.Global
	s.WarpDomains = append([]string(nil), cfg.Domains...)
}

func (s *RoutingState) LoadSocks5Config(cfg *domainConfig.Socks5Config) {
	if cfg == nil {
		return
	}
	s.Socks5InboundEnabled = cfg.Inbound.Enabled
	s.Socks5OutboundEnabled = cfg.Outbound.Enabled
}

func (s *RoutingState) LoadIPv6Config(cfg *domainConfig.IPv6SplitConfig) {
	if cfg == nil {
		return
	}
	s.IPv6Enabled = cfg.Enabled
	s.IPv6Global = cfg.Global
}

func (s *RoutingState) LoadDNSConfig(cfg *domainConfig.DNSRoutingConfig) {
	if cfg == nil {
		return
	}
	s.DNSEnabled = cfg.Enabled
	s.DNSServer = cfg.Server
}

func (s *RoutingState) LoadSNIProxyConfig(cfg *domainConfig.SNIProxyConfig) {
	if cfg == nil {
		return
	}
	s.SNIEnabled = cfg.Enabled
	s.SNITargetIP = cfg.TargetIP
}

// StartEditing 開始編輯
func (s *RoutingState) StartEditing(editType, field string) {
	s.EditingType = editType
	s.EditingField = field
}

// StopEditing 停止編輯
func (s *RoutingState) StopEditing() {
	s.EditingType = ""
	s.EditingField = ""
}

// IsEditing 是否在編輯中
func (s *RoutingState) IsEditing() bool {
	return s.EditingType != ""
}
