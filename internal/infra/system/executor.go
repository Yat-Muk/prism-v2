package system

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"go.uber.org/zap"

	"github.com/Yat-Muk/prism-v2/internal/pkg/errors"
)

// Executor 命令執行器接口
type Executor interface {
	// Execute 執行命令
	Execute(ctx context.Context, name string, args ...string) (string, error)

	// ExecuteWithTimeout 帶超時的命令執行
	ExecuteWithTimeout(ctx context.Context, timeout time.Duration, name string, args ...string) (string, error)

	// IsAllowed 檢查命令是否在白名單中
	IsAllowed(name string) bool
}

// SafeExecutor 安全的命令執行器
type SafeExecutor struct {
	allowlist map[string]bool
	logger    *zap.Logger
}

// NewExecutor 創建命令執行器
func NewExecutor(logger *zap.Logger) Executor {
	return &SafeExecutor{
		allowlist: map[string]bool{
			// --- 核心系統管理 ---
			"systemctl": true,
			"service":   true,
			"sysctl":    true,
			"uname":     true,
			"modprobe":  true,

			// --- 包管理器 (用於安裝內核/工具) ---
			"apt":     true, // Debian/Ubuntu
			"apt-get": true,
			"yum":     true, // CentOS/RHEL
			"dnf":     true,
			"dpkg":    true,
			"rpm":     true,

			// --- 文件與磁盤操作 ---
			"mkdir":     true,
			"chmod":     true,
			"chown":     true,
			"cp":        true,
			"mv":        true,
			"cat":       true,
			"tee":       true,
			"fallocate": true,
			"dd":        true,
			"mkswap":    true,
			"swapon":    true,
			"swapoff":   true,

			// --- 網絡工具 ---
			"ip":        true,
			"iptables":  true,
			"ip6tables": true,
			"nft":       true,
			"ufw":       true,
			"curl":      true,
			"wget":      true,
			"ping":      true,
			"ping6":     true,

			// --- 其他 ---
			"openssl":     true,
			"grep":        true,
			"awk":         true,
			"sed":         true,
			"cut":         true,
			"date":        true,
			"timedatectl": true,
		},
		logger: logger,
	}
}

// Execute 執行命令
func (e *SafeExecutor) Execute(ctx context.Context, name string, args ...string) (string, error) {
	// 檢查命令是否在白名單中
	if !e.IsAllowed(name) {
		return "", errors.New("SYS001", fmt.Sprintf("命令 %q 不在白名單中", name))
	}

	// 創建命令
	cmd := exec.CommandContext(ctx, name, args...)

	// 記錄日誌
	e.logger.Debug("執行命令",
		zap.String("cmd", name),
		zap.Strings("args", args),
	)

	// 執行並獲取輸出
	output, err := cmd.CombinedOutput()
	outputStr := strings.TrimSpace(string(output))

	if err != nil {
		e.logger.Error("命令執行失敗",
			zap.String("cmd", name),
			zap.Strings("args", args),
			zap.String("output", outputStr),
			zap.Error(err),
		)
		return outputStr, errors.Wrap(err, "SYS002", "命令執行失敗")
	}

	e.logger.Debug("命令執行成功",
		zap.String("cmd", name),
		zap.String("output", outputStr),
	)

	return outputStr, nil
}

// ExecuteWithTimeout 帶超時的命令執行
func (e *SafeExecutor) ExecuteWithTimeout(ctx context.Context, timeout time.Duration, name string, args ...string) (string, error) {
	// 創建帶超時的上下文
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	return e.Execute(ctx, name, args...)
}

// IsAllowed 檢查命令是否在白名單中
func (e *SafeExecutor) IsAllowed(name string) bool {
	return e.allowlist[name]
}
