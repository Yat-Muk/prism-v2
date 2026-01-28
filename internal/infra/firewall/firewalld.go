package firewall

import (
	"context"
	"fmt"
	"os/exec"
	"sort"
	"strings"
	"sync"

	"go.uber.org/zap"

	"github.com/Yat-Muk/prism-v2/internal/pkg/errors"
)

type FirewalldManager struct {
	log         *zap.Logger
	openedPorts map[int]bool
	mu          sync.Mutex
}

func NewFirewalld(log *zap.Logger) *FirewalldManager {
	return &FirewalldManager{
		log:         log,
		openedPorts: make(map[int]bool),
	}
}

func (f *FirewalldManager) Type() string {
	return "firewalld"
}

func (f *FirewalldManager) Capabilities() Capabilities {
	return Capabilities{
		SupportIPv6:        true,  // firewalld 默認支持
		SupportPortHopping: false, // firewalld 不適合做 DNAT
		SupportComment:     false, // firewalld 不支持規則標記
		SupportBoth:        false,
	}
}

func (f *FirewalldManager) OpenPort(ctx context.Context, port int, protocol string) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	if protocol == "both" {
		if err := f.openSinglePort(ctx, port, "tcp"); err != nil {
			return err
		}
		return f.openSinglePort(ctx, port, "udp")
	}
	return f.openSinglePort(ctx, port, protocol)
}

func (f *FirewalldManager) openSinglePort(ctx context.Context, port int, protocol string) error {
	portStr := fmt.Sprintf("%d/%s", port, strings.ToLower(protocol))

	// 檢查規則是否已存在
	checkCmd := exec.CommandContext(ctx, "firewall-cmd", "--query-port="+portStr)
	if checkCmd.Run() == nil {
		f.openedPorts[port] = true
		return nil
	}

	// 添加規則（臨時）
	cmd := exec.CommandContext(ctx, "firewall-cmd", "--add-port="+portStr)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("添加臨時規則失敗: %w", err)
	}

	// 添加規則（永久）
	cmd = exec.CommandContext(ctx, "firewall-cmd", "--permanent", "--add-port="+portStr)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("添加永久規則失敗: %w", err)
	}

	f.openedPorts[port] = true
	return nil
}

func (f *FirewalldManager) OpenPortRange(ctx context.Context, startPort, endPort int, protocol string) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	rangeStr := fmt.Sprintf("%d-%d/%s", startPort, endPort, strings.ToLower(protocol))

	// 臨時規則
	cmd := exec.CommandContext(ctx, "firewall-cmd", "--add-port="+rangeStr)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("添加臨時範圍規則失敗: %w", err)
	}

	// 永久規則
	cmd = exec.CommandContext(ctx, "firewall-cmd", "--permanent", "--add-port="+rangeStr)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("添加永久範圍規則失敗: %w", err)
	}

	return nil
}

func (f *FirewalldManager) OpenHysteria2PortHopping(ctx context.Context, listenPort, startPort, endPort int) error {
	f.log.Warn("⚠️ firewalld 不支持端口跳躍功能",
		zap.Int("listen", listenPort),
		zap.Int("start", startPort),
		zap.Int("end", endPort),
		zap.String("建議", "切換到 nftables 或 iptables"))
	return errors.New("FIREWALL_UNSUPPORTED", "firewalld 不支持端口跳躍功能，請改用 nftables 或 iptables")
}

func (f *FirewalldManager) GetOpenedPorts() []int {
	f.mu.Lock()
	defer f.mu.Unlock()

	ports := make([]int, 0, len(f.openedPorts))
	for port := range f.openedPorts {
		ports = append(ports, port)
	}
	sort.Ints(ports)
	return ports
}

func (f *FirewalldManager) SaveRules(ctx context.Context) error {
	// firewalld 自動持久化規則
	cmd := exec.CommandContext(ctx, "firewall-cmd", "--reload")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("重載 firewalld 失敗: %w", err)
	}
	f.log.Info("✅ firewalld 規則已重載")
	return nil
}

func (f *FirewalldManager) FlushRules(ctx context.Context) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.log.Info("🔥 重載 firewalld 配置")

	// firewalld 沒有按標記刪除的功能，只能重載配置
	cmd := exec.CommandContext(ctx, "firewall-cmd", "--reload")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("重載失敗: %w", err)
	}

	f.openedPorts = make(map[int]bool)
	f.log.Info("✅ firewalld 已重載")
	return nil
}

// Backend 實現
type firewalldBackend struct{}

func NewFirewalldBackend() Backend {
	return &firewalldBackend{}
}

func (b *firewalldBackend) Name() string {
	return "firewalld"
}

func (b *firewalldBackend) IsAvailable() bool {
	if !commandExists("firewall-cmd") {
		return false
	}

	// 檢查 firewalld 是否運行
	cmd := exec.Command("firewall-cmd", "--state")
	output, err := cmd.Output()
	if err != nil {
		return false
	}

	return strings.TrimSpace(string(output)) == "running"
}

func (b *firewalldBackend) CreateManager(log *zap.Logger) Manager {
	return NewFirewalld(log)
}
