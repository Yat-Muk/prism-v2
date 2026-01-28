package firewall

import (
	"go.uber.org/zap"

	"github.com/Yat-Muk/prism-v2/internal/pkg/errors"
)

// DetectAndCreateManager 自動檢測並創建防火牆管理器
func DetectAndCreateManager(log *zap.Logger) (Manager, error) {
	backend, err := DetectBackend()
	if err != nil {
		return nil, err
	}

	log.Info("✅ 檢測到防火牆後端", zap.String("backend", backend.Name()))
	return backend.CreateManager(log), nil
}

// DetectBackend 自動檢測可用的防火牆後端
func DetectBackend() (Backend, error) {
	// 按優先級檢測（nftables > firewalld > ufw > iptables）
	backends := []Backend{
		NewNftablesBackend(),
		NewFirewalldBackend(),
		NewUFWBackend(),
		NewIptablesBackend(),
	}

	for _, backend := range backends {
		if backend.IsAvailable() {
			return backend, nil
		}
	}

	return nil, errors.New("FIREWALL010", "未檢測到可用的防火牆工具 (nftables/firewalld/ufw/iptables)")
}
