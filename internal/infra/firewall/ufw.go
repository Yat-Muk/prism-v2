package firewall

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"sort"
	"strings"
	"sync"

	"go.uber.org/zap"

	"github.com/Yat-Muk/prism-v2/internal/pkg/errors"
)

type UFWManager struct {
	log         *zap.Logger
	openedPorts map[int]bool
	mu          sync.Mutex
}

func NewUFW(log *zap.Logger) *UFWManager {
	return &UFWManager{
		log:         log,
		openedPorts: make(map[int]bool),
	}
}

func (u *UFWManager) Type() string {
	return "ufw"
}

func (u *UFWManager) Capabilities() Capabilities {
	return Capabilities{
		SupportIPv6:        true,  // ufw 支持
		SupportPortHopping: false, // ufw 無法做 DNAT
		SupportComment:     true,
		SupportBoth:        false,
	}
}

func (u *UFWManager) OpenPort(ctx context.Context, port int, protocol string) error {
	u.mu.Lock()
	defer u.mu.Unlock()

	if protocol == "both" {
		if err := u.openSinglePort(ctx, port, "tcp"); err != nil {
			return err
		}
		return u.openSinglePort(ctx, port, "udp")
	}
	return u.openSinglePort(ctx, port, protocol)
}

func (u *UFWManager) openSinglePort(ctx context.Context, port int, protocol string) error {
	// 格式規範：ufw 規則通常為 "8080/tcp"
	portStr := fmt.Sprintf("%d/%s", port, strings.ToLower(protocol))
	comment := "prism"

	// 檢查規則是否已存在 (避免重複添加)
	if u.checkRuleExists(ctx, portStr) {
		u.openedPorts[port] = true
		return nil
	}

	// 添加規則: ufw allow 8080/tcp comment 'prism'
	cmd := exec.CommandContext(ctx, "ufw", "allow", portStr, "comment", comment)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("添加規則失敗: %s", string(output))
	}

	u.openedPorts[port] = true
	return nil
}

func (u *UFWManager) OpenPortRange(ctx context.Context, startPort, endPort int, protocol string) error {
	u.mu.Lock()
	defer u.mu.Unlock()

	rangeStr := fmt.Sprintf("%d:%d/%s", startPort, endPort, strings.ToLower(protocol))

	// ufw allow 1000:2000/tcp comment 'prism'
	cmd := exec.CommandContext(ctx, "ufw", "allow", rangeStr, "comment", "prism")
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("添加範圍規則失敗: %s", string(output))
	}

	return nil
}

func (u *UFWManager) OpenHysteria2PortHopping(ctx context.Context, listenPort, startPort, endPort int) error {
	u.log.Warn("⚠️ ufw 不支持端口跳躍功能 (NAT)",
		zap.Int("listen", listenPort),
		zap.Int("start", startPort),
		zap.Int("end", endPort),
		zap.String("建議", "請禁用 UFW 並使用 iptables 模式"))
	return errors.New("FIREWALL_UNSUPPORTED", "ufw 不支持端口跳躍 (DNAT)，請切換到 iptables 模式")
}

func (u *UFWManager) GetOpenedPorts() []int {
	u.mu.Lock()
	defer u.mu.Unlock()

	ports := make([]int, 0, len(u.openedPorts))
	for port := range u.openedPorts {
		ports = append(ports, port)
	}
	sort.Ints(ports)
	return ports
}

func (u *UFWManager) SaveRules(ctx context.Context) error {
	// UFW 自動持久化規則，無需手動 save
	u.log.Info("ufw 規則已自動生效")
	return nil
}

// FlushRules 清空規則 (穩健版)
func (u *UFWManager) FlushRules(ctx context.Context) error {
	u.mu.Lock()
	defer u.mu.Unlock()

	u.log.Info("開始清空 ufw 的 prism 規則")

	// 1. 獲取當前狀態
	cmd := exec.CommandContext(ctx, "ufw", "status")
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("獲取 ufw 狀態失敗: %w", err)
	}

	// 2. 解析並收集需要刪除的規則
	scanner := bufio.NewScanner(bytes.NewReader(output))
	var rulesToDelete []string

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.Contains(line, "# prism") {
			fields := strings.Fields(line)
			if len(fields) > 0 {
				target := fields[0]
				if target != "To" && target != "--" {
					rulesToDelete = append(rulesToDelete, target)
				}
			}
		}
	}

	// 3. 執行刪除
	successCount := 0
	for _, ruleTarget := range rulesToDelete {
		delCmd := exec.CommandContext(ctx, "ufw", "--force", "delete", "allow", ruleTarget)
		if out, err := delCmd.CombinedOutput(); err != nil {
			u.log.Debug("刪除 ufw 規則失敗",
				zap.String("target", ruleTarget),
				zap.String("output", string(out)))
		} else {
			successCount++
		}
	}

	u.openedPorts = make(map[int]bool)
	u.log.Info("ufw 規則清理統計", zap.Int("deleted", successCount))
	return nil
}

// 輔助函數：檢查規則是否存在
func (u *UFWManager) checkRuleExists(ctx context.Context, portStr string) bool {
	cmd := exec.CommandContext(ctx, "ufw", "status")
	output, err := cmd.Output()
	if err != nil {
		return false
	}
	return strings.Contains(string(output), portStr)
}

// Backend 實現
type ufwBackend struct{}

func NewUFWBackend() Backend {
	return &ufwBackend{}
}

func (b *ufwBackend) Name() string {
	return "ufw"
}

func (b *ufwBackend) IsAvailable() bool {
	if !commandExists("ufw") { // ✅ 直接使用同包下的定義
		return false
	}

	// 檢查 UFW 是否啟用
	cmd := exec.Command("ufw", "status")
	output, err := cmd.Output()
	if err != nil {
		return false
	}

	return strings.Contains(string(output), "Status: active")
}

func (b *ufwBackend) CreateManager(log *zap.Logger) Manager {
	return NewUFW(log)
}
