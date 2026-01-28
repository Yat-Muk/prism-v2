package firewall

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"sort"
	"sync"

	"go.uber.org/zap"
)

const (
	// Filter 表 (inet: 同時支持 IPv4/IPv6)
	NftTableType = "inet"
	NftTableName = "prism"
	NftChainName = "input"

	// NAT 表 (獨立表，防止與系統或其他軟件衝突)
	// 使用 ip 和 ip6 協議族，以獲得最廣泛的內核兼容性
	NftIPv4NatTableName = "prism_nat_v4"
	NftIPv6NatTableName = "prism_nat_v6"
	NftNatChainName     = "prerouting"
)

// NFTablesManager 實現 Manager 接口
type NFTablesManager struct {
	log         *zap.Logger
	openedPorts map[int]bool
	mu          sync.Mutex
}

// NewNFTables 創建管理器實例
func NewNFTables(log *zap.Logger) *NFTablesManager {
	mgr := &NFTablesManager{
		log:         log,
		openedPorts: make(map[int]bool),
	}
	return mgr
}

func (n *NFTablesManager) Type() string {
	return "nftables"
}

func (n *NFTablesManager) Capabilities() Capabilities {
	return Capabilities{
		SupportIPv6:        true,
		SupportPortHopping: true,
		SupportComment:     true,
		SupportBoth:        false,
	}
}

// ensureTableExists 確保基礎 Filter 表和鏈存在
func (n *NFTablesManager) ensureTableExists(ctx context.Context) error {
	// 1. 創建 Filter 表
	cmd := exec.CommandContext(ctx, "nft", "add", "table", NftTableType, NftTableName)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("創建 Filter 表失敗: %s, %w", string(out), err)
	}

	// 2. 創建 Input 鏈
	// priority 0, policy accept
	chainDef := fmt.Sprintf("add chain %s %s %s { type filter hook input priority 0; policy accept; }",
		NftTableType, NftTableName, NftChainName)
	cmd = exec.CommandContext(ctx, "nft", chainDef)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("創建 Filter 鏈失敗: %s, %w", string(out), err)
	}

	return nil
}

// ensureNatTablesExists 確保 NAT 表和鏈存在 (專門用於端口跳躍)
func (n *NFTablesManager) ensureNatTablesExists(ctx context.Context) error {
	// --- IPv4 NAT ---
	// 1. 創建表
	cmd := exec.CommandContext(ctx, "nft", "add", "table", "ip", NftIPv4NatTableName)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("創建 IPv4 NAT 表失敗: %s", string(out))
	}
	// 2. 創建鏈 (dstnat priority -100)
	// add chain ip prism_nat_v4 prerouting { type nat hook prerouting priority dstnat; policy accept; }
	chainDefV4 := fmt.Sprintf("add chain ip %s %s { type nat hook prerouting priority dstnat; policy accept; }",
		NftIPv4NatTableName, NftNatChainName)
	cmd = exec.CommandContext(ctx, "nft", chainDefV4)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("創建 IPv4 NAT 鏈失敗: %s", string(out))
	}

	// --- IPv6 NAT ---
	// 1. 創建表
	cmd = exec.CommandContext(ctx, "nft", "add", "table", "ip6", NftIPv6NatTableName)
	if out, err := cmd.CombinedOutput(); err != nil {
		n.log.Debug("創建 IPv6 NAT 表失敗 (可能不支持 IPv6 NAT)", zap.String("err", string(out)))
		return nil // 不阻斷流程
	}
	// 2. 創建鏈
	chainDefV6 := fmt.Sprintf("add chain ip6 %s %s { type nat hook prerouting priority dstnat; policy accept; }",
		NftIPv6NatTableName, NftNatChainName)
	cmd = exec.CommandContext(ctx, "nft", chainDefV6)
	if out, err := cmd.CombinedOutput(); err != nil {
		n.log.Debug("創建 IPv6 NAT 鏈失敗", zap.String("err", string(out)))
	}

	return nil
}

func (n *NFTablesManager) OpenPort(ctx context.Context, port int, protocol string) error {
	n.mu.Lock()
	defer n.mu.Unlock()

	if err := n.ensureTableExists(ctx); err != nil {
		return err
	}

	if protocol == "both" {
		if err := n.openSinglePort(ctx, port, "tcp"); err != nil {
			return err
		}
		return n.openSinglePort(ctx, port, "udp")
	}
	return n.openSinglePort(ctx, port, protocol)
}

func (n *NFTablesManager) openSinglePort(ctx context.Context, port int, protocol string) error {
	cmd := exec.CommandContext(ctx, "nft", "add", "rule",
		NftTableType, NftTableName, NftChainName,
		protocol, "dport", fmt.Sprintf("%d", port),
		"accept", "comment", "\"prism-managed\"")

	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("開放端口失敗: %s", string(out))
	}

	n.openedPorts[port] = true
	return nil
}

func (n *NFTablesManager) OpenPortRange(ctx context.Context, start, end int, protocol string) error {
	n.mu.Lock()
	defer n.mu.Unlock()

	if err := n.ensureTableExists(ctx); err != nil {
		return err
	}

	if protocol == "both" {
		if err := n.openRange(ctx, start, end, "tcp"); err != nil {
			return err
		}
		return n.openRange(ctx, start, end, "udp")
	}
	return n.openRange(ctx, start, end, protocol)
}

func (n *NFTablesManager) openRange(ctx context.Context, start, end int, protocol string) error {
	rangeStr := fmt.Sprintf("%d-%d", start, end)
	cmd := exec.CommandContext(ctx, "nft", "add", "rule",
		NftTableType, NftTableName, NftChainName,
		protocol, "dport", rangeStr,
		"accept", "comment", "\"prism-managed-range\"")

	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("開放範圍失敗: %s", string(out))
	}
	return nil
}

// OpenHysteria2PortHopping 實現端口跳躍 (Filter Accept + NAT Redirect)
func (n *NFTablesManager) OpenHysteria2PortHopping(ctx context.Context, listenPort, start, end int) error {
	n.mu.Lock()
	defer n.mu.Unlock()

	n.log.Info("配置 NFTables 端口跳躍", zap.Int("listen", listenPort), zap.Int("start", start), zap.Int("end", end))

	// 1. Filter 表放行 (UDP)
	// Hysteria 2 使用 UDP，必須先允許流量進入
	if err := n.ensureTableExists(ctx); err != nil {
		return err
	}
	if err := n.openRange(ctx, start, end, "udp"); err != nil {
		return fmt.Errorf("Filter 表放行失敗: %w", err)
	}

	// 2. NAT 表轉發 (Redirect)
	if err := n.ensureNatTablesExists(ctx); err != nil {
		return err
	}

	rangeStr := fmt.Sprintf("%d-%d", start, end)
	toPort := fmt.Sprintf(":%d", listenPort)
	comment := fmt.Sprintf("\"prism-hy2-hop-%s\"", rangeStr)

	// 添加 IPv4 NAT 規則
	// nft add rule ip prism_nat_v4 prerouting udp dport 10000-20000 redirect to :443
	cmd := exec.CommandContext(ctx, "nft", "add", "rule",
		"ip", NftIPv4NatTableName, NftNatChainName,
		"udp", "dport", rangeStr,
		"redirect", "to", toPort,
		"comment", comment)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("IPv4 NAT 規則失敗: %s", string(out))
	}

	// 添加 IPv6 NAT 規則
	cmd = exec.CommandContext(ctx, "nft", "add", "rule",
		"ip6", NftIPv6NatTableName, NftNatChainName,
		"udp", "dport", rangeStr,
		"redirect", "to", toPort,
		"comment", comment)
	if out, err := cmd.CombinedOutput(); err != nil {
		n.log.Debug("IPv6 NAT 規則失敗 (可能忽略)", zap.String("err", string(out)))
	}

	return nil
}

func (n *NFTablesManager) GetOpenedPorts() []int {
	n.mu.Lock()
	defer n.mu.Unlock()
	ports := make([]int, 0, len(n.openedPorts))
	for k := range n.openedPorts {
		ports = append(ports, k)
	}
	sort.Ints(ports)
	return ports
}

// SaveRules 保存規則到文件
func (n *NFTablesManager) SaveRules(ctx context.Context) error {
	// 1. 導出當前規則集
	cmd := exec.CommandContext(ctx, "nft", "list", "ruleset")
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("導出規則失敗: %w", err)
	}

	configPath := "/etc/nftables.conf"

	// 2. 備份原文件 (如果存在)
	if _, err := os.Stat(configPath); err == nil {
		backupPath := configPath + ".bak"
		// 簡單的文件複製備份
		input, err := os.ReadFile(configPath)
		if err == nil {
			_ = os.WriteFile(backupPath, input, 0644)
		}
	}

	// 3. 寫入新規則
	if err := os.WriteFile(configPath, output, 0644); err != nil {
		return fmt.Errorf("寫入配置文件失敗: %w", err)
	}

	// 4. 嘗試啟用 nftables 服務（確保開機自啟）
	_ = exec.CommandContext(ctx, "systemctl", "enable", "nftables").Run()

	n.log.Info("✅ NFTables 規則已保存並備份原配置")
	return nil
}

// FlushRules 清空 Prism 相關的所有規則 (Filter + NAT)
func (n *NFTablesManager) FlushRules(ctx context.Context) error {
	n.mu.Lock()
	defer n.mu.Unlock()

	n.log.Info("🔥 正在清理 Prism 防火牆規則...")

	// 1. 刪除 Filter 表
	exec.CommandContext(ctx, "nft", "delete", "table", NftTableType, NftTableName).Run()

	// 2. 刪除 NAT 表 (IPv4 & IPv6)
	exec.CommandContext(ctx, "nft", "delete", "table", "ip", NftIPv4NatTableName).Run()
	exec.CommandContext(ctx, "nft", "delete", "table", "ip6", NftIPv6NatTableName).Run()

	n.openedPorts = make(map[int]bool)

	// 3. 重建基礎 Filter 表結構 (NAT 表按需創建)
	if err := n.ensureTableExists(ctx); err != nil {
		return fmt.Errorf("重建表失敗: %w", err)
	}

	n.log.Info("✅ Prism 規則已清理")
	return nil
}

// Backend 實現保持不變 ...
type nftablesBackend struct{}

func NewNftablesBackend() Backend {
	return &nftablesBackend{}
}

func (b *nftablesBackend) Name() string {
	return "nftables"
}

func (b *nftablesBackend) IsAvailable() bool {
	path, err := exec.LookPath("nft")
	return err == nil && path != ""
}

func (b *nftablesBackend) CreateManager(log *zap.Logger) Manager {
	return NewNFTables(log)
}
