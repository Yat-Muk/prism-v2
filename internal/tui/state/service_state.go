package state

import (
	"github.com/Yat-Muk/prism-v2/internal/tui/types"
)

// ServiceState 服務狀態
type ServiceState struct {
	AutoStart   bool
	ConfirmStop bool
	HealthCheck *types.HealthCheckResult
}

// HealthCheckStatus 健康檢查狀態
type HealthCheckStatus struct {
	LastCheck  string
	Status     string // "healthy", "warning", "error"
	IssueCount int
}

// NewServiceState 創建新的服務狀態
func NewServiceState() *ServiceState {
	return &ServiceState{
		AutoStart:   false,
		ConfirmStop: false,
		HealthCheck: nil,
	}
}
