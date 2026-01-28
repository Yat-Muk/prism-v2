package system

import (
	"context"
	"fmt"
	"math"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/coreos/go-systemd/v22/dbus"
	"go.uber.org/zap"
)

// SystemdManager Systemd 服務管理器接口
type SystemdManager interface {
	Start(ctx context.Context, service string) error
	Stop(ctx context.Context, service string) error
	Restart(ctx context.Context, service string) error
	Reload(ctx context.Context, service string) error
	Enable(ctx context.Context, service string) error
	Disable(ctx context.Context, service string) error
	Status(ctx context.Context, service string) (*ServiceStatus, error)
	IsActive(ctx context.Context, service string) (bool, error)
	IsEnabled(ctx context.Context, service string) (bool, error)
	Close()
}

// ServiceStatus 服務詳細狀態
type ServiceStatus struct {
	Name        string
	Active      bool
	Running     bool
	Failed      bool
	Enabled     bool
	PID         string
	Memory      string
	MemoryBytes uint64
	Uptime      string
	UptimeDur   time.Duration
}

type dbusManager struct {
	conn *dbus.Conn
	log  *zap.Logger
	mu   sync.Mutex
}

func NewSystemdManager(log *zap.Logger) (SystemdManager, error) {
	conn, err := dbus.NewSystemdConnectionContext(context.Background())
	if err != nil {
		return nil, fmt.Errorf("無法連接 Systemd DBus: %w", err)
	}
	return &dbusManager{conn: conn, log: log}, nil
}

func (m *dbusManager) Close() {
	if m.conn != nil {
		m.conn.Close()
	}
}

// ensureSuffix 確保服務名以 .service 結尾
// 這是修復 "Unit name sing-box is not valid" 的關鍵
func ensureSuffix(service string) string {
	if !strings.HasSuffix(service, ".service") {
		return service + ".service"
	}
	return service
}

func (m *dbusManager) Start(ctx context.Context, service string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	service = ensureSuffix(service)

	ch := make(chan string, 1)
	_, err := m.conn.StartUnitContext(ctx, service, "replace", ch)
	if err != nil {
		return err
	}

	select {
	case result := <-ch:
		if result != "done" {
			return fmt.Errorf("啟動服務失敗: %s", result)
		}
		return nil
	case <-ctx.Done():
		return fmt.Errorf("啟動服務超時: %w", ctx.Err())
	}
}

func (m *dbusManager) Stop(ctx context.Context, service string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	service = ensureSuffix(service)

	ch := make(chan string)
	_, err := m.conn.StopUnitContext(ctx, service, "replace", ch)
	if err != nil {
		return err
	}
	result := <-ch
	if result != "done" {
		return fmt.Errorf("停止服務失敗: %s", result)
	}
	return nil
}

func (m *dbusManager) Restart(ctx context.Context, service string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	service = ensureSuffix(service)

	// 重啟前先重載配置，防止 Unit 文件變更未生效
	m.conn.ReloadContext(ctx)

	ch := make(chan string)
	_, err := m.conn.RestartUnitContext(ctx, service, "replace", ch)
	if err != nil {
		return err
	}
	result := <-ch
	if result != "done" {
		return fmt.Errorf("重啟服務失敗: %s", result)
	}
	return nil
}

func (m *dbusManager) Reload(ctx context.Context, service string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	service = ensureSuffix(service)

	ch := make(chan string)
	_, err := m.conn.ReloadUnitContext(ctx, service, "replace", ch)
	if err != nil {
		return err
	}
	result := <-ch
	if result != "done" {
		return fmt.Errorf("重載服務失敗: %s", result)
	}
	return nil
}

func (m *dbusManager) Enable(ctx context.Context, service string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	service = ensureSuffix(service)

	_, _, err := m.conn.EnableUnitFilesContext(ctx, []string{service}, false, true)
	return err
}

func (m *dbusManager) Disable(ctx context.Context, service string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	service = ensureSuffix(service)

	_, err := m.conn.DisableUnitFilesContext(ctx, []string{service}, false)
	return err
}

func (m *dbusManager) Status(ctx context.Context, service string) (*ServiceStatus, error) {
	service = ensureSuffix(service)

	// 獲取基本屬性
	units, err := m.conn.ListUnitsByNamesContext(ctx, []string{service})
	if err != nil {
		return nil, err
	}

	status := &ServiceStatus{Name: service}

	if len(units) > 0 {
		unit := units[0]
		status.Active = unit.ActiveState == "active"
		status.Running = unit.SubState == "running"
		status.Failed = unit.ActiveState == "failed"

		// 獲取詳細屬性 (PID, Memory 等)
		// 使用 GetAllPropertiesContext 比 GetUnitTypePropertiesContext 更穩健
		props, err := m.conn.GetAllPropertiesContext(ctx, service)
		if err == nil {
			if pid, ok := props["MainPID"].(uint32); ok && pid > 0 {
				status.PID = fmt.Sprintf("%d", pid)
			}
			if mem, ok := props["MemoryCurrent"].(uint64); ok && mem != math.MaxUint64 {
				status.MemoryBytes = mem
				// 假設 sysinfo.go 在同一包下提供了 formatBytes
				status.Memory = formatBytes(mem)
			}
			if ts, ok := props["ActiveEnterTimestamp"].(uint64); ok && ts > 0 {
				startTime := time.UnixMicro(int64(ts))
				if status.Active {
					status.UptimeDur = time.Since(startTime)
					// 假設 sysinfo.go 在同一包下提供了 formatDuration
					// 如果 sysinfo.go 未導出此函數，請將其複製到此文件中
					status.Uptime = status.UptimeDur.String()
				}
			}
		}
	}

	// 檢查是否啟用
	// 使用 systemctl fallback，因為 dbus 檢查 enable 狀態比較繁瑣且容易出錯
	cmd := exec.CommandContext(ctx, "systemctl", "is-enabled", service)
	output, _ := cmd.Output()
	fileState := strings.TrimSpace(string(output))
	status.Enabled = (fileState == "enabled" || fileState == "enabled-runtime")

	return status, nil
}

func (m *dbusManager) IsActive(ctx context.Context, service string) (bool, error) {
	service = ensureSuffix(service)
	units, err := m.conn.ListUnitsByNamesContext(ctx, []string{service})
	if err != nil {
		return false, err
	}
	if len(units) == 0 {
		return false, nil
	}
	return units[0].ActiveState == "active", nil
}

func (m *dbusManager) IsEnabled(ctx context.Context, service string) (bool, error) {
	service = ensureSuffix(service)
	// 簡單實現：使用 CLI
	cmd := exec.CommandContext(ctx, "systemctl", "is-enabled", service)
	err := cmd.Run()
	return err == nil, nil
}
