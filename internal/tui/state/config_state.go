package state

import (
	domainConfig "github.com/Yat-Muk/prism-v2/internal/domain/config"
)

type ConfigState struct {
	Config            *domainConfig.Config
	ConfirmMode       bool  // 重置確認模式
	ExitConfirmMode   bool  // [新增] 退出確認模式
	HasUnsavedChanges bool  // [新增] 標記是否有未保存的修改 (內存緩存)
	EnabledProtocols  []int // UI 狀態緩存
	Dirty             bool
}

// NewConfigState 構造函數
func NewConfigState(cfg *domainConfig.Config) *ConfigState {
	if cfg == nil {
		cfg = domainConfig.DefaultConfig()
	}
	s := &ConfigState{
		Config:            cfg,
		ConfirmMode:       false,
		ExitConfirmMode:   false,
		HasUnsavedChanges: false,
		Dirty:             false,
	}
	s.SyncProtocolsFromConfig()
	return s
}

// GetConfig 獲取配置
func (s *ConfigState) GetConfig() *domainConfig.Config {
	if s.Config == nil {
		s.Config = domainConfig.DefaultConfig()
	}
	return s.Config
}

// UpdateConfig 更新配置 (從磁盤加載後調用)
func (s *ConfigState) UpdateConfig(cfg *domainConfig.Config) {
	if cfg == nil {
		cfg = domainConfig.DefaultConfig()
	}
	s.Config = cfg

	s.ConfirmMode = false
	s.ExitConfirmMode = false

	s.SyncProtocolsFromConfig()
	s.SyncPortsToMap(NewPortState())
}

// SyncProtocolsFromConfig 同步協議列表
func (s *ConfigState) SyncProtocolsFromConfig() {
	if s.Config == nil {
		s.EnabledProtocols = []int{}
		return
	}

	var list []int
	p := s.Config.Protocols

	if p.RealityVision.Enabled {
		list = append(list, 1)
	}
	if p.RealityGRPC.Enabled {
		list = append(list, 2)
	}
	if p.Hysteria2.Enabled {
		list = append(list, 3)
	}
	if p.TUIC.Enabled {
		list = append(list, 4)
	}
	if p.AnyTLS.Enabled {
		list = append(list, 5)
	}
	if p.AnyTLSReality.Enabled {
		list = append(list, 6)
	}
	if p.ShadowTLS.Enabled {
		list = append(list, 7)
	}

	s.EnabledProtocols = list
}

// SyncPortsToMap 从 Config 提取端口并填充到 PortState
func (c *ConfigState) SyncPortsToMap(portState *PortState) {
	if c.Config == nil {
		portState.ClearPorts()
		return
	}

	// 初始化 map，确保是 int 类型
	portState.CurrentPorts = make(map[int]int)

	// 协议 ID 定义：
	// 1=Reality Vision, 2=Reality gRPC, 3=Hysteria2, 4=TUIC,
	// 5=AnyTLS, 6=AnyTLS Reality, 7=ShadowTLS

	// 直接赋值 int，不再使用 fmt.Sprintf
	if c.Config.Protocols.RealityVision.Port > 0 {
		portState.CurrentPorts[1] = c.Config.Protocols.RealityVision.Port
	}

	if c.Config.Protocols.RealityGRPC.Port > 0 {
		portState.CurrentPorts[2] = c.Config.Protocols.RealityGRPC.Port
	}

	if c.Config.Protocols.Hysteria2.Port > 0 {
		portState.CurrentPorts[3] = c.Config.Protocols.Hysteria2.Port
	}

	if c.Config.Protocols.TUIC.Port > 0 {
		portState.CurrentPorts[4] = c.Config.Protocols.TUIC.Port
	}

	if c.Config.Protocols.AnyTLS.Port > 0 {
		portState.CurrentPorts[5] = c.Config.Protocols.AnyTLS.Port
	}

	if c.Config.Protocols.AnyTLSReality.Port > 0 {
		portState.CurrentPorts[6] = c.Config.Protocols.AnyTLSReality.Port
	}

	if c.Config.Protocols.ShadowTLS.Port > 0 {
		portState.CurrentPorts[7] = c.Config.Protocols.ShadowTLS.Port
	}

	// 如果 PortState 有 Hy2HoppingRange 字段，也可以在这里同步
	portState.Hy2HoppingRange = c.Config.Protocols.Hysteria2.PortHopping
}
