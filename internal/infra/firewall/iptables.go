package firewall

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"sort"
	"strconv"
	"strings"
	"sync"

	"go.uber.org/zap"
)

type IPTablesManager struct {
	log         *zap.Logger
	openedPorts map[int]bool
	mu          sync.Mutex
}

func NewIPTables(log *zap.Logger) *IPTablesManager {
	return &IPTablesManager{
		log:         log,
		openedPorts: make(map[int]bool),
	}
}

func (i *IPTablesManager) Type() string {
	return "iptables"
}

func (i *IPTablesManager) Capabilities() Capabilities {
	return Capabilities{
		SupportIPv6:        true, // iptables + ip6tables
		SupportPortHopping: true, // 通過 PREROUTING REDIRECT
		SupportComment:     true,
		SupportBoth:        false, // 需要分別調用 tcp/udp
	}
}

func (i *IPTablesManager) OpenPort(ctx context.Context, port int, protocol string) error {
	i.mu.Lock()
	defer i.mu.Unlock()

	if protocol == "both" {
		if err := i.openSinglePort(ctx, port, "tcp"); err != nil {
			return err
		}
		return i.openSinglePort(ctx, port, "udp")
	}
	return i.openSinglePort(ctx, port, protocol)
}

func (i *IPTablesManager) openSinglePort(ctx context.Context, port int, protocol string) error {
	portStr := strconv.Itoa(port)
	comment := fmt.Sprintf("prism-%s-%d", protocol, port)

	// 檢查規則是否已存在（IPv4）
	// 注意：這裡使用 -C (Check) 可以避免重複添加
	checkCmd := exec.CommandContext(ctx, "iptables", "-C", "INPUT", "-p", protocol,
		"--dport", portStr, "-j", "ACCEPT", "-m", "comment", "--comment", comment)
	if checkCmd.Run() == nil {
		i.openedPorts[port] = true
		return nil
	}

	// IPv4
	cmd := exec.CommandContext(ctx, "iptables", "-I", "INPUT", "-p", protocol,
		"--dport", portStr, "-j", "ACCEPT", "-m", "comment", "--comment", comment)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("IPv4 失敗: %s", string(output))
	}

	// IPv6
	cmd = exec.CommandContext(ctx, "ip6tables", "-I", "INPUT", "-p", protocol,
		"--dport", portStr, "-j", "ACCEPT", "-m", "comment", "--comment", comment)
	if output, err := cmd.CombinedOutput(); err != nil {
		i.log.Debug("IPv6 規則添加跳過（可能不支持）",
			zap.Error(err),
			zap.String("output", string(output)))
	}

	i.openedPorts[port] = true
	return nil
}

func (i *IPTablesManager) OpenPortRange(ctx context.Context, startPort, endPort int, protocol string) error {
	i.mu.Lock()
	defer i.mu.Unlock()

	portRange := fmt.Sprintf("%d:%d", startPort, endPort)
	comment := fmt.Sprintf("prism-range-%s-%d-%d", protocol, startPort, endPort)

	// IPv4
	cmd := exec.CommandContext(ctx, "iptables", "-I", "INPUT", "-p", protocol,
		"-m", "multiport", "--dports", portRange, "-j", "ACCEPT", "-m", "comment", "--comment", comment)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("IPv4 失敗: %s", string(output))
	}

	// IPv6
	cmd = exec.CommandContext(ctx, "ip6tables", "-I", "INPUT", "-p", protocol,
		"-m", "multiport", "--dports", portRange, "-j", "ACCEPT", "-m", "comment", "--comment", comment)
	if output, err := cmd.CombinedOutput(); err != nil {
		i.log.Debug("IPv6 規則添加跳過",
			zap.Error(err),
			zap.String("output", string(output)))
	}

	return nil
}

func (i *IPTablesManager) OpenHysteria2PortHopping(ctx context.Context, listenPort, startPort, endPort int) error {
	i.mu.Lock()
	defer i.mu.Unlock()

	portRange := fmt.Sprintf("%d:%d", startPort, endPort)
	comment := fmt.Sprintf("prism-hy2-hop-%d-%d", startPort, endPort)
	listenPortStr := strconv.Itoa(listenPort)

	i.log.Info("配置 Hysteria2 端口跳躍（iptables）",
		zap.Int("listen", listenPort),
		zap.String("range", portRange))

	// 1. Accept 規則（IPv4）
	cmd := exec.CommandContext(ctx, "iptables", "-I", "INPUT", "-p", "udp",
		"-m", "multiport", "--dports", portRange, "-j", "ACCEPT", "-m", "comment", "--comment", comment)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("IPv4 accept 失敗: %s", string(output))
	}

	// 2. REDIRECT 規則（IPv4）
	cmd = exec.CommandContext(ctx, "iptables", "-t", "nat", "-I", "PREROUTING",
		"-p", "udp", "-m", "multiport", "--dports", portRange,
		"-j", "REDIRECT", "--to-ports", listenPortStr,
		"-m", "comment", "--comment", comment+"-dnat")
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("IPv4 REDIRECT 失敗: %s", string(output))
	}

	// 3. IPv6
	cmd = exec.CommandContext(ctx, "ip6tables", "-I", "INPUT", "-p", "udp",
		"-m", "multiport", "--dports", portRange, "-j", "ACCEPT", "-m", "comment", "--comment", comment)
	if output, err := cmd.CombinedOutput(); err != nil {
		i.log.Debug("IPv6 Hopping Accept 跳過", zap.Error(err), zap.String("output", string(output)))
	}

	cmd = exec.CommandContext(ctx, "ip6tables", "-t", "nat", "-I", "PREROUTING",
		"-p", "udp", "-m", "multiport", "--dports", portRange,
		"-j", "REDIRECT", "--to-ports", listenPortStr,
		"-m", "comment", "--comment", comment+"-dnat")
	if output, err := cmd.CombinedOutput(); err != nil {
		i.log.Debug("IPv6 Hopping REDIRECT 跳過", zap.Error(err), zap.String("output", string(output)))
	}

	return nil
}

func (i *IPTablesManager) GetOpenedPorts() []int {
	i.mu.Lock()
	defer i.mu.Unlock()

	ports := make([]int, 0, len(i.openedPorts))
	for port := range i.openedPorts {
		ports = append(ports, port)
	}
	sort.Ints(ports)
	return ports
}

func (i *IPTablesManager) SaveRules(ctx context.Context) error {
	// 優先嘗試 netfilter-persistent (Debian/Ubuntu 標準)
	if commandExists("netfilter-persistent") {
		cmd := exec.CommandContext(ctx, "netfilter-persistent", "save")
		if err := cmd.Run(); err == nil {
			i.log.Info("規則已保存（netfilter-persistent）")
			return nil
		}
	}

	// 通用兜底方法：直接導出到文件
	// 注意：這裡使用 sh -c 是為了重定向，但比起複雜的管道，這是單一命令，相對安全
	cmd := exec.CommandContext(ctx, "sh", "-c", "iptables-save > /etc/iptables/rules.v4")
	if err := cmd.Run(); err != nil {
		i.log.Warn("保存 iptables 規則失敗", zap.Error(err))
	}

	cmd = exec.CommandContext(ctx, "sh", "-c", "ip6tables-save > /etc/iptables/rules.v6")
	if err := cmd.Run(); err != nil {
		i.log.Warn("保存 ip6tables 規則失敗", zap.Error(err))
	}

	return nil
}

// FlushRules 清空規則 (穩健版)
// 使用 iptables-save 獲取所有規則，在內存中過濾，然後生成刪除命令
func (i *IPTablesManager) FlushRules(ctx context.Context) error {
	i.mu.Lock()
	defer i.mu.Unlock()

	i.log.Info("開始清空 iptables 規則")

	// 1. 清理 IPv4
	if err := i.flushBinaryRules(ctx, "iptables"); err != nil {
		i.log.Error("清理 IPv4 規則時出錯", zap.Error(err))
	}

	// 2. 清理 IPv6
	if err := i.flushBinaryRules(ctx, "ip6tables"); err != nil {
		// IPv6 失敗通常不影響核心功能，降級為 Debug 日誌
		i.log.Debug("清理 IPv6 規則時出錯（可能未啟用 IPv6）", zap.Error(err))
	}

	i.openedPorts = make(map[int]bool)
	i.log.Info("iptables 規則清空完成")
	return nil
}

// flushBinaryRules 內部輔助函數：處理指定二進制（iptables 或 ip6tables）的清理
func (i *IPTablesManager) flushBinaryRules(ctx context.Context, binary string) error {
	// 1. 檢查命令是否存在
	if !commandExists(binary + "-save") {
		return fmt.Errorf("命令 %s-save 不存在", binary)
	}

	// 2. 獲取當前所有規則
	out, err := exec.CommandContext(ctx, binary+"-save").Output()
	if err != nil {
		return fmt.Errorf("執行 %s-save 失敗: %w", binary, err)
	}

	scanner := bufio.NewScanner(bytes.NewReader(out))
	var currentTable string
	var deleteCmds [][]string

	// 3. 解析輸出
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// 識別表名 (例如 *filter, *nat)
		if strings.HasPrefix(line, "*") {
			currentTable = line[1:] // 去掉 * 號
			continue
		}

		// 識別規則 (以 -A 開頭) 且 包含 prism 標記
		if strings.HasPrefix(line, "-A") && strings.Contains(line, "prism") {
			// 將 -A (Append) 替換為 -D (Delete)
			// line 示例: -A INPUT -p tcp --dport 80 ...
			args := strings.Fields(line)
			if len(args) > 0 {
				args[0] = "-D" // 修改動作為刪除
				// 構造完整命令參數: -t <table> -D <chain> ...
				fullArgs := append([]string{"-t", currentTable}, args...)
				deleteCmds = append(deleteCmds, fullArgs)
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("解析規則失敗: %w", err)
	}

	// 4. 執行刪除
	// 為了防止刪除順序問題（雖然刪除特定規則通常不依賴順序），我們按順序執行
	successCount := 0
	for _, args := range deleteCmds {
		// 使用 wait 參數確保每條命令執行完畢
		cmd := exec.CommandContext(ctx, binary, args...)
		if output, err := cmd.CombinedOutput(); err != nil {
			// 刪除失敗可能是規則已不存在，記錄為 Debug
			i.log.Debug("刪除規則失敗",
				zap.String("binary", binary),
				zap.Strings("args", args),
				zap.String("output", string(output)))
		} else {
			successCount++
		}
	}

	if len(deleteCmds) > 0 {
		i.log.Debug("規則清理統計",
			zap.String("binary", binary),
			zap.Int("found", len(deleteCmds)),
			zap.Int("deleted", successCount))
	}

	return nil
}

// Backend 實現
type iptablesBackend struct{}

func NewIptablesBackend() Backend {
	return &iptablesBackend{}
}

func (b *iptablesBackend) Name() string {
	return "iptables"
}

func (b *iptablesBackend) IsAvailable() bool {
	return commandExists("iptables")
}

func (b *iptablesBackend) CreateManager(log *zap.Logger) Manager {
	return NewIPTables(log)
}

// 輔助函數
func commandExists(cmd string) bool {
	_, err := exec.LookPath(cmd)
	return err == nil
}
