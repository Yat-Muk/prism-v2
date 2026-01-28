package state

import "github.com/Yat-Muk/prism-v2/internal/tui/types"

// SystemState 系統監控數據
type SystemState struct {
	Stats        *types.SystemStats
	ServiceStats *types.ServiceStats
}

// NewSystemState 初始化
func NewSystemState() *SystemState {
	return &SystemState{
		Stats:        nil,
		ServiceStats: nil,
	}
}
