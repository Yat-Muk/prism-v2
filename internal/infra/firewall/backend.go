package firewall

import "go.uber.org/zap"

// Backend 防火牆後端檢測接口（只負責檢測和創建）
type Backend interface {
	// Name 返回後端名稱
	Name() string

	// IsAvailable 檢查後端是否可用
	IsAvailable() bool

	// CreateManager 創建防火牆管理器實例
	CreateManager(log *zap.Logger) Manager
}
