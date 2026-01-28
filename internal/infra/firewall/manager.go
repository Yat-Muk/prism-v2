package firewall

import (
	"context"
	"os/exec"

	"go.uber.org/zap"
)

// Manager 防火牆管理器接口
type Manager interface {
	Type() string
	GetOpenedPorts() []int
	OpenPort(ctx context.Context, port int, protocol string) error
	OpenPortRange(ctx context.Context, startPort, endPort int, protocol string) error
	OpenHysteria2PortHopping(ctx context.Context, listenPort, startPort, endPort int) error
	SaveRules(ctx context.Context) error
	FlushRules(ctx context.Context) error
	Capabilities() Capabilities
}

// Capabilities 防火牆能力聲明
type Capabilities struct {
	SupportIPv6        bool
	SupportPortHopping bool
	SupportComment     bool
	SupportBoth        bool
}

// NewManager 工廠函數：自動檢測並返回合適的防火牆實現
func NewManager(log *zap.Logger) Manager {
	// 1. 優先檢測 UFW (Debian/Ubuntu 常見)
	if isCommandAvailable("ufw") && isServiceRunning("ufw") {
		log.Info("檢測到 UFW，使用 UFW 後端")
		return NewUFW(log)
	}

	// 2. 檢測 Firewalld (CentOS/Fedora 常見)
	if isCommandAvailable("firewall-cmd") && isServiceRunning("firewalld") {
		log.Info("檢測到 Firewalld，使用 Firewalld 後端")
		return NewFirewalld(log)
	}

	// 3. 檢測 NFTables (現代 Linux 底層)
	if isCommandAvailable("nft") {
		log.Info("檢測到 nftables，使用 NFTables 後端")
		return NewNFTables(log)
	}

	// 4. 兜底使用 IPTables
	if isCommandAvailable("iptables") {
		log.Info("使用 iptables 後端")
		return NewIPTables(log)
	}

	// 5. 什麼都沒有，返回空實現防止崩潰
	log.Warn("未檢測到支持的防火牆，防火牆功能將失效")
	return &NoOpManager{log: log}
}

// --- 輔助工具 ---

func isCommandAvailable(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

func isServiceRunning(service string) bool {
	cmd := exec.Command("systemctl", "is-active", "--quiet", service)
	return cmd.Run() == nil
}

// --- NoOpManager 空實現 (兜底用) ---

type NoOpManager struct {
	log *zap.Logger
}

func (m *NoOpManager) Type() string                                                    { return "none" }
func (m *NoOpManager) GetOpenedPorts() []int                                           { return []int{} }
func (m *NoOpManager) OpenPort(ctx context.Context, p int, proto string) error         { return nil }
func (m *NoOpManager) OpenPortRange(ctx context.Context, s, e int, proto string) error { return nil }
func (m *NoOpManager) OpenHysteria2PortHopping(ctx context.Context, l, s, e int) error { return nil }
func (m *NoOpManager) SaveRules(ctx context.Context) error                             { return nil }
func (m *NoOpManager) FlushRules(ctx context.Context) error                            { return nil }
func (m *NoOpManager) Capabilities() Capabilities                                      { return Capabilities{} }
