package handlers

import (
	"bufio"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Yat-Muk/prism-v2/internal/application"
	domainConfig "github.com/Yat-Muk/prism-v2/internal/domain/config"
	"github.com/Yat-Muk/prism-v2/internal/domain/protocol"
	"github.com/Yat-Muk/prism-v2/internal/infra/backup"
	"github.com/Yat-Muk/prism-v2/internal/infra/firewall"
	infraSingbox "github.com/Yat-Muk/prism-v2/internal/infra/singbox"
	"github.com/Yat-Muk/prism-v2/internal/infra/system"
	"github.com/Yat-Muk/prism-v2/internal/pkg/appctx"
	"github.com/Yat-Muk/prism-v2/internal/pkg/clash"
	"github.com/Yat-Muk/prism-v2/internal/pkg/singbox"
	"github.com/Yat-Muk/prism-v2/internal/tui/msg"
	"github.com/Yat-Muk/prism-v2/internal/tui/state"
	"github.com/Yat-Muk/prism-v2/internal/tui/types"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

type CommandBuilder struct {
	log          *zap.Logger
	stateMgr     *state.Manager
	configSvc    *application.ConfigService
	portSvc      application.PortService
	protocolSvc  application.ProtocolService
	singboxSvc   *application.SingboxService
	certSvc      *application.CertService
	backupMgr    *backup.Manager
	sysInfo      *system.SystemInfo
	paths        *appctx.Paths
	executor     system.Executor
	firewallMgr  firewall.Manager
	protoFactory protocol.Factory
}

// NewCommandBuilder 構造函數
func NewCommandBuilder(
	log *zap.Logger,
	stateMgr *state.Manager,
	configSvc *application.ConfigService,
	portSvc application.PortService,
	protocolSvc application.ProtocolService,
	singboxSvc *application.SingboxService,
	certSvc *application.CertService,
	backupMgr *backup.Manager,
	sysInfo *system.SystemInfo,
	paths *appctx.Paths,
	executor system.Executor,
	firewallMgr firewall.Manager,
	protoFactory protocol.Factory,
) *CommandBuilder {
	return &CommandBuilder{
		log:          log,
		stateMgr:     stateMgr,
		configSvc:    configSvc,
		portSvc:      portSvc,
		protocolSvc:  protocolSvc,
		singboxSvc:   singboxSvc,
		certSvc:      certSvc,
		backupMgr:    backupMgr,
		sysInfo:      sysInfo,
		paths:        paths,
		executor:     executor,
		firewallMgr:  firewallMgr,
		protoFactory: protoFactory,
	}
}

// ========================================
// 基礎配置與數據更新
// ========================================

// LoadConfigCmd 加載配置
func (b *CommandBuilder) LoadConfigCmd(m *state.Manager) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if b.configSvc == nil {
			return msg.ConfigLoadedMsg{Err: fmt.Errorf("ConfigService 未初始化")}
		}

		cfg, err := b.configSvc.GetConfig(ctx)
		return msg.ConfigLoadedMsg{Config: cfg, Err: err}
	}
}

// LoadConfigSilentCmd 靜默加載配置
func (b *CommandBuilder) LoadConfigSilentCmd(m *state.Manager) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		cfg, err := b.configSvc.GetConfig(ctx)
		if err != nil {
			return msg.ConfigLoadedMsg{Err: err, Silent: true}
		}

		return msg.ConfigLoadedMsg{Config: cfg, Silent: true}
	}
}

// EnsureConfigJsonCmd 確保 config.json 存在
func (b *CommandBuilder) EnsureConfigJsonCmd(m *state.Manager) tea.Cmd {
	return func() tea.Msg {
		jsonPath := filepath.Join(b.paths.ConfigDir, "config.json")
		yamlPath := b.paths.ConfigFile

		if _, err := os.Stat(yamlPath); os.IsNotExist(err) {
			return nil
		}

		needRegenerate := false
		yamlInfo, _ := os.Stat(yamlPath)
		jsonInfo, err := os.Stat(jsonPath)

		if os.IsNotExist(err) || yamlInfo.ModTime().After(jsonInfo.ModTime()) {
			needRegenerate = true
		}

		if !needRegenerate {
			return nil
		}

		cfg := m.Config().GetConfig()
		if cfg == nil {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			var err error
			cfg, err = b.configSvc.GetConfig(ctx)
			if err != nil {
				return msg.ConfigUpdateMsg{Err: err}
			}
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := b.singboxSvc.ApplyConfig(ctx, cfg); err != nil {
			return msg.ConfigUpdateMsg{Err: err}
		}

		return msg.ConfigUpdateMsg{Message: "已自動同步配置文件", Applied: true}
	}
}

// UpdateDataCmd 更新系統數據 (核心修復點)
func (b *CommandBuilder) UpdateDataCmd(m *state.Manager) tea.Cmd {
	return func() tea.Msg {
		infraStats, _ := b.sysInfo.GetStats()
		svcStats, _ := b.sysInfo.GetServiceStats("sing-box")

		var uiStats types.SystemStats
		if infraStats != nil {
			uiStats = types.SystemStats{
				Hostname: infraStats.Hostname,
				OS:       infraStats.OS,
				Arch:     infraStats.Arch,
				Kernel:   infraStats.Kernel,
				Uptime:   formatDuration(infraStats.Uptime),
				LoadAvg:  infraStats.LoadAvg,
				CPUModel: infraStats.CPUModel,
				CPUUsage: infraStats.CPUUsage,

				// 內存轉換
				MemTotal: formatBytes(int64(infraStats.MemoryTotal)),
				MemUsed:  formatBytes(int64(infraStats.MemoryUsed)),
				MemUsage: infraStats.MemoryUsage,

				// 磁盤轉換
				DiskTotal: formatBytes(int64(infraStats.DiskTotal)),
				DiskUsed:  formatBytes(int64(infraStats.DiskUsed)),
				DiskUsage: infraStats.DiskUsage,

				// 網絡流量轉換
				NetSentTotal: formatBytes(int64(infraStats.NetSentTotal)),
				NetRecvTotal: formatBytes(int64(infraStats.NetRecvTotal)),

				// 格式化速率
				NetSentRate: fmt.Sprintf("%.2f MB/s", infraStats.NetworkTX),
				NetRecvRate: fmt.Sprintf("%.2f MB/s", infraStats.NetworkRX),

				// 原始速率 (浮點數)，用於某些 View 組件
				NetworkTX: infraStats.NetworkTX,
				NetworkRX: infraStats.NetworkRX,

				BBR:  infraStats.BBR,
				IPv4: infraStats.IPv4,
				IPv6: infraStats.IPv6,
			}
		}

		var uiSvcStats types.ServiceStats
		if svcStats != nil {
			uiSvcStats = types.ServiceStats{
				Status:      svcStats.Status,
				Uptime:      svcStats.Uptime,
				MemoryUsage: svcStats.Memory,
				CPUUsage:    0,
			}
		} else {
			uiSvcStats = types.ServiceStats{Status: "unknown"}
		}

		coreVer := "unknown"
		isInstalled := false
		if _, err := exec.LookPath("sing-box"); err == nil {
			isInstalled = true
			if out, err := exec.Command("sing-box", "version").Output(); err == nil {
				lines := strings.Split(string(out), "\n")
				if len(lines) > 0 {
					parts := strings.Fields(lines[0])
					if len(parts) >= 3 {
						coreVer = parts[2]
					}
				}
			}
		}

		return msg.DataUpdateMsg{
			Stats:        uiStats,
			ServiceStats: uiSvcStats,
			CoreVersion:  coreVer,
			IsInstalled:  isInstalled,
			HasUpdate:    false,
		}
	}
}

// ========================================
// 服務控制與安裝
// ========================================

// ServiceControlCmd 服務控制
func (b *CommandBuilder) ServiceControlCmd(action string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		var err error
		switch action {
		case "restart":
			err = b.singboxSvc.Restart(ctx)
		case "stop":
			err = b.singboxSvc.Stop(ctx)
		case "start":
			err = b.singboxSvc.Start(ctx)
		}

		return msg.ServiceResultMsg{Action: action, Err: err}
	}
}

// ToggleAutoStartCmd 切換開機自啓
func (b *CommandBuilder) ToggleAutoStartCmd(m *state.Manager) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// 檢查當前狀態
		cmd := exec.Command("systemctl", "is-enabled", "sing-box")
		output, err := cmd.Output()
		isEnabled := err == nil && strings.TrimSpace(string(output)) == "enabled"

		var newStatus bool
		if isEnabled {
			b.executor.Execute(ctx, "systemctl", "disable", "sing-box")
			newStatus = false
		} else {
			b.executor.Execute(ctx, "systemctl", "enable", "sing-box")
			newStatus = true
		}

		return msg.ServiceAutoStartMsg{
			Enabled: newStatus,
		}
	}
}

// ServiceHealthCheckCmd 服務健康檢查
func (b *CommandBuilder) ServiceHealthCheckCmd(m *state.Manager) tea.Cmd {
	return func() tea.Msg {
		b.log.Info("執行服務健康檢查")

		cfg := m.Config().GetConfig()
		svcStats, _ := b.sysInfo.GetServiceStats("sing-box")

		result := &types.HealthCheckResult{
			OverallStatus:   "error",
			Issues:          []string{},
			Recommendations: []string{},
		}

		if svcStats != nil && svcStats.Status == "running" {
			result.ServiceRunning = true
		} else {
			result.Issues = append(result.Issues, "服務未運行")
			result.Recommendations = append(result.Recommendations, "使用主菜單啓動服務")
		}

		if cfg != nil {
			result.ConfigValid = true
		} else {
			result.Issues = append(result.Issues, "配置文件無效")
		}

		if result.ServiceRunning && result.ConfigValid {
			result.OverallStatus = "healthy"
		}

		return msg.ServiceHealthMsg{Result: result}
	}
}

// InstallPrismCmd 安裝 Prism (首次配置生成)
func (b *CommandBuilder) InstallPrismCmd(m *state.Manager) tea.Cmd {
	return func() tea.Msg {
		protos := m.Install().InstallProtocols
		if len(protos) == 0 {
			return msg.InstallResultMsg{Err: fmt.Errorf("未選擇協議")}
		}

		// 原子化初始化配置
		err := b.configSvc.UpdateConfig(context.Background(), func(cfg *domainConfig.Config) error {
			if cfg.Version == 0 {
				*cfg = *domainConfig.DefaultConfig()
			}
			if err := b.protocolSvc.UpdateConfigWithEnabledProtocols(cfg, protos); err != nil {
				return fmt.Errorf("設置協議失敗: %w", err)
			}
			return nil
		})

		if err != nil {
			return msg.InstallResultMsg{Err: fmt.Errorf("保存配置失敗: %w", err)}
		}

		b.log.Info("✅ YAML 配置已初始化", zap.String("path", b.paths.ConfigFile))

		// 獲取最新配置快照
		cfg, _ := b.configSvc.GetConfig(context.Background())

		// 確保目錄結構
		os.MkdirAll(b.paths.ConfigDir, 0755)
		os.MkdirAll(b.paths.CertDir, 0700)

		// 生成自簽名證書
		certPath := filepath.Join(b.paths.CertDir, "self_signed.crt")
		keyPath := filepath.Join(b.paths.CertDir, "self_signed.key")

		b.log.Info("開始生成自簽名證書...")
		if err := b.certSvc.EnsureSelfSigned(certPath, keyPath, "www.bing.com"); err != nil {
			return msg.InstallResultMsg{Err: fmt.Errorf("生成自簽名證書失敗: %w", err)}
		}
		b.log.Info("✅ 自簽名證書已生成")

		// 生成並寫入 JSON 配置
		ctx := context.Background()
		if b.singboxSvc != nil {
			b.log.Info("開始生成 sing-box JSON 配置...")
			if err := b.singboxSvc.ApplyConfig(ctx, cfg); err != nil {
				return msg.InstallResultMsg{Err: fmt.Errorf("❌ JSON 配置生成失敗: %w", err)}
			}
			b.log.Info("✅ JSON 配置已生成")
		}

		// 設置開機自啟
		b.executor.Execute(ctx, "systemctl", "enable", "sing-box")

		// 啟動服務
		if err := b.singboxSvc.Restart(ctx); err != nil {
			return msg.InstallResultMsg{Err: fmt.Errorf("服務啟動失敗: %w\n配置已生成，請嘗試手動啟動", err)}
		}

		return msg.InstallResultMsg{Err: nil}
	}
}

// ========================================
// 配置管理
// ========================================

// SaveConfigCmd 保存配置 (原子更新)
func (b *CommandBuilder) SaveConfigCmd(m *state.Manager) tea.Cmd {
	return func() tea.Msg {
		uiConfig := m.Config().GetConfig()
		if uiConfig == nil {
			return msg.ConfigUpdateMsg{Err: fmt.Errorf("配置為空")}
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		// 原子寫入
		err := b.configSvc.UpdateConfig(ctx, func(diskCfg *domainConfig.Config) error {
			// 用 UI 狀態覆蓋磁盤狀態
			*diskCfg = *uiConfig
			return nil
		})

		if err != nil {
			return msg.ConfigUpdateMsg{Err: err}
		}

		// 應用配置
		if b.singboxSvc != nil {
			err = b.singboxSvc.ApplyConfig(ctx, uiConfig)
		}

		return msg.ConfigUpdateMsg{
			Err:       err,
			Message:   "配置已保存並應用",
			Applied:   true,
			NewConfig: uiConfig,
		}
	}
}

// ResetAllConfigCmd 重置配置
func (b *CommandBuilder) ResetAllConfigCmd() tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()

		if b.backupMgr != nil {
			configFile := b.paths.ConfigFile
			b.backupMgr.Backup(configFile, "yaml-before-reset")
		}

		newCfg := domainConfig.DefaultConfig()

		err := b.configSvc.UpdateConfig(ctx, func(cfg *domainConfig.Config) error {
			*cfg = *newCfg
			return nil
		})

		if err != nil {
			return msg.ConfigUpdateMsg{Err: fmt.Errorf("重置保存失敗: %w", err)}
		}

		if b.singboxSvc != nil {
			if err := b.singboxSvc.ApplyConfig(ctx, newCfg); err != nil {
				return msg.ConfigUpdateMsg{Err: fmt.Errorf("重置應用失敗: %w", err), NewConfig: newCfg}
			}
		}

		return msg.ConfigUpdateMsg{NewConfig: newCfg, Applied: true, Message: "配置已重置並應用"}
	}
}

// UpdateAnyTLSPaddingCmd 更新 Padding
func (b *CommandBuilder) UpdateAnyTLSPaddingCmd(m *state.Manager, profile int) tea.Cmd {
	return func() tea.Msg {
		cfg := m.Config().GetConfig()
		if cfg == nil {
			return msg.ConfigUpdateMsg{Err: fmt.Errorf("配置對象為空")}
		}

		var mode string
		switch profile {
		case 1:
			mode = "balanced"
		case 2:
			mode = "minimal"
		case 3:
			mode = "highresist"
		case 4:
			mode = "video"
		case 5:
			mode = "official"
		default:
			return msg.ConfigUpdateMsg{Err: fmt.Errorf("無效的配置 ID: %d", profile)}
		}

		cfg.Protocols.AnyTLS.PaddingMode = mode
		cfg.Protocols.AnyTLSReality.PaddingMode = mode

		return msg.ConfigUpdateMsg{
			NewConfig: cfg,
			Applied:   false,
			Message:   fmt.Sprintf("已切換到 %s 模式", mode),
		}
	}
}

// UpdateSinglePortCmd 僅更新內存配置
func (b *CommandBuilder) UpdateSinglePortCmd(m *state.Manager, protoID int, portInput string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		cfg := m.Config().GetConfig()

		newCfg, err := b.portSvc.UpdateSinglePort(ctx, cfg, protoID, portInput)
		if err != nil {
			return msg.ConfigUpdateMsg{Err: err}
		}

		return msg.ConfigUpdateMsg{NewConfig: newCfg, Applied: false}
	}
}

// ResetPortsCmd 僅重置內存配置
func (b *CommandBuilder) ResetPortsCmd(m *state.Manager) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		cfg := m.Config().GetConfig()

		newCfg, err := b.portSvc.ResetAllPorts(ctx, cfg)
		if err != nil {
			return msg.ConfigUpdateMsg{Err: err}
		}

		return msg.ConfigUpdateMsg{NewConfig: newCfg, Applied: false}
	}
}

// UpdateHy2HoppingCmd 僅更新跳躍設置
func (b *CommandBuilder) UpdateHy2HoppingCmd(m *state.Manager, start, end int) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		cfg := m.Config().GetConfig()

		newCfg, err := b.portSvc.UpdateHy2Hopping(ctx, cfg, start, end)
		if err != nil {
			return msg.ConfigUpdateMsg{Err: err}
		}

		return msg.ConfigUpdateMsg{
			NewConfig: newCfg,
			Applied:   false,
			Message:   fmt.Sprintf("已設置跳躍端口 %d-%d (未保存)", start, end),
		}
	}
}

// ClearHy2HoppingCmd 僅清除跳躍設置
func (b *CommandBuilder) ClearHy2HoppingCmd(m *state.Manager) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		cfg := m.Config().GetConfig()

		newCfg, err := b.portSvc.ClearHy2Hopping(ctx, cfg)
		if err != nil {
			return msg.ConfigUpdateMsg{Err: err}
		}

		return msg.ConfigUpdateMsg{NewConfig: newCfg, Applied: false}
	}
}

// SaveProtocolsOnlyCmd 僅保存協議開關狀態
func (b *CommandBuilder) SaveProtocolsOnlyCmd(m *state.Manager, enabled []int) tea.Cmd {
	return func() tea.Msg {
		cfg := m.Config().GetConfig()
		if cfg == nil {
			return msg.ConfigUpdateMsg{Err: fmt.Errorf("配置為空")}
		}
		newCfg := cfg.DeepCopy()

		if err := b.protocolSvc.UpdateConfigWithEnabledProtocols(newCfg, enabled); err != nil {
			return msg.ConfigUpdateMsg{Err: err}
		}

		return msg.ConfigUpdateMsg{NewConfig: newCfg, Applied: false}
	}
}

// ChangeSNICmd 修改 SNI
func (b *CommandBuilder) ChangeSNICmd(m *state.Manager, sni string) tea.Cmd {
	return func() tea.Msg {
		cfg := m.Config().GetConfig()
		if cfg == nil {
			return msg.ConfigUpdateMsg{Err: fmt.Errorf("配置對象為空")}
		}

		if err := b.protocolSvc.UpdateAllSNI(cfg, sni); err != nil {
			return msg.ConfigUpdateMsg{Err: err}
		}

		return msg.ConfigUpdateMsg{
			NewConfig: cfg,
			Applied:   false,
			Message:   fmt.Sprintf("SNI 已更新為 %s", sni),
		}
	}
}

// GenerateUUIDCmd 生成 UUID
func (b *CommandBuilder) GenerateUUIDCmd() tea.Cmd {
	return func() tea.Msg {
		newID := uuid.New().String()
		return msg.UUIDGeneratedMsg{UUID: newID}
	}
}

func (b *CommandBuilder) UpdateOutboundStrategyCmd(m *state.Manager, strategy string) tea.Cmd {
	return func() tea.Msg {
		cfg := m.Config().GetConfig()
		if cfg == nil {
			return msg.ConfigUpdateMsg{Err: fmt.Errorf("配置對象為空")}
		}
		cfg.Routing.DomainStrategy = strategy
		return msg.ConfigUpdateMsg{
			NewConfig: cfg,
			Applied:   false,
			Message:   fmt.Sprintf("已切換到 %s 策略", strategy),
		}
	}
}

// ========================================
// 備份管理
// ========================================

// ListBackupsCmd 列出備份
func (b *CommandBuilder) ListBackupsCmd(m *state.Manager, limit int) tea.Cmd {
	return func() tea.Msg {
		if b.backupMgr == nil {
			return msg.BackupListMsg{Err: fmt.Errorf("備份管理器未啟用")}
		}

		infraList, err := b.backupMgr.List()

		var uiList []types.BackupItem
		for _, f := range infraList {
			uiList = append(uiList, types.BackupItem{
				Name:      f.Name,
				Path:      f.Path,
				ModTime:   f.ModTime,
				Size:      f.Size,
				Encrypted: f.Encrypted,
			})
		}

		return msg.BackupListMsg{Entries: uiList, Err: err}
	}
}

// CreateBackupCmd 創建備份
func (b *CommandBuilder) CreateBackupCmd(m *state.Manager) tea.Cmd {
	return func() tea.Msg {
		if b.backupMgr == nil {
			return msg.BackupCreateMsg{Err: fmt.Errorf("備份管理器不可用")}
		}
		name := "manual"
		err := b.backupMgr.Backup(b.paths.ConfigFile, name)
		return msg.BackupCreateMsg{Name: name, Err: err}
	}
}

// RestoreBackupCmd 恢復備份
func (b *CommandBuilder) RestoreBackupCmd(m *state.Manager, index int) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		// 這裡為了簡化，從 TUI 狀態獲取名稱。更安全是重新 List
		uiList := m.Backup().BackupList
		if index < 0 || index >= len(uiList) {
			return msg.BackupRestoreMsg{Err: fmt.Errorf("無效索引")}
		}

		targetName := uiList[index].Name
		configFile := b.paths.ConfigFile

		if err := b.backupMgr.Restore(targetName, configFile); err != nil {
			return msg.BackupRestoreMsg{Err: err}
		}

		// 重載配置
		newCfg, err := b.configSvc.GetConfig(ctx)
		if err == nil {
			b.singboxSvc.ApplyConfig(ctx, newCfg)
		}

		return msg.ConfigUpdateMsg{NewConfig: newCfg, Applied: true, Message: "備份已恢復"}
	}
}

// DeleteBackupCmd 刪除備份
func (b *CommandBuilder) DeleteBackupCmd(m *state.Manager, backupName string) tea.Cmd {
	return func() tea.Msg {
		if b.backupMgr == nil {
			return msg.BackupListMsg{Err: fmt.Errorf("備份管理器不可用")}
		}

		path := filepath.Join(b.paths.BackupDir, backupName)
		os.Remove(path)
		os.Remove(path + ".sha256")

		// 重新獲取列表並轉換
		infraList, err := b.backupMgr.List()
		var uiList []types.BackupItem
		for _, f := range infraList {
			uiList = append(uiList, types.BackupItem{
				Name:      f.Name,
				Path:      f.Path,
				ModTime:   f.ModTime,
				Size:      f.Size,
				Encrypted: f.Encrypted,
			})
		}

		return msg.BackupListMsg{Entries: uiList, Err: err}
	}
}

// ========================================
// 證書管理
// ========================================

// CheckPort80Cmd 檢測 80 端口命令
func (b *CommandBuilder) CheckPort80Cmd() tea.Cmd {
	return func() tea.Msg {
		if b.certSvc == nil {
			return msg.CommandResultMsg{Success: false, Err: fmt.Errorf("未初始化")}
		}
		err := b.certSvc.CheckPort80Available()
		return msg.CommandResultMsg{Success: err == nil, Err: err, Data: "PORT80_CHECK"}
	}
}

func (b *CommandBuilder) RefreshCertListCmd() tea.Cmd {
	return func() tea.Msg {
		all, acmeDomains, selfDomains := b.certSvc.GetCertList()
		currentCA := b.certSvc.GetCurrentProvider()

		var uiList []types.CertInfo
		for _, c := range all {
			uiList = append(uiList, types.CertInfo{
				Domain:     c.Domain,
				Provider:   c.Provider,
				ExpireDate: c.ExpireDate,
				DaysLeft:   c.DaysLeft,
				Status:     c.Status,
			})
		}

		return msg.CertListRefreshMsg{
			ACMEDomains:       acmeDomains,
			SelfSignedDomains: selfDomains,
			CertList:          uiList,
			CurrentCAProvider: currentCA,
		}
	}
}

// RequestACMEDNSCmd 申請 ACME DNS 證書
func (b *CommandBuilder) RequestACMEDNSCmd(domain, provider, id, secret string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()

		params := map[string]string{
			"id":     id,
			"secret": secret,
		}

		err := b.certSvc.RequestACMEDNS(ctx, domain, provider, params)
		return msg.CertRequestMsg{Domain: domain, Err: err}
	}
}

// RequestACMEHTTPCmd 申請 ACME HTTP 證書
func (b *CommandBuilder) RequestACMEHTTPCmd(domain, email string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()

		err := b.certSvc.RequestACMEHTTP(ctx, domain, email)
		return msg.CertRequestMsg{Domain: domain, Err: err}
	}
}

// RenewCertCmd 續期證書
func (b *CommandBuilder) RenewCertCmd(domain string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()
		var err error
		if domain == "" || domain == "all" {
			err = b.certSvc.RenewAll(ctx)
		} else {
			err = b.certSvc.RenewCertificate(ctx, domain)
		}
		return msg.CertRenewMsg{Domain: domain, Err: err}
	}
}

// RenewAllCertsCmd 批量續期所有證書
func (b *CommandBuilder) RenewAllCertsCmd() tea.Cmd {
	return b.RenewCertCmd("all")
}

// DeleteCertCmd 刪除證書
func (b *CommandBuilder) DeleteCertCmd(domain string) tea.Cmd {
	return func() tea.Msg {
		err := b.certSvc.DeleteCert(domain)
		return msg.CertDeleteMsg{Domain: domain, Err: err}
	}
}

// SwitchProviderCmd 切換 ACME 提供商
func (b *CommandBuilder) SwitchProviderCmd(provider string) tea.Cmd {
	return func() tea.Msg {
		if b.certSvc == nil {
			return msg.ProviderSwitchMsg{Err: fmt.Errorf("證書服務未初始化")}
		}

		validProviders := map[string]bool{"letsencrypt": true, "zerossl": true}
		if !validProviders[provider] {
			return msg.ProviderSwitchMsg{Err: fmt.Errorf("無效的 CA 提供商: %s", provider)}
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		err := b.certSvc.SwitchProvider(ctx, provider)
		return msg.ProviderSwitchMsg{Provider: provider, Err: err}
	}
}

// ToggleCertModeCmd 切換協議的證書模式 (ACME <-> Self-Signed)
func (b *CommandBuilder) ToggleCertModeCmd(protocol string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if b.certSvc == nil {
			return msg.CommandResultMsg{Success: false, Message: "證書服務未初始化"}
		}

		resultMsg, err := b.certSvc.ToggleProtocolCertMode(ctx, protocol)
		if err != nil {
			return msg.CommandResultMsg{
				Success: false,
				Message: "切換失敗",
				Err:     err,
			}
		}

		return msg.CommandResultMsg{
			Success: true,
			Message: resultMsg,
			Data:    "CERT_MODE_SWITCH",
		}
	}
}

// ========================================
// 路由管理命令
// ========================================

// LoadWARPConfigCmd 加載 WARP 配置
func (b *CommandBuilder) LoadWARPConfigCmd(m *state.Manager) tea.Cmd {
	return func() tea.Msg {
		cfg := m.Config().GetConfig()
		if cfg == nil {
			return msg.CommandResultMsg{
				Success: false,
				Message: "配置未加載",
			}
		}

		return msg.RoutingConfigLoadedMsg{
			Type:   "warp",
			Config: cfg.Routing.WARP,
		}
	}
}

// ToggleWARPCmd 切換 WARP 啓用狀態
func (b *CommandBuilder) ToggleWARPCmd(m *state.Manager, ipType string) tea.Cmd {
	return func() tea.Msg {
		cfg := m.Config().GetConfig()
		if cfg == nil {
			return msg.ConfigUpdateMsg{
				Err: fmt.Errorf("配置未加載"),
			}
		}

		cfg.Routing.WARP.Enabled = !cfg.Routing.WARP.Enabled

		status := "禁用"
		if cfg.Routing.WARP.Enabled {
			status = "啓用"
		}

		return msg.ConfigUpdateMsg{
			NewConfig: cfg,
			Applied:   false,
			Message:   fmt.Sprintf("WARP %s 已%s", ipType, status),
		}
	}
}

// SetWARPGlobalCmd 設置 WARP 全局模式
func (b *CommandBuilder) SetWARPGlobalCmd(m *state.Manager, global bool) tea.Cmd {
	return func() tea.Msg {
		cfg := m.Config().GetConfig()
		if cfg == nil {
			return msg.ConfigUpdateMsg{
				Err: fmt.Errorf("配置未加載"),
			}
		}

		cfg.Routing.WARP.Global = global

		mode := "域名分流"
		if global {
			mode = "全局"
		}

		return msg.ConfigUpdateMsg{
			NewConfig: cfg,
			Applied:   false,
			Message:   fmt.Sprintf("WARP 已設置為%s模式", mode),
		}
	}
}

// ShowWARPConfigCmd 顯示 WARP 配置
func (b *CommandBuilder) ShowWARPConfigCmd(m *state.Manager) tea.Cmd {
	return func() tea.Msg {
		cfg := m.Config().GetConfig()
		if cfg == nil {
			return msg.CommandResultMsg{
				Success: false,
				Message: "配置未加載",
			}
		}

		warp := cfg.Routing.WARP
		status := "已禁用"
		if warp.Enabled {
			status = "已啓用"
		}

		mode := "域名分流"
		if warp.Global {
			mode = "全局模式"
		}

		domains := "無"
		if len(warp.Domains) > 0 {
			domains = strings.Join(warp.Domains, ", ")
		}

		message := fmt.Sprintf(
			"WARP 配置：\n狀態：%s\n模式：%s\n分流域名：%s",
			status, mode, domains,
		)

		return msg.CommandResultMsg{
			Success: true,
			Message: message,
		}
	}
}

// LoadSocks5ConfigCmd 加載 Socks5 配置
func (b *CommandBuilder) LoadSocks5ConfigCmd(m *state.Manager) tea.Cmd {
	return func() tea.Msg {
		cfg := m.Config().GetConfig()
		if cfg == nil {
			return msg.CommandResultMsg{
				Success: false,
				Message: "配置未加載",
			}
		}

		return msg.RoutingConfigLoadedMsg{
			Type:   "socks5",
			Config: cfg.Routing.Socks5,
		}
	}
}

// ToggleSocks5InboundCmd 切換 Socks5 入站
func (b *CommandBuilder) ToggleSocks5InboundCmd(m *state.Manager) tea.Cmd {
	return func() tea.Msg {
		cfg := m.Config().GetConfig()
		if cfg == nil {
			return msg.ConfigUpdateMsg{
				Err: fmt.Errorf("配置未加載"),
			}
		}

		cfg.Routing.Socks5.Inbound.Enabled = !cfg.Routing.Socks5.Inbound.Enabled

		status := "禁用"
		if cfg.Routing.Socks5.Inbound.Enabled {
			status = "啓用"
		}

		return msg.ConfigUpdateMsg{
			NewConfig: cfg,
			Applied:   false,
			Message:   fmt.Sprintf("Socks5 入站已%s", status),
		}
	}
}

// ToggleSocks5OutboundCmd 切換 Socks5 出站
func (b *CommandBuilder) ToggleSocks5OutboundCmd(m *state.Manager) tea.Cmd {
	return func() tea.Msg {
		cfg := m.Config().GetConfig()
		if cfg == nil {
			return msg.ConfigUpdateMsg{
				Err: fmt.Errorf("配置未加載"),
			}
		}

		cfg.Routing.Socks5.Outbound.Enabled = !cfg.Routing.Socks5.Outbound.Enabled

		status := "禁用"
		if cfg.Routing.Socks5.Outbound.Enabled {
			status = "啓用"
		}

		return msg.ConfigUpdateMsg{
			NewConfig: cfg,
			Applied:   false,
			Message:   fmt.Sprintf("Socks5 出站已%s", status),
		}
	}
}

// ShowSocks5ConfigCmd 顯示 Socks5 配置
func (b *CommandBuilder) ShowSocks5ConfigCmd(m *state.Manager) tea.Cmd {
	return func() tea.Msg {
		cfg := m.Config().GetConfig()
		if cfg == nil {
			return msg.CommandResultMsg{
				Success: false,
				Message: "配置未加載",
			}
		}

		inbound := cfg.Routing.Socks5.Inbound
		outbound := cfg.Routing.Socks5.Outbound

		inboundStatus := "已禁用"
		if inbound.Enabled {
			inboundStatus = fmt.Sprintf("已啓用 (端口: %d)", inbound.Port)
		}

		outboundStatus := "已禁用"
		if outbound.Enabled {
			outboundStatus = fmt.Sprintf("已啓用 (%s:%d)", outbound.Server, outbound.Port)
		}

		message := fmt.Sprintf(
			"Socks5 配置：\n入站：%s\n出站：%s",
			inboundStatus, outboundStatus,
		)

		return msg.CommandResultMsg{
			Success: true,
			Message: message,
		}
	}
}

// LoadIPv6ConfigCmd 加載 IPv6 配置
func (b *CommandBuilder) LoadIPv6ConfigCmd(m *state.Manager) tea.Cmd {
	return func() tea.Msg {
		cfg := m.Config().GetConfig()
		if cfg == nil {
			return msg.CommandResultMsg{
				Success: false,
				Message: "配置未加載",
			}
		}

		return msg.RoutingConfigLoadedMsg{
			Type:   "ipv6",
			Config: cfg.Routing.IPv6Split,
		}
	}
}

// ToggleIPv6RoutingCmd 切換 IPv6 分流
func (b *CommandBuilder) ToggleIPv6RoutingCmd(m *state.Manager, enable bool) tea.Cmd {
	return func() tea.Msg {
		cfg := m.Config().GetConfig()
		if cfg == nil {
			return msg.ConfigUpdateMsg{
				Err: fmt.Errorf("配置未加載"),
			}
		}

		cfg.Routing.IPv6Split.Enabled = enable

		status := "禁用"
		if enable {
			status = "啓用"
		}

		return msg.ConfigUpdateMsg{
			NewConfig: cfg,
			Applied:   false,
			Message:   fmt.Sprintf("IPv6 分流已%s", status),
		}
	}
}

// SetIPv6GlobalCmd 設置 IPv6 全局模式
func (b *CommandBuilder) SetIPv6GlobalCmd(m *state.Manager, global bool) tea.Cmd {
	return func() tea.Msg {
		cfg := m.Config().GetConfig()
		if cfg == nil {
			return msg.ConfigUpdateMsg{
				Err: fmt.Errorf("配置未加載"),
			}
		}

		cfg.Routing.IPv6Split.Global = global

		mode := "域名分流"
		if global {
			mode = "全局"
		}

		return msg.ConfigUpdateMsg{
			NewConfig: cfg,
			Applied:   false,
			Message:   fmt.Sprintf("IPv6 已設置為%s模式", mode),
		}
	}
}

// LoadDNSConfigCmd 加載 DNS 配置
func (b *CommandBuilder) LoadDNSConfigCmd(m *state.Manager) tea.Cmd {
	return func() tea.Msg {
		cfg := m.Config().GetConfig()
		if cfg == nil {
			return msg.CommandResultMsg{
				Success: false,
				Message: "配置未加載",
			}
		}

		return msg.RoutingConfigLoadedMsg{
			Type:   "dns",
			Config: cfg.Routing.DNS,
		}
	}
}

// DisableDNSRoutingCmd 禁用 DNS 分流
func (b *CommandBuilder) DisableDNSRoutingCmd(m *state.Manager) tea.Cmd {
	return func() tea.Msg {
		cfg := m.Config().GetConfig()
		if cfg == nil {
			return msg.ConfigUpdateMsg{
				Err: fmt.Errorf("配置未加載"),
			}
		}

		cfg.Routing.DNS.Enabled = false

		return msg.ConfigUpdateMsg{
			NewConfig: cfg,
			Applied:   false,
			Message:   "DNS 分流已禁用",
		}
	}
}

// ShowDNSRoutingConfigCmd 顯示 DNS 分流配置
func (b *CommandBuilder) ShowDNSRoutingConfigCmd(m *state.Manager) tea.Cmd {
	return func() tea.Msg {
		cfg := m.Config().GetConfig()
		if cfg == nil {
			return msg.CommandResultMsg{
				Success: false,
				Message: "配置未加載",
			}
		}

		dns := cfg.Routing.DNS
		status := "已禁用"
		if dns.Enabled {
			status = fmt.Sprintf("已啓用 (服務器: %s)", dns.Server)
		}

		domains := "無"
		if len(dns.DomainRules) > 0 {
			domains = strings.Join(dns.DomainRules, ", ")
		}

		message := fmt.Sprintf(
			"DNS 分流配置：\n狀態：%s\n分流域名：%s",
			status, domains,
		)

		return msg.CommandResultMsg{
			Success: true,
			Message: message,
		}
	}
}

// LoadSNIProxyConfigCmd 加載 SNI 代理配置
func (b *CommandBuilder) LoadSNIProxyConfigCmd(m *state.Manager) tea.Cmd {
	return func() tea.Msg {
		cfg := m.Config().GetConfig()
		if cfg == nil {
			return msg.CommandResultMsg{
				Success: false,
				Message: "配置未加載",
			}
		}

		return msg.RoutingConfigLoadedMsg{
			Type:   "sni_proxy",
			Config: cfg.Routing.SNIProxy,
		}
	}
}

// DisableSNIProxyCmd 禁用 SNI 代理
func (b *CommandBuilder) DisableSNIProxyCmd(m *state.Manager) tea.Cmd {
	return func() tea.Msg {
		cfg := m.Config().GetConfig()
		if cfg == nil {
			return msg.ConfigUpdateMsg{
				Err: fmt.Errorf("配置未加載"),
			}
		}

		cfg.Routing.SNIProxy.Enabled = false

		return msg.ConfigUpdateMsg{
			NewConfig: cfg,
			Applied:   false,
			Message:   "SNI 反向代理已禁用",
		}
	}
}

// ShowSNIProxyRulesCmd 顯示 SNI 代理規則
func (b *CommandBuilder) ShowSNIProxyRulesCmd(m *state.Manager) tea.Cmd {
	return func() tea.Msg {
		cfg := m.Config().GetConfig()
		if cfg == nil {
			return msg.CommandResultMsg{
				Success: false,
				Message: "配置未加載",
			}
		}

		sni := cfg.Routing.SNIProxy
		status := "已禁用"
		if sni.Enabled {
			status = fmt.Sprintf("已啓用 (目標: %s)", sni.TargetIP)
		}

		domains := "無"
		if len(sni.DomainRules) > 0 {
			domains = strings.Join(sni.DomainRules, ", ")
		}

		message := fmt.Sprintf(
			"SNI 反向代理配置：\n狀態：%s\n分流域名：%s",
			status, domains,
		)

		return msg.CommandResultMsg{
			Success: true,
			Message: message,
		}
	}
}

// UpdateSocks5InboundPortCmd 更新 Socks5 入站端口
func (b *CommandBuilder) UpdateSocks5InboundPortCmd(m *state.Manager, portStr string) tea.Cmd {
	return func() tea.Msg {
		cfg := m.Config().GetConfig()
		if cfg == nil {
			return msg.ConfigUpdateMsg{Err: fmt.Errorf("配置未加載")}
		}

		var port int
		fmt.Sscanf(portStr, "%d", &port)
		if port <= 0 || port > 65535 {
			return msg.ConfigUpdateMsg{Err: fmt.Errorf("無效端口: %s", portStr)}
		}

		cfg.Routing.Socks5.Inbound.Port = port
		return msg.ConfigUpdateMsg{NewConfig: cfg, Applied: false, Message: fmt.Sprintf("Socks5 入站端口已更新為 %d", port)}
	}
}

// UpdateSocks5InboundAuthCmd 更新 Socks5 入站認證
func (b *CommandBuilder) UpdateSocks5InboundAuthCmd(m *state.Manager, authStr string) tea.Cmd {
	return func() tea.Msg {
		cfg := m.Config().GetConfig()
		if cfg == nil {
			return msg.ConfigUpdateMsg{Err: fmt.Errorf("配置未加載")}
		}

		parts := strings.SplitN(authStr, ":", 2)
		if len(parts) == 2 {
			cfg.Routing.Socks5.Inbound.Username = parts[0]
			cfg.Routing.Socks5.Inbound.Password = parts[1]
			return msg.ConfigUpdateMsg{NewConfig: cfg, Applied: false, Message: "Socks5 入站認證已更新"}
		}
		cfg.Routing.Socks5.Inbound.Username = ""
		cfg.Routing.Socks5.Inbound.Password = ""
		return msg.ConfigUpdateMsg{NewConfig: cfg, Applied: false, Message: "Socks5 入站認證已清除"}
	}
}

// UpdateSocks5OutboundServerCmd 更新 Socks5 落地機
func (b *CommandBuilder) UpdateSocks5OutboundServerCmd(m *state.Manager, serverStr string) tea.Cmd {
	return func() tea.Msg {
		cfg := m.Config().GetConfig()
		if cfg == nil {
			return msg.ConfigUpdateMsg{Err: fmt.Errorf("配置未加載")}
		}

		parts := strings.Split(serverStr, ":")
		if len(parts) >= 2 {
			cfg.Routing.Socks5.Outbound.Server = parts[0]
			var port int
			fmt.Sscanf(parts[1], "%d", &port)
			cfg.Routing.Socks5.Outbound.Port = port
			return msg.ConfigUpdateMsg{NewConfig: cfg, Applied: false, Message: "Socks5 落地機地址已更新"}
		}
		return msg.ConfigUpdateMsg{Err: fmt.Errorf("格式錯誤，請使用 IP:Port")}
	}
}

// AddRoutingDomainCmd 通用添加域名命令
func (b *CommandBuilder) AddRoutingDomainCmd(m *state.Manager, routeType, domain string) tea.Cmd {
	return func() tea.Msg {
		cfg := m.Config().GetConfig()
		if cfg == nil {
			return msg.ConfigUpdateMsg{Err: fmt.Errorf("配置未加載")}
		}

		domain = strings.TrimSpace(domain)
		if domain == "" {
			return nil
		}

		var targetList *[]string
		var msgStr string

		switch routeType {
		case "ipv6":
			targetList = &cfg.Routing.IPv6Split.Domains
			msgStr = "IPv6"
		case "warp":
			targetList = &cfg.Routing.WARP.Domains
			msgStr = "WARP"
		case "dns":
			targetList = &cfg.Routing.DNS.DomainRules
			msgStr = "DNS"
		case "sni":
			targetList = &cfg.Routing.SNIProxy.DomainRules
			msgStr = "SNI"
		}

		if targetList != nil {
			for _, d := range *targetList {
				if d == domain {
					return msg.ConfigUpdateMsg{Err: fmt.Errorf("域名已存在")}
				}
			}
			*targetList = append(*targetList, domain)
			return msg.ConfigUpdateMsg{NewConfig: cfg, Applied: false, Message: fmt.Sprintf("已添加 %s 分流域名: %s", msgStr, domain)}
		}
		return nil
	}
}

// EnableRoutingWithTargetCmd 啓用帶目標 IP 的路由
func (b *CommandBuilder) EnableRoutingWithTargetCmd(m *state.Manager, routeType, target string) tea.Cmd {
	return func() tea.Msg {
		cfg := m.Config().GetConfig()
		if cfg == nil {
			return msg.ConfigUpdateMsg{Err: fmt.Errorf("配置未加載")}
		}

		switch routeType {
		case "dns":
			cfg.Routing.DNS.Enabled = true
			cfg.Routing.DNS.Server = target
		case "sni":
			cfg.Routing.SNIProxy.Enabled = true
			cfg.Routing.SNIProxy.TargetIP = target
		}
		return msg.ConfigUpdateMsg{NewConfig: cfg, Applied: false, Message: "路由規則已啓用"}
	}
}

// UpdateSocks5OutboundAuthCmd 更新 Socks5 出站認證
func (b *CommandBuilder) UpdateSocks5OutboundAuthCmd(m *state.Manager, authStr string) tea.Cmd {
	return func() tea.Msg {
		cfg := m.Config().GetConfig()
		if cfg == nil {
			return msg.ConfigUpdateMsg{Err: fmt.Errorf("配置未加載")}
		}

		parts := strings.SplitN(authStr, ":", 2)
		if len(parts) == 2 {
			cfg.Routing.Socks5.Outbound.Username = parts[0]
			cfg.Routing.Socks5.Outbound.Password = parts[1]
			return msg.ConfigUpdateMsg{NewConfig: cfg, Applied: false, Message: "Socks5 出站認證已更新"}
		}

		cfg.Routing.Socks5.Outbound.Username = ""
		cfg.Routing.Socks5.Outbound.Password = ""
		return msg.ConfigUpdateMsg{NewConfig: cfg, Applied: false, Message: "Socks5 出站認證已清除"}
	}
}

// SetSocks5GlobalCmd 設置 Socks5 全局轉發
func (b *CommandBuilder) SetSocks5GlobalCmd(m *state.Manager, global bool) tea.Cmd {
	return func() tea.Msg {
		cfg := m.Config().GetConfig()
		if cfg == nil {
			return msg.ConfigUpdateMsg{Err: fmt.Errorf("配置未加載")}
		}

		newState := !cfg.Routing.Socks5.Outbound.GlobalRoute
		cfg.Routing.Socks5.Outbound.GlobalRoute = newState

		status := "禁用"
		if newState {
			status = "啓用"
		}

		return msg.ConfigUpdateMsg{
			NewConfig: cfg,
			Applied:   false,
			Message:   fmt.Sprintf("Socks5 全局轉發已%s", status),
		}
	}
}

// SetWARPStateCmd
func (b *CommandBuilder) SetWARPStateCmd(m *state.Manager, enable bool) tea.Cmd {
	return func() tea.Msg {
		cfg := m.Config().GetConfig()
		if cfg == nil {
			return msg.ConfigUpdateMsg{Err: fmt.Errorf("配置未加載")}
		}

		cfg.Routing.WARP.Enabled = enable

		status := "已禁用"
		if enable {
			status = "已啓用"
		}

		return msg.ConfigUpdateMsg{
			NewConfig: cfg,
			Applied:   false,
			Message:   fmt.Sprintf("WARP %s", status),
		}
	}
}

// UpdateWARPLicenseCmd 更新 WARP 許可證密鑰
func (b *CommandBuilder) UpdateWARPLicenseCmd(m *state.Manager, license string) tea.Cmd {
	return func() tea.Msg {
		cfg := m.Config().GetConfig()
		if cfg == nil {
			return msg.ConfigUpdateMsg{Err: fmt.Errorf("配置未加載")}
		}

		license = strings.TrimSpace(license)
		cfg.Routing.WARP.LicenseKey = license

		msgText := "WARP 許可證已更新 (未保存)"
		if license == "" {
			msgText = "WARP 許可證已清除 (將使用免費版)"
		}

		return msg.ConfigUpdateMsg{
			NewConfig: cfg,
			Applied:   false,
			Message:   msgText,
		}
	}
}

// ========================================
// 核心管理命令
// ========================================

// CheckCoreUpdateCmd 檢查核心更新
func (b *CommandBuilder) CheckCoreUpdateCmd(m *state.Manager, silent bool) tea.Cmd {
	return func() tea.Msg {
		b.log.Info("檢查核心更新", zap.Bool("silent", silent))

		client := &http.Client{Timeout: 10 * time.Second}
		resp, err := client.Get("https://api.github.com/repos/SagerNet/sing-box/releases/latest")
		if err != nil {
			b.log.Error("檢查更新失敗", zap.Error(err))
			return msg.CoreCheckMsg{
				HasUpdate: false,
				IsSilent:  silent,
				Err:       err,
			}
		}
		defer resp.Body.Close()

		var result struct {
			TagName string `json:"tag_name"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			return msg.CoreCheckMsg{
				HasUpdate: false,
				IsSilent:  silent,
				Err:       fmt.Errorf("解析版本信息失敗: %w", err),
			}
		}

		latestVersion := strings.TrimPrefix(result.TagName, "v")
		currentVersion := m.Core().CoreVersion
		hasUpdate := latestVersion != currentVersion && currentVersion != "unknown"

		return msg.CoreCheckMsg{
			LatestVersion: latestVersion,
			HasUpdate:     hasUpdate,
			IsSilent:      silent,
		}
	}
}

// UpdateCoreCmd 更新核心
func (b *CommandBuilder) UpdateCoreCmd(m *state.Manager, version string) tea.Cmd {
	targetVersion := version
	if targetVersion == "" {
		targetVersion = m.Core().LatestVersion
	}
	if targetVersion == "" {
		targetVersion = "latest"
	}
	return b.InstallCoreCmdFull(targetVersion, m.Core().CoreVersion)
}

// downloadFile 下載文件輔助函數
func (b *CommandBuilder) downloadFile(url, destPath string) error {
	if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
		return fmt.Errorf("創建目錄失敗: %w", err)
	}
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("網絡請求失敗: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP 錯誤: %d", resp.StatusCode)
	}
	out, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("創建文件失敗: %w", err)
	}
	defer out.Close()
	_, err = io.Copy(out, resp.Body)
	return err
}

// InstallCoreCmdFull 執行核心安裝/更新任務
func (b *CommandBuilder) InstallCoreCmdFull(requestVersion string, currentInstalledVersion string) tea.Cmd {
	return func() tea.Msg {
		b.log.Info("開始安裝核心流程", zap.String("requested", requestVersion), zap.String("current", currentInstalledVersion))
		targetVersion := requestVersion

		if requestVersion == "latest" {
			b.log.Info("正在解析 sing-box 最新版本號...")
			client := &http.Client{Timeout: 10 * time.Second}
			resp, err := client.Get("https://api.github.com/repos/SagerNet/sing-box/releases/latest")
			if err != nil {
				return msg.CoreInstallMsg{Version: requestVersion, Success: false, Err: fmt.Errorf("無法獲取最新版本號: %w", err)}
			}
			defer resp.Body.Close()
			var result struct {
				TagName string `json:"tag_name"`
			}
			if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
				return msg.CoreInstallMsg{Version: requestVersion, Success: false, Err: fmt.Errorf("解析版本信息失敗: %w", err)}
			}
			targetVersion = strings.TrimPrefix(result.TagName, "v")
		}

		cleanCurrent := strings.TrimSpace(strings.TrimPrefix(currentInstalledVersion, "v"))
		cleanTarget := strings.TrimSpace(strings.TrimPrefix(targetVersion, "v"))

		if cleanCurrent != "" && cleanTarget == cleanCurrent {
			b.log.Info("版本一致，跳過更新", zap.String("version", cleanTarget))
			return msg.CoreInstallMsg{
				Version:   cleanTarget,
				Success:   true,
				Installed: true,
				Message:   fmt.Sprintf("當前已是最新版本 (%s)，無需更新", cleanTarget),
			}
		}

		coreInstaller := infraSingbox.NewInstaller(b.log, b.paths)
		if err := coreInstaller.InstallLatest(targetVersion); err != nil {
			return msg.CoreInstallMsg{Version: targetVersion, Success: false, Err: fmt.Errorf("核心下載失敗: %w", err)}
		}

		assetDir := b.paths.ConfigDir
		if assetDir == "" {
			assetDir = filepath.Dir(b.paths.CoreBinPath)
		}

		b.log.Info("正在下載資源文件 (GeoIP/GeoSite)...")
		geoipUrl := "https://raw.githubusercontent.com/SagerNet/sing-geoip/rule-set/geoip-cn.srs"
		geositeUrl := "https://raw.githubusercontent.com/SagerNet/sing-geosite/rule-set/geosite-cn.srs"
		adsUrl := "https://raw.githubusercontent.com/SagerNet/sing-geosite/rule-set/geosite-category-ads-all.srs"

		_ = b.downloadFile(geoipUrl, filepath.Join(assetDir, "geoip-cn.srs"))
		_ = b.downloadFile(geositeUrl, filepath.Join(assetDir, "geosite-cn.srs"))
		_ = b.downloadFile(adsUrl, filepath.Join(assetDir, "geosite-ads.srs"))

		configPath := filepath.Join(b.paths.ConfigDir, "config.json")
		svcInstaller := system.NewServiceInstaller(b.paths.CoreBinPath, configPath)
		if err := svcInstaller.Install(b.paths.SystemdServicePath); err != nil {
			return msg.CoreInstallMsg{Version: targetVersion, Success: false, Err: fmt.Errorf("服務註冊失敗: %w", err)}
		}

		return msg.CoreInstallMsg{
			Version:   targetVersion,
			Success:   true,
			Installed: true,
			Message:   fmt.Sprintf("核心已更新至 v%s 並重啟服務", targetVersion),
		}
	}
}

// UninstallCoreCmd 卸載核心
func (b *CommandBuilder) UninstallCoreCmd(m *state.Manager) tea.Cmd {
	return func() tea.Msg {
		b.log.Info("開始卸載核心...")
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		b.executor.Execute(ctx, "systemctl", "stop", "sing-box")
		b.executor.Execute(ctx, "systemctl", "disable", "sing-box")

		if err := os.Remove(b.paths.SystemdServicePath); err != nil && !os.IsNotExist(err) {
			b.log.Error("刪除服務文件失敗", zap.Error(err))
		}

		b.executor.Execute(ctx, "systemctl", "daemon-reload")

		if err := os.Remove(b.paths.CoreBinPath); err != nil {
			return msg.CommandResultMsg{Success: false, Message: "刪除核心文件失敗", Err: err}
		}

		m.Core().IsInstalled = false
		m.Core().CoreVersion = "unknown"

		return msg.CommandResultMsg{
			Success: true,
			Message: "sing-box 核心已成功卸載",
		}
	}
}

// ReinstallCoreCmd 重新安裝核心
func (b *CommandBuilder) ReinstallCoreCmd(m *state.Manager) tea.Cmd {
	currentVersion := m.Core().CoreVersion
	if currentVersion == "" || currentVersion == "unknown" {
		return func() tea.Msg {
			return msg.CommandResultMsg{Success: false, Message: "無法獲取當前版本，請嘗試手動安裝"}
		}
	}

	b.log.Info("觸發重新安裝", zap.String("version", currentVersion))
	return b.InstallCoreCmdFull(currentVersion, "")
}

// LoadCoreVersionsCmd 加載可用版本列表
func (b *CommandBuilder) LoadCoreVersionsCmd(m *state.Manager) tea.Cmd {
	return func() tea.Msg {
		b.log.Info("加載核心版本列表")

		client := &http.Client{Timeout: 10 * time.Second}
		resp, err := client.Get("https://api.github.com/repos/SagerNet/sing-box/releases?per_page=10")
		if err != nil {
			return msg.CoreVersionsMsg{Err: err}
		}
		defer resp.Body.Close()

		var releases []struct {
			TagName string `json:"tag_name"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&releases); err != nil {
			return msg.CoreVersionsMsg{Err: err}
		}

		var versions []string
		for _, rel := range releases {
			versions = append(versions, strings.TrimPrefix(rel.TagName, "v"))
		}

		return msg.CoreVersionsMsg{Versions: versions}
	}
}

// SetCoreSourceCmd 設置更新源
func (b *CommandBuilder) SetCoreSourceCmd(m *state.Manager, source string) tea.Cmd {
	return func() tea.Msg {
		b.log.Info("設置更新源", zap.String("source", source))

		m.Core().UpdateSource = source

		sourceName := map[string]string{
			"github":  "GitHub 官方源",
			"ghproxy": "鏡像加速源",
		}
		return msg.CommandResultMsg{
			Success: true,
			Message: fmt.Sprintf("更新源已切換到：%s", sourceName[source]),
		}
	}
}

// CheckScriptUpdateCmd 檢查腳本更新
func (b *CommandBuilder) CheckScriptUpdateCmd(m *state.Manager) tea.Cmd {
	return func() tea.Msg {
		repoURL := "https://api.github.com/repos/Yat-Muk/prism-v2/releases/latest"

		client := &http.Client{Timeout: 10 * time.Second}
		resp, err := client.Get(repoURL)
		if err != nil {
			return msg.ScriptCheckMsg{Success: false, Err: err}
		}
		defer resp.Body.Close()

		if resp.StatusCode != 200 {
			return msg.ScriptCheckMsg{Success: false, Err: fmt.Errorf("API 錯誤: %d", resp.StatusCode)}
		}

		var result struct {
			TagName string `json:"tag_name"`
			Body    string `json:"body"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			return msg.ScriptCheckMsg{Success: false, Err: err}
		}

		latestVer := strings.TrimPrefix(result.TagName, "v")

		return msg.ScriptCheckMsg{
			Success:   true,
			LatestVer: latestVer,
			Changelog: result.Body,
		}
	}
}

// UpdateScriptExecCmd 執行腳本自我更新
func (b *CommandBuilder) UpdateScriptExecCmd(m *state.Manager) tea.Cmd {
	return func() tea.Msg {
		arch := runtime.GOARCH
		osName := runtime.GOOS
		if osName != "linux" {
			return msg.CommandResultMsg{Success: false, Message: "僅支持 Linux 系統自動更新"}
		}

		fileName := fmt.Sprintf("prism_%s_%s", osName, arch)
		url := fmt.Sprintf("https://github.com/Yat-Muk/prism-v2/releases/latest/download/%s", fileName)

		exePath, err := os.Executable()
		if err != nil {
			return msg.CommandResultMsg{Success: false, Message: "無法獲取程序路徑", Err: err}
		}

		tmpPath := exePath + ".new"
		b.log.Info("正在下載新版本腳本...", zap.String("url", url))

		if err := b.downloadFile(url, tmpPath); err != nil {
			return msg.CommandResultMsg{Success: false, Message: "下載失敗", Err: err}
		}

		if err := os.Chmod(tmpPath, 0755); err != nil {
			return msg.CommandResultMsg{Success: false, Message: "權限設置失敗", Err: err}
		}

		if err := os.Rename(tmpPath, exePath); err != nil {
			return msg.CommandResultMsg{Success: false, Message: "替換文件失敗", Err: err}
		}

		b.log.Info("腳本更新成功，準備重啟")

		return msg.CommandResultMsg{
			Success: true,
			Message: "更新成功！請手動重啟程序生效 (輸入 prism)",
		}
	}
}

// ========================================
// 服務管理命令
// ========================================

// FollowServiceLogCmd 跟蹤服務日誌
func (b *CommandBuilder) FollowServiceLogCmd(m *state.Manager) tea.Cmd {
	return func() tea.Msg {
		logs := []string{"[INFO] sing-box service started", "[INFO] running..."}
		return msg.LogViewMsg{
			Mode:      "service",
			Logs:      logs,
			Following: true,
		}
	}
}

// ========================================
// 系統工具命令
// ========================================

// LoadSwapInfoCmd 獲取 Swap 狀態
func (b *CommandBuilder) LoadSwapInfoCmd(m *state.Manager) tea.Cmd {
	return func() tea.Msg {
		data, err := os.ReadFile("/proc/swaps")
		enabled := false
		swapFile := ""

		if err == nil {
			lines := strings.Split(string(data), "\n")
			for _, line := range lines {
				fields := strings.Fields(line)
				if len(fields) < 2 {
					continue
				}
				if fields[0] == "Filename" {
					continue
				}
				if fields[1] == "file" || fields[1] == "partition" {
					enabled = true
					swapFile = fields[0]
					break
				}
			}
		}

		cmd := exec.Command("sh", "-c", "LC_ALL=C free -h")
		out, err := cmd.Output()

		total, used, free := "未知", "0B", "0B"
		if err == nil {
			lines := strings.Split(string(out), "\n")
			for _, line := range lines {
				if strings.Contains(line, "Swap:") {
					fields := strings.Fields(line)
					if len(fields) >= 4 {
						total = fields[1]
						used = fields[2]
						free = fields[3]
					}
				}
			}
		}

		return msg.SwapInfoMsg{
			Info: &types.SwapInfo{
				Enabled:  enabled,
				Total:    total,
				Used:     used,
				Free:     free,
				SwapFile: swapFile,
			},
		}
	}
}

// CreateSwapCmd 創建 Swap
func (b *CommandBuilder) CreateSwapCmd(m *state.Manager, sizeGB int) tea.Cmd {
	return func() tea.Msg {
		b.log.Info("開始創建 SWAP", zap.Int("size_gb", sizeGB))

		if _, err := os.Stat("/swapfile"); err == nil {
			return msg.CommandResultMsg{Success: false, Message: "SWAP 文件已存在，請先刪除"}
		}

		cmds := []string{
			fmt.Sprintf("fallocate -l %dG /swapfile", sizeGB),
			"chmod 600 /swapfile",
			"mkswap /swapfile",
			"swapon /swapfile",
			"cp /etc/fstab /etc/fstab.bak",
			"echo '/swapfile none swap sw 0 0' | tee -a /etc/fstab",
		}

		for _, cmdStr := range cmds {
			cmd := exec.Command("sh", "-c", cmdStr)
			if err := cmd.Run(); err != nil {
				exec.Command("rm", "/swapfile").Run()
				return msg.CommandResultMsg{Success: false, Message: "創建失敗: " + err.Error()}
			}
		}

		return b.LoadSwapInfoCmd(m)()
	}
}

// DeleteSwapCmd 刪除 Swap
func (b *CommandBuilder) DeleteSwapCmd(currentPath string) tea.Cmd {
	return func() tea.Msg {
		target := "/swapfile"
		if currentPath != "" && strings.HasPrefix(currentPath, "/") {
			target = currentPath
		}

		b.log.Info("開始刪除 SWAP", zap.String("target", target))

		if err := exec.Command("swapoff", target).Run(); err != nil {
			b.log.Warn("swapoff 執行警告", zap.Error(err))
		}

		escapedTarget := strings.ReplaceAll(target, "/", "\\/")
		sedExpr := fmt.Sprintf("/%s/d", escapedTarget)
		exec.Command("sed", "-i", sedExpr, "/etc/fstab").Run()

		if err := os.Remove(target); err != nil && !os.IsNotExist(err) {
			return msg.CommandResultMsg{Success: false, Message: "刪除文件失敗: " + err.Error()}
		}

		return msg.CommandResultMsg{
			Success: true,
			Message: "SWAP 已刪除",
			Data:    "REFRESH_SWAP",
		}
	}
}

// ========  Fail2Ban  ========

const PrismFail2BanConfigPath = "/etc/fail2ban/jail.d/99-prism_sshd.conf"
const PrismJailName = "prism-sshd"

func formatSecondsToDuration(seconds int) string {
	if seconds >= 86400 {
		return fmt.Sprintf("%dd", seconds/86400)
	} else if seconds >= 3600 {
		return fmt.Sprintf("%dh", seconds/3600)
	} else if seconds >= 60 {
		return fmt.Sprintf("%dm", seconds/60)
	}
	return fmt.Sprintf("%ds", seconds)
}

func parseDurationToSeconds(input string) string {
	input = strings.ToLower(strings.TrimSpace(input))
	multiplier := 1
	numStr := input

	if strings.HasSuffix(input, "d") {
		multiplier = 86400
		numStr = strings.TrimSuffix(input, "d")
	} else if strings.HasSuffix(input, "h") {
		multiplier = 3600
		numStr = strings.TrimSuffix(input, "h")
	} else if strings.HasSuffix(input, "m") {
		multiplier = 60
		numStr = strings.TrimSuffix(input, "m")
	} else if strings.HasSuffix(input, "s") {
		numStr = strings.TrimSuffix(input, "s")
	}

	val, err := strconv.Atoi(numStr)
	if err != nil {
		return "3600"
	}
	return strconv.Itoa(val * multiplier)
}

func parseFail2BanInt(output, key string) int {
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if strings.Contains(line, key) {
			parts := strings.Split(line, key)
			if len(parts) > 1 {
				numStr := strings.TrimSpace(parts[1])
				var val int
				fmt.Sscanf(numStr, "%d", &val)
				return val
			}
		}
	}
	return 0
}

func detectSSHLogPath() string {
	if _, err := os.Stat("/var/log/secure"); err == nil {
		return "/var/log/secure"
	}
	return "/var/log/auth.log"
}

func readStaticFail2BanConfig() (int, string) {
	retry := 5
	banTime := "1h"

	content, err := os.ReadFile(PrismFail2BanConfigPath)
	if err != nil {
		return retry, banTime
	}

	lines := strings.Split(string(content), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "maxretry") {
			parts := strings.Split(line, "=")
			if len(parts) == 2 {
				if val, err := strconv.Atoi(strings.TrimSpace(parts[1])); err == nil {
					retry = val
				}
			}
		}
		if strings.HasPrefix(line, "bantime") {
			parts := strings.Split(line, "=")
			if len(parts) == 2 {
				valStr := strings.TrimSpace(parts[1])
				if secs, err := strconv.Atoi(valStr); err == nil {
					banTime = formatSecondsToDuration(secs)
				} else {
					banTime = valStr
				}
			}
		}
	}
	return retry, banTime
}

// LoadFail2BanInfoCmd 加載信息
func (b *CommandBuilder) LoadFail2BanInfoCmd(m *state.Manager) tea.Cmd {
	return func() tea.Msg {
		staticRetry, staticBanTime := readStaticFail2BanConfig()

		info := &types.Fail2BanInfo{
			Installed: false,
			Running:   false,
			MaxRetry:  staticRetry,
			BanTime:   staticBanTime,
		}

		path, err := exec.LookPath("fail2ban-server")
		if err == nil && path != "" {
			info.Installed = true
		} else {
			return msg.Fail2BanInfoMsg{Info: info}
		}

		out, _ := exec.Command("systemctl", "is-active", "fail2ban").Output()
		if strings.TrimSpace(string(out)) == "active" {
			info.Running = true
		}

		if info.Running {
			statusOut, err := exec.Command("fail2ban-client", "status", PrismJailName).Output()
			if err == nil {
				output := string(statusOut)
				info.BannedIPs = parseFail2BanInt(output, "Currently banned:")
				info.SSHAttempts = parseFail2BanInt(output, "Total failed:")
			}

			retryOut, err := exec.Command("fail2ban-client", "get", PrismJailName, "maxretry").Output()
			if err == nil {
				valStr := strings.TrimSpace(string(retryOut))
				if val, err := strconv.Atoi(valStr); err == nil {
					info.MaxRetry = val
				}
			}

			banOut, err := exec.Command("fail2ban-client", "get", PrismJailName, "bantime").Output()
			if err == nil {
				secondsStr := strings.TrimSpace(string(banOut))
				if seconds, err := strconv.Atoi(secondsStr); err == nil {
					info.BanTime = formatSecondsToDuration(seconds)
				}
			}
		} else {
			info.MaxRetry, info.BanTime = readStaticFail2BanConfig()
		}

		return msg.Fail2BanInfoMsg{Info: info}
	}
}

// UpdateFail2BanConfigCmd 更新配置
func (b *CommandBuilder) UpdateFail2BanConfigCmd(maxRetry, banTime string) tea.Cmd {
	return func() tea.Msg {
		b.log.Info("更新 Fail2Ban 配置", zap.String("retry", maxRetry), zap.String("time", banTime))

		logPath := detectSSHLogPath()
		banTimeSeconds := parseDurationToSeconds(banTime)

		configContent := fmt.Sprintf(`[sshd]
enabled = false

[%s]
enabled = true
filter = sshd
port    = ssh
logpath = %s
backend = auto
maxretry = %s
bantime  = %s
findtime = 1d
ignoreip = 127.0.0.1/8 ::1
`, PrismJailName, logPath, maxRetry, banTimeSeconds)

		os.MkdirAll("/etc/fail2ban/jail.d", 0755)

		if err := os.WriteFile(PrismFail2BanConfigPath, []byte(configContent), 0644); err != nil {
			return msg.CommandResultMsg{Success: false, Message: "寫入配置失敗: " + err.Error()}
		}

		os.Remove("/etc/fail2ban/jail.d/99-prism_sshd.local")
		os.Remove("/etc/fail2ban/jail.local")

		cmd := exec.Command("systemctl", "restart", "fail2ban")
		if out, err := cmd.CombinedOutput(); err != nil {
			return msg.CommandResultMsg{Success: false, Message: "重啟服務失敗: " + string(out)}
		}

		time.Sleep(2 * time.Second)

		return msg.CommandResultMsg{
			Success: true,
			Message: fmt.Sprintf("規則已更新 (Retry: %s, Time: %s)", maxRetry, banTime),
			Data:    "REFRESH_FAIL2BAN",
		}
	}
}

// InstallFail2BanCmd 安裝
func (b *CommandBuilder) InstallFail2BanCmd(m *state.Manager) tea.Cmd {
	return func() tea.Msg {
		b.log.Info("開始安裝 Fail2Ban")

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()

		var cmd *exec.Cmd

		if _, err := exec.LookPath("apt-get"); err == nil {
			updateCmd := exec.CommandContext(ctx, "apt-get", "update")
			updateCmd.Env = append(os.Environ(), "DEBIAN_FRONTEND=noninteractive")
			updateCmd.Run()
			cmd = exec.CommandContext(ctx, "apt-get", "install", "-y", "fail2ban")
			cmd.Env = append(os.Environ(), "DEBIAN_FRONTEND=noninteractive")
		} else if _, err := exec.LookPath("yum"); err == nil {
			exec.CommandContext(ctx, "yum", "install", "-y", "epel-release").Run()
			cmd = exec.CommandContext(ctx, "yum", "install", "-y", "fail2ban")
		} else {
			return msg.CommandResultMsg{Success: false, Message: "未找到支持的包管理器"}
		}

		if out, err := cmd.CombinedOutput(); err != nil {
			return msg.CommandResultMsg{Success: false, Message: "安裝失敗: " + string(out)}
		}

		os.MkdirAll("/etc/fail2ban/jail.d", 0755)
		if _, err := os.Stat(PrismFail2BanConfigPath); os.IsNotExist(err) {
			logPath := detectSSHLogPath()
			jailConfig := fmt.Sprintf(`[sshd]
enabled = false

[%s]
enabled = true
filter = sshd
port    = ssh
logpath = %s
backend = auto
maxretry = 5
bantime = 3600
findtime = 600
ignoreip = 127.0.0.1/8 ::1
`, PrismJailName, logPath)
			os.WriteFile(PrismFail2BanConfigPath, []byte(jailConfig), 0644)
		}

		b.executor.Execute(ctx, "systemctl", "enable", "fail2ban")
		if _, err := b.executor.Execute(ctx, "systemctl", "restart", "fail2ban"); err != nil {
			return msg.CommandResultMsg{Success: false, Message: "服務啟動失敗: " + err.Error()}
		}

		time.Sleep(2 * time.Second)

		return msg.CommandResultMsg{
			Success: true,
			Message: "Fail2Ban 安裝並啟動成功",
			Data:    "REFRESH_FAIL2BAN",
		}
	}
}

// UnbanIPCmd 解封 IP
func (b *CommandBuilder) UnbanIPCmd(ip string) tea.Cmd {
	return func() tea.Msg {
		b.log.Info("解封 IP", zap.String("ip", ip))

		cmd := exec.Command("fail2ban-client", "set", PrismJailName, "unbanip", ip)
		if out, err := cmd.CombinedOutput(); err != nil {
			return msg.CommandResultMsg{Success: false, Message: "解封失敗: " + string(out)}
		}

		return msg.CommandResultMsg{
			Success: true,
			Message: "IP " + ip + " 已解封",
			Data:    "REFRESH_FAIL2BAN_LIST",
		}
	}
}

// GetFail2BanListCmd 獲取封禁列表
func (b *CommandBuilder) GetFail2BanListCmd(m *state.Manager) tea.Cmd {
	return func() tea.Msg {
		cmd := exec.Command("fail2ban-client", "status", PrismJailName)
		out, err := cmd.CombinedOutput()
		if err != nil {
			return msg.LogViewMsg{Mode: "static", Logs: []string{
				fmt.Sprintf("獲取 %s 狀態失敗", PrismJailName),
				"請嘗試進入[配置規則]重新保存以應用新架構",
				"錯誤信息: " + err.Error(),
			}, Err: err}
		}

		output := string(out)
		var bannedIPs []string
		lines := strings.Split(output, "\n")
		for _, line := range lines {
			if strings.Contains(line, "Banned IP list:") {
				parts := strings.Split(line, ":")
				if len(parts) > 1 {
					ipStr := strings.TrimSpace(parts[1])
					if ipStr != "" {
						bannedIPs = strings.Fields(ipStr)
					}
				}
			}
		}

		var logs []string
		logs = append(logs, fmt.Sprintf("檢索時間: %s", time.Now().Format("2006-01-02 15:04:05")))
		logs = append(logs, fmt.Sprintf("監控對象: %s", PrismJailName))
		logs = append(logs, "")
		if len(bannedIPs) == 0 {
			logs = append(logs, "暫無被封禁的 IP")
		} else {
			for i, ip := range bannedIPs {
				logs = append(logs, fmt.Sprintf("[%02d] %s", i+1, ip))
			}
		}
		logs = append(logs, "")
		logs = append(logs, fmt.Sprintf("總計: %d 個", len(bannedIPs)))

		return msg.LogViewMsg{Mode: "static", Logs: logs}
	}
}

// ToggleFail2BanCmd
func (b *CommandBuilder) ToggleFail2BanCmd(running bool) tea.Cmd {
	return func() tea.Msg {
		action := "start"
		if running {
			action = "stop"
		}
		cmd := exec.Command("systemctl", action, "fail2ban")
		if err := cmd.Run(); err != nil {
			return msg.CommandResultMsg{Success: false, Message: fmt.Sprintf("執行 %s 失敗: %v", action, err)}
		}
		if action == "stop" {
			time.Sleep(500 * time.Millisecond)
		} else {
			time.Sleep(1 * time.Second)
		}
		return msg.CommandResultMsg{Success: true, Message: "服務狀態已更新", Data: "REFRESH_FAIL2BAN"}
	}
}

// UninstallFail2BanCmd 卸載
func (b *CommandBuilder) UninstallFail2BanCmd() tea.Cmd {
	return func() tea.Msg {
		exec.Command("systemctl", "stop", "fail2ban").Run()
		exec.Command("systemctl", "disable", "fail2ban").Run()

		if _, err := exec.LookPath("apt-get"); err == nil {
			exec.Command("apt-get", "remove", "--purge", "-y", "fail2ban").Run()
		} else {
			exec.Command("yum", "remove", "-y", "fail2ban").Run()
		}

		os.Remove("/etc/fail2ban/jail.local")
		os.Remove(PrismFail2BanConfigPath)
		return msg.CommandResultMsg{Success: true, Message: "Fail2Ban 已卸載", Data: "REFRESH_FAIL2BAN"}
	}
}

// ==== 時間校準 ====

// TimeSyncCmd 執行時間校準
func (b *CommandBuilder) TimeSyncCmd() tea.Cmd {
	return func() tea.Msg {
		b.log.Info("開始執行時間校準")
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		var err error
		var toolUsed string

		if _, e := exec.LookPath("chronyc"); e == nil {
			toolUsed = "chrony"
			_, err = b.executor.Execute(ctx, "chronyc", "-a", "makestep")
		} else if _, e := exec.LookPath("ntpdate"); e == nil {
			toolUsed = "ntpdate"
			_, err = b.executor.Execute(ctx, "ntpdate", "pool.ntp.org")
		} else if _, e := exec.LookPath("timedatectl"); e == nil {
			toolUsed = "systemd"
			b.executor.Execute(ctx, "timedatectl", "set-ntp", "false")
			_, err = b.executor.Execute(ctx, "timedatectl", "set-ntp", "true")
		} else {
			return msg.CommandResultMsg{
				Success: false,
				Message: "未找到時間同步工具 (需安裝 chrony 或 ntpdate)",
			}
		}

		if err != nil {
			return msg.CommandResultMsg{
				Success: false,
				Message: fmt.Sprintf("校準失敗 (%s): %v", toolUsed, err),
			}
		}

		return msg.CommandResultMsg{
			Success: true,
			Message: fmt.Sprintf("時間已通過 %s 校準成功", toolUsed),
		}
	}
}

// --- BBR ---

// LoadBBRInfoCmd 加載 BBR 信息
func (b *CommandBuilder) LoadBBRInfoCmd(m *state.Manager) tea.Cmd {
	return func() tea.Msg {
		kernelOut, _ := exec.Command("uname", "-r").CombinedOutput()
		kernelVer := strings.TrimSpace(string(kernelOut))

		algoOut, _ := exec.Command("sysctl", "-n", "net.ipv4.tcp_congestion_control").CombinedOutput()
		algo := strings.TrimSpace(string(algoOut))

		enabled := strings.Contains(algo, "bbr")
		bbrType := "未知"
		if enabled {
			if strings.Contains(kernelVer, "xanmod") {
				bbrType = "BBRv3 (XanMod)"
			} else if algo == "bbr2" {
				bbrType = "BBR2"
			} else {
				bbrType = "BBR (原版)"
			}
		}

		info := &types.BBRInfo{
			Enabled:       enabled,
			Type:          bbrType,
			KernelVersion: kernelVer,
			Algorithm:     algo,
		}

		return msg.BBRInfoMsg{Info: info}
	}
}

// EnableBBRCmd 啟用 BBR (原版或 bbr2)
func (b *CommandBuilder) EnableBBRCmd(algo string) tea.Cmd {
	return func() tea.Msg {
		b.log.Info("正在嘗試啟用 BBR", zap.String("algo", algo))

		out, err := exec.Command("sysctl", "-n", "net.ipv4.tcp_available_congestion_control").Output()
		if err != nil {
			return msg.CommandResultMsg{Success: false, Message: "無法讀取系統算法列表: " + err.Error()}
		}

		available := strings.Fields(string(out))
		isSupported := false
		for _, a := range available {
			if a == algo {
				isSupported = true
				break
			}
		}

		if !isSupported {
			if algo == "bbr2" {
				return msg.CommandResultMsg{
					Success: false,
					Message: "當前內核不支持 BBR2 算法",
					Data:    "BBR2_NOT_SUPPORTED",
				}
			}
			return msg.CommandResultMsg{
				Success: false,
				Message: fmt.Sprintf("錯誤：當前內核不支持 '%s' 算法 (可用: %v)", algo, available),
			}
		}

		conf := fmt.Sprintf(`
# Prism BBR Configuration
net.core.default_qdisc=fq
net.ipv4.tcp_congestion_control=%s
`, algo)

		f, err := os.OpenFile("/etc/sysctl.conf", os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
		if err != nil {
			return msg.CommandResultMsg{Success: false, Message: "無法打開配置文件: " + err.Error()}
		}
		if _, err := f.WriteString(conf); err != nil {
			f.Close()
			return msg.CommandResultMsg{Success: false, Message: "寫入配置文件失敗"}
		}
		f.Close()

		if out, err := exec.Command("sysctl", "-p").CombinedOutput(); err != nil {
			errMsg := strings.TrimSpace(string(out))
			return msg.CommandResultMsg{Success: false, Message: fmt.Sprintf("應用配置失敗: %s", errMsg)}
		}

		return msg.CommandResultMsg{Success: true, Message: fmt.Sprintf("成功啟用 %s 算法", algo), Data: "REFRESH_BBR"}
	}
}

// InstallBBR2KernelCmd 安裝 BBR2 專用內核
func (b *CommandBuilder) InstallBBR2KernelCmd() tea.Cmd {
	return func() tea.Msg {
		if runtime.GOARCH != "amd64" {
			return msg.CommandResultMsg{
				Success: false,
				Message: fmt.Sprintf("嚴重錯誤: BBR2 內核僅支持 x86_64 (amd64) 架構，當前為 %s", runtime.GOARCH),
			}
		}

		if _, err := exec.LookPath("apt-get"); err != nil {
			return msg.CommandResultMsg{
				Success: false,
				Message: "不支持的系統: 未檢測到 apt 包管理器 (僅支持 Debian/Ubuntu)",
			}
		}

		script := `
set -e
export DEBIAN_FRONTEND=noninteractive

cd /tmp
rm -f linux-image-*-bbr2*.deb linux-headers-*-bbr2*.deb

BASE_URL="https://github.com/ylx2016/kernel/releases/download/5.10.198-1bbr2"
IMG_NAME="linux-image-5.10.198-bbr2-1-987654_5.10.198-bbr2-1-987654_amd64.deb"
HDR_NAME="linux-headers-5.10.198-bbr2-1-987654_5.10.198-bbr2-1-987654_amd64.deb"

wget -q --show-progress -O "$IMG_NAME" "$BASE_URL/$IMG_NAME"
wget -q --show-progress -O "$HDR_NAME" "$BASE_URL/$HDR_NAME"

dpkg -i "$IMG_NAME" "$HDR_NAME"
apt-get -f install -y
update-grub
rm -f "$IMG_NAME" "$HDR_NAME"
`
		scriptPath := "/tmp/install_bbr2.sh"
		if err := os.WriteFile(scriptPath, []byte(script), 0755); err != nil {
			return msg.CommandResultMsg{Success: false, Message: "無法寫入安裝腳本: " + err.Error()}
		}

		cmd := exec.Command("bash", scriptPath)
		out, err := cmd.CombinedOutput()
		if err != nil {
			logs := string(out)
			if len(logs) > 500 {
				logs = "..." + logs[len(logs)-500:]
			}
			return msg.CommandResultMsg{
				Success: false,
				Message: fmt.Sprintf("安裝失敗 (Code %v). 日誌尾部:\n%s", err, logs),
			}
		}

		return msg.CommandResultMsg{
			Success: true,
			Message: "BBR2 內核安裝成功！請立即手動重啟系統 (命令: reboot) 以生效。",
			Data:    "REFRESH_BBR",
		}
	}
}

// DisableBBRCmd 禁用 BBR
func (b *CommandBuilder) DisableBBRCmd() tea.Cmd {
	return func() tea.Msg {
		exec.Command("sed", "-i", "/net.core.default_qdisc=fq/d", "/etc/sysctl.conf").Run()
		exec.Command("sed", "-i", "/net.ipv4.tcp_congestion_control/d", "/etc/sysctl.conf").Run()

		exec.Command("sysctl", "-w", "net.core.default_qdisc=pfifo_fast").Run()
		exec.Command("sysctl", "-w", "net.ipv4.tcp_congestion_control=cubic").Run()

		return msg.CommandResultMsg{Success: true, Message: "BBR 已禁用，恢復為 cubic", Data: "REFRESH_BBR"}
	}
}

// InstallXanModCmd 安裝 XanMod 內核
func (b *CommandBuilder) InstallXanModCmd() tea.Cmd {
	return func() tea.Msg {
		if runtime.GOARCH != "amd64" {
			return msg.CommandResultMsg{
				Success: false,
				Message: fmt.Sprintf("嚴重錯誤: XanMod 僅支持 x86_64 架構，檢測到當前為 %s，已終止操作以保護系統。", runtime.GOARCH),
			}
		}

		if _, err := exec.LookPath("apt-get"); err != nil {
			return msg.CommandResultMsg{Success: false, Message: "不支持的系統 (僅限 Debian/Ubuntu)"}
		}

		script := `
export DEBIAN_FRONTEND=noninteractive
wget -qO - https://dl.xanmod.org/gpg.key | gpg --dearmor -o /usr/share/keyrings/xanmod-archive-keyring.gpg --yes
echo 'deb [signed-by=/usr/share/keyrings/xanmod-archive-keyring.gpg] http://deb.xanmod.org releases main' | tee /etc/apt/sources.list.d/xanmod-release.list
apt-get update -y
apt-get install -y linux-xanmod-x64v3
`
		os.WriteFile("/tmp/install_xanmod.sh", []byte(script), 0755)

		cmd := exec.Command("bash", "/tmp/install_xanmod.sh")
		if out, err := cmd.CombinedOutput(); err != nil {
			errMsg := string(out)
			if len(errMsg) > 300 {
				errMsg = errMsg[len(errMsg)-300:]
			}
			return msg.CommandResultMsg{Success: false, Message: "安裝失敗 (日誌尾部): " + errMsg}
		}

		return msg.CommandResultMsg{
			Success: true,
			Message: "XanMod 內核安裝成功！請手動重啟系統 (reboot) 以生效。",
			Data:    "REFRESH_BBR",
		}
	}
}

// --- 系統清理 ---

// ScanSystemCleanupCmd 掃描清理
func (b *CommandBuilder) ScanSystemCleanupCmd() tea.Cmd {
	return func() tea.Msg {
		getDirSizeBytes := func(path string) int64 {
			size, _ := calculateDirSize(path)
			return size
		}

		var logSize, cacheSize, tempSize int64

		logSize += getDirSizeBytes("/var/log")
		if b.paths.LogDir != "" && !strings.HasPrefix(b.paths.LogDir, "/var/log") {
			logSize += getDirSizeBytes(b.paths.LogDir)
		}

		cachePaths := []string{
			"/var/cache/apt/archives",
			"/var/cache/yum",
			"/var/cache/dnf",
			"/var/cache/pacman/pkg",
		}
		for _, p := range cachePaths {
			if _, err := os.Stat(p); err == nil {
				cacheSize += getDirSizeBytes(p)
			}
		}

		tempSize += getDirSizeBytes("/tmp")

		total := logSize + cacheSize + tempSize

		return msg.CleanupInfoMsg{
			Info: &types.CleanupInfo{
				LogSize:   formatBytes(logSize),
				CacheSize: formatBytes(cacheSize),
				TempSize:  formatBytes(tempSize),
				TotalSize: formatBytes(total),
			},
		}
	}
}

// CleanSystemCmd 執行系統清理
func (b *CommandBuilder) CleanSystemCmd(m *state.Manager, target string) tea.Cmd {
	return func() tea.Msg {
		b.log.Info("執行系統清理", zap.String("target", target))

		cleanLogs := func() {
			exec.Command("sh", "-c", "find /var/log -type f -name '*.log' -exec truncate -s 0 {} \\;").Run()
			exec.Command("sh", "-c", "find /var/log -type f \\( -name '*.gz' -o -name '*.1' -o -name '*.old' -o -name '*.xz' \\) -delete").Run()
			if _, err := exec.LookPath("journalctl"); err == nil {
				exec.Command("journalctl", "--vacuum-time=1s").Run()
			}
			if b.paths.LogDir != "" {
				exec.Command("sh", "-c", fmt.Sprintf("truncate -s 0 %s/*.log", b.paths.LogDir)).Run()
			}
		}

		cleanPkg := func() {
			if _, err := exec.LookPath("apt-get"); err == nil {
				exec.Command("apt-get", "clean").Run()
				exec.Command("apt-get", "autoremove", "-y").Run()
			}
			if _, err := exec.LookPath("yum"); err == nil {
				exec.Command("yum", "clean", "all").Run()
			}
			if _, err := exec.LookPath("dnf"); err == nil {
				exec.Command("dnf", "clean", "all").Run()
			}
		}

		cleanTemp := func() {
			exec.Command("sh", "-c", "rm -rf /tmp/*").Run()
		}

		switch target {
		case "log":
			cleanLogs()
		case "pkg":
			cleanPkg()
		case "temp":
			cleanTemp()
		case "all":
			cleanLogs()
			cleanPkg()
			cleanTemp()
		}

		time.Sleep(500 * time.Millisecond)
		return b.ScanSystemCleanupCmd()()
	}
}

// --- 流媒體檢測 ---

// RunStreamingCheckCmd 執行流媒體檢測
func (b *CommandBuilder) CheckStreamingCmd(m *state.Manager, fullCheck bool) tea.Cmd {
	return func() tea.Msg {
		b.log.Info("開始執行深度流媒體檢測...")

		var wg sync.WaitGroup
		result := &types.StreamingCheckResult{}

		client := &http.Client{
			Timeout: 10 * time.Second,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
		}

		const ua = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"

		wg.Add(1)
		go func() {
			defer wg.Done()
			resp, err := client.Get("https://ipapi.co/json/")
			if err == nil {
				defer resp.Body.Close()
				var info struct {
					IP      string `json:"ip"`
					Country string `json:"country_name"`
					Org     string `json:"org"`
				}
				if json.NewDecoder(resp.Body).Decode(&info) == nil {
					result.IPv4 = info.IP
					result.Location = fmt.Sprintf("%s (%s)", info.Country, info.Org)
				}
			}
			if result.IPv4 == "" {
				result.IPv4 = "檢測失敗"
			}
		}()

		wg.Add(1)
		go func() {
			defer wg.Done()
			transportV6 := &http.Transport{
				DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
					return (&net.Dialer{Timeout: 5 * time.Second}).DialContext(ctx, "tcp6", addr)
				},
			}
			clientV6 := &http.Client{Transport: transportV6, Timeout: 5 * time.Second}

			apis := []string{"https://api64.ipify.org", "https://ipv6.icanhazip.com", "https://ifconfig.co/ip"}
			var ip string
			for _, api := range apis {
				if resp, err := clientV6.Get(api); err == nil {
					defer resp.Body.Close()
					if body, _ := io.ReadAll(resp.Body); len(body) > 0 {
						trimmed := strings.TrimSpace(string(body))
						if strings.Contains(trimmed, ":") {
							ip = trimmed
							break
						}
					}
				}
			}
			if ip != "" {
				result.IPv6 = ip
			} else {
				result.IPv6 = "未檢測到"
			}
		}()

		wg.Add(1)
		go func() {
			defer wg.Done()

			isAllowed := func(url string) bool {
				req, _ := http.NewRequest("GET", url, nil)
				req.Header.Set("User-Agent", ua)
				resp, err := client.Do(req)
				if err != nil {
					return false
				}
				defer resp.Body.Close()

				if resp.StatusCode == 200 {
					return true
				}

				if resp.StatusCode == 301 || resp.StatusCode == 302 {
					loc := resp.Header.Get("Location")
					if strings.Contains(loc, "geo-error") || strings.Contains(loc, "not-available") {
						return false
					}
					return true
				}
				return false
			}

			if isAllowed("https://www.netflix.com/title/70143836") {
				result.Netflix = "✓ 完整解鎖 (全區)"
				return
			}

			if isAllowed("https://www.netflix.com/title/80018499") {
				result.Netflix = "! 僅解鎖自製劇"
				return
			}

			result.Netflix = "✗ 封鎖 (無服務)"
		}()

		wg.Add(1)
		go func() {
			defer wg.Done()
			webStatus := "✗"
			appStatus := "✗"

			reqWeb, _ := http.NewRequest("GET", "https://chatgpt.com/", nil)
			reqWeb.Header.Set("User-Agent", ua)
			respWeb, err := client.Do(reqWeb)
			if err == nil {
				defer respWeb.Body.Close()
				if respWeb.StatusCode == 200 || respWeb.StatusCode == 302 {
					webStatus = "✓"
				} else if respWeb.StatusCode == 403 {
					webStatus = "?"
				}
			}

			reqApp, _ := http.NewRequest("GET", "https://ios.chat.openai.com/public-api/mobile/server_status/v1", nil)
			reqApp.Header.Set("User-Agent", ua)
			respApp, err := client.Do(reqApp)
			if err == nil {
				defer respApp.Body.Close()
				if respApp.StatusCode == 200 {
					appStatus = "✓"
				}
			}

			result.ChatGPT = fmt.Sprintf("Web: %s / App: %s", webStatus, appStatus)
		}()

		wg.Add(3)

		go func() {
			defer wg.Done()
			req, _ := http.NewRequest("GET", "https://www.youtube.com/premium", nil)
			req.Header.Set("User-Agent", ua)
			req.Header.Set("Accept-Language", "en-US")
			if resp, err := client.Do(req); err == nil {
				defer resp.Body.Close()
				if resp.StatusCode == 200 {
					result.YouTube = "✓ 完整解鎖"
				} else {
					result.YouTube = "! 僅限免費內容"
				}
			} else {
				result.YouTube = "✗ 連接失敗"
			}
		}()

		go func() {
			defer wg.Done()
			req, _ := http.NewRequest("GET", "https://www.disneyplus.com/", nil)
			req.Header.Set("User-Agent", ua)
			if resp, err := client.Do(req); err == nil {
				defer resp.Body.Close()
				if resp.StatusCode == 200 || resp.StatusCode == 301 || resp.StatusCode == 302 {
					result.Disney = "✓ 解鎖"
				} else if resp.StatusCode == 403 {
					result.Disney = "✗ 封鎖"
				} else {
					result.Disney = "! 未知狀態"
				}
			} else {
				result.Disney = "✗ 連接失敗"
			}
		}()

		go func() {
			defer wg.Done()
			req, _ := http.NewRequest("GET", "https://www.tiktok.com/", nil)
			req.Header.Set("User-Agent", ua)
			if resp, err := client.Do(req); err == nil {
				defer resp.Body.Close()
				if resp.StatusCode == 200 {
					result.TikTok = "✓ 解鎖"
				} else {
					result.TikTok = "✗ 封鎖"
				}
			} else {
				result.TikTok = "✗ 連接失敗"
			}
		}()

		wg.Wait()
		if result.Location == "" {
			result.Location = "未知"
		}
		return msg.StreamingCheckResultMsg{
			Result: result,
		}
	}
}

// RestartServiceCmd 重啟服務
func (b *CommandBuilder) RestartServiceCmd() tea.Cmd {
	return func() tea.Msg {
		if b.singboxSvc == nil {
			return msg.ServiceResultMsg{Action: "restart", Err: nil}
		}
		err := b.singboxSvc.Restart(context.Background())
		return msg.ServiceResultMsg{Action: "restart", Err: err}
	}
}

// StopServiceCmd 停止服務
func (b *CommandBuilder) StopServiceCmd() tea.Cmd {
	return func() tea.Msg {
		if b.singboxSvc == nil {
			return msg.ServiceResultMsg{Action: "stop", Err: nil}
		}
		err := b.singboxSvc.Stop(context.Background())
		return msg.ServiceResultMsg{Action: "stop", Err: err}
	}
}

// ========================================
// 日誌管理命令
// ========================================

// LoadLogInfoCmd 加載日誌信息
func (b *CommandBuilder) LoadLogInfoCmd(m *state.Manager) tea.Cmd {
	return func() tea.Msg {
		logPath := filepath.Join(b.paths.LogDir, "prism.log")

		logLevel := "info"
		if cfg := m.Config().GetConfig(); cfg != nil {
			logLevel = cfg.Log.Level
		}

		info, err := os.Stat(logPath)
		var size int64 = 0
		if err == nil {
			size = info.Size()
		}

		todayLines := 0
		errCount := 0

		if size < 10*1024*1024 {
			todayLines = countLogLines(logPath)
			errCount = countErrorLines(logPath)
		} else {
			todayLines = -1
			errCount = -1
		}

		previewLogs, _ := readLastNLines(logPath, 5)

		return msg.LogInfoMsg{
			LogLevel:   logLevel,
			LogPath:    logPath,
			LogSize:    formatBytes(size),
			TodayLines: todayLines,
			ErrorCount: errCount,
			RecentLogs: previewLogs,
		}
	}
}

// ViewRealtimeLogCmd 查看實時日誌
func (b *CommandBuilder) ViewRealtimeLogCmd(m *state.Manager) tea.Cmd {
	return func() tea.Msg {
		logPath := filepath.Join(b.paths.LogDir, "prism.log")
		logs, err := readLastNBytes(logPath, 20*1024)

		return msg.LogViewMsg{
			Mode: "realtime",
			Logs: logs,
			Err:  err,
		}
	}
}

// ViewFullLogCmd 查看完整日誌 (限制最大讀取量)
func (b *CommandBuilder) ViewFullLogCmd(m *state.Manager) tea.Cmd {
	return func() tea.Msg {
		logPath := filepath.Join(b.paths.LogDir, "prism.log")
		logs, err := readLastNBytes(logPath, 500*1024)
		if len(logs) > 0 {
			logs = append([]string{fmt.Sprintf("--- 僅顯示最後 500KB (總計 %d 行) ---", len(logs))}, logs...)
		}
		return msg.LogViewMsg{
			Mode: "full",
			Logs: logs,
			Err:  err,
		}
	}
}

// ViewErrorLogCmd 查看錯誤日誌
func (b *CommandBuilder) ViewErrorLogCmd(m *state.Manager) tea.Cmd {
	return func() tea.Msg {
		logPath := filepath.Join(b.paths.LogDir, "prism.log")
		logs, err := readLogFileWithFilter(logPath, []string{"ERROR", "FATAL"}, 1024*1024)
		return msg.LogViewMsg{
			Mode: "error",
			Logs: logs,
			Err:  err,
		}
	}
}

// --- 高性能輔助函數 ---

// readLastNLines 讀取文件最後 N 行
func readLastNLines(path string, n int) ([]string, error) {
	lines, err := readLastNBytes(path, int64(n*200))
	if err != nil {
		return nil, err
	}
	if len(lines) > n {
		return lines[len(lines)-n:], nil
	}
	return lines, nil
}

// readLastNBytes 核心函數：從文件末尾讀取 N 字節
func readLastNBytes(path string, n int64) ([]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		return nil, err
	}
	filesize := stat.Size()

	start := filesize - n
	if start < 0 {
		start = 0
		n = filesize
	}

	_, err = file.Seek(start, 0)
	if err != nil {
		return nil, err
	}

	buf := make([]byte, n)
	_, err = file.Read(buf)
	if err != nil && err != io.EOF {
		return nil, err
	}

	content := string(buf)
	lines := strings.Split(content, "\n")

	if start > 0 && len(lines) > 0 {
		lines = lines[1:]
	}

	var result []string
	for _, l := range lines {
		if strings.TrimSpace(l) != "" {
			result = append(result, l)
		}
	}

	return result, nil
}

// readLogFileWithFilter 優化版過濾
func readLogFileWithFilter(path string, keywords []string, maxBytes int64) ([]string, error) {
	rawLines, err := readLastNBytes(path, maxBytes)
	if err != nil {
		return nil, err
	}

	var filtered []string
	for _, line := range rawLines {
		for _, kw := range keywords {
			if strings.Contains(line, kw) {
				filtered = append(filtered, line)
				break
			}
		}
	}
	return filtered, nil
}

// ChangeLogLevelCmd 修改日誌級別
func (b *CommandBuilder) ChangeLogLevelCmd(m *state.Manager, newLevel string) tea.Cmd {
	return func() tea.Msg {
		uiConfig := m.Config().GetConfig()
		if uiConfig == nil {
			return msg.ConfigUpdateMsg{Err: fmt.Errorf("配置未加載")}
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		err := b.configSvc.UpdateConfig(ctx, func(cfg *domainConfig.Config) error {
			cfg.Log.Level = newLevel
			// 同步 UI 配置
			uiConfig.Log.Level = newLevel
			return nil
		})

		if err != nil {
			return msg.ConfigUpdateMsg{Err: err}
		}

		b.singboxSvc.Restart(ctx)
		return msg.ConfigUpdateMsg{
			Message:   fmt.Sprintf("日誌級別已更新為 %s", newLevel),
			Applied:   true,
			NewConfig: uiConfig,
		}
	}
}

// ClearLogCmd 清空日誌
func (b *CommandBuilder) ClearLogCmd(m *state.Manager) tea.Cmd {
	return func() tea.Msg {
		logPath := filepath.Join(b.paths.LogDir, "prism.log")
		os.Truncate(logPath, 0)
		return msg.CommandResultMsg{Success: true, Message: "日誌已清空"}
	}
}

// ExportLogCmd 導出日誌
func (b *CommandBuilder) ExportLogCmd(m *state.Manager) tea.Cmd {
	return func() tea.Msg {
		srcPath := filepath.Join(b.paths.LogDir, "prism.log")
		dstPath := filepath.Join(os.TempDir(), fmt.Sprintf("prism-log-%s.log", time.Now().Format("20060102")))

		src, err := os.Open(srcPath)
		if err != nil {
			return msg.CommandResultMsg{Success: false, Message: "無法讀取日誌"}
		}
		defer src.Close()

		dst, err := os.Create(dstPath)
		if err != nil {
			return msg.CommandResultMsg{Success: false, Message: "無法創建導出文件"}
		}
		defer dst.Close()

		io.Copy(dst, src)
		return msg.CommandResultMsg{Success: true, Message: "日誌已導出至 " + dstPath}
	}
}

// ========================================
// 節點信息命令
// ========================================

// GenerateProtocolLinksCmd 生成協議鏈接
func (b *CommandBuilder) GenerateProtocolLinksCmd(m *state.Manager) tea.Cmd {
	return func() tea.Msg {
		b.log.Info("正在生成協議鏈接...")

		cfg := m.Config().GetConfig()
		if cfg == nil {
			return msg.NodeInfoMsg{Err: fmt.Errorf("配置未加載，無法生成鏈接")}
		}

		var links []types.ProtocolLink

		serverIP := "YOUR_IP"
		if m.System() != nil && m.System().Stats != nil && m.System().Stats.IPv4 != "" {
			serverIP = m.System().Stats.IPv4
		} else {
			stats, _ := b.sysInfo.GetStats()
			if stats != nil && stats.IPv4 != "" {
				serverIP = stats.IPv4
			}
		}

		if cfg.Protocols.RealityVision.Enabled {
			p := cfg.Protocols.RealityVision
			rawLink := fmt.Sprintf("vless://%s@%s:%d?encryption=none&flow=xtls-rprx-vision&security=reality&sni=%s&fp=chrome&pbk=%s&sid=%s&type=tcp&headerType=none#Reality-Vision",
				cfg.UUID, serverIP, p.Port, p.SNI, p.PublicKey, p.ShortID)

			links = append(links, types.ProtocolLink{
				Name: "VLESS Reality Vision",
				URL:  rawLink,
				Port: p.Port,
			})
		}

		if cfg.Protocols.RealityGRPC.Enabled {
			p := cfg.Protocols.RealityGRPC
			serviceName := "grpc"
			rawLink := fmt.Sprintf("vless://%s@%s:%d?encryption=none&security=reality&sni=%s&fp=chrome&pbk=%s&sid=%s&type=grpc&serviceName=%s&mode=gun#Reality-gRPC",
				cfg.UUID, serverIP, p.Port, p.SNI, p.PublicKey, p.ShortID, serviceName)

			links = append(links, types.ProtocolLink{
				Name: "VLESS Reality gRPC",
				URL:  rawLink,
				Port: p.Port,
			})
		}

		if cfg.Protocols.Hysteria2.Enabled {
			p := cfg.Protocols.Hysteria2
			pass := cfg.Password

			rawLink := fmt.Sprintf("hysteria2://%s@%s:%d?sni=%s&insecure=1#Hysteria2",
				pass, serverIP, p.Port, p.SNI)

			links = append(links, types.ProtocolLink{
				Name: "Hysteria 2",
				URL:  rawLink,
				Port: p.Port,
			})
		}

		if cfg.Protocols.TUIC.Enabled {
			p := cfg.Protocols.TUIC
			rawLink := fmt.Sprintf("tuic://%s:%s@%s:%d?sni=%s&congestion_control=bbr&alpn=h3#TUIC-v5",
				cfg.UUID, cfg.Password, serverIP, p.Port, p.SNI)

			links = append(links, types.ProtocolLink{
				Name: "TUIC v5",
				URL:  rawLink,
				Port: p.Port,
			})
		}

		if cfg.Protocols.AnyTLS.Enabled {
			p := cfg.Protocols.AnyTLS
			rawLink := fmt.Sprintf("anytls://%s@%s:%d?sni=%s&idle_timeout=30s#AnyTLS",
				cfg.UUID, serverIP, p.Port, p.SNI)

			links = append(links, types.ProtocolLink{
				Name: "AnyTLS",
				URL:  rawLink,
				Port: p.Port,
			})
		}

		if cfg.Protocols.AnyTLSReality.Enabled {
			p := cfg.Protocols.AnyTLSReality
			rawLink := fmt.Sprintf("anytls://%s@%s:%d?security=reality&sni=%s&pbk=%s&sid=%s&idle_timeout=30s#AnyTLS-Reality",
				cfg.UUID, serverIP, p.Port, p.SNI, p.PublicKey, p.ShortID)

			links = append(links, types.ProtocolLink{
				Name: "AnyTLS Reality",
				URL:  rawLink,
				Port: p.Port,
			})
		}

		if cfg.Protocols.ShadowTLS.Enabled {
			p := cfg.Protocols.ShadowTLS

			method := "2022-blake3-aes-128-gcm"
			password := cfg.Password

			ssUserInfo := base64.URLEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", method, password)))

			stlsPass := cfg.Password
			pluginParam := fmt.Sprintf("host=%s;password=%s;version=3", p.SNI, stlsPass)

			rawLink := fmt.Sprintf("ss://%s@%s:%d?plugin=shadowtls%%3B%s#ShadowTLS-v3",
				ssUserInfo, serverIP, p.Port, pluginParam)

			links = append(links, types.ProtocolLink{
				Name: "ShadowTLS v3",
				URL:  rawLink,
				Port: p.Port,
			})
		}

		nodeInfo := &types.NodeInfo{
			ServerIP:  serverIP,
			Protocols: []string{},
		}
		for _, l := range links {
			nodeInfo.Protocols = append(nodeInfo.Protocols, l.Name)
		}

		return msg.NodeInfoMsg{
			Type:  "protocol_links",
			Links: links,
			Info:  nodeInfo,
		}
	}
}

// GenerateSubscriptionCmd 生成訂閱 (包含離線 Base64)
func (b *CommandBuilder) GenerateSubscriptionCmd(m *state.Manager) tea.Cmd {
	return func() tea.Msg {
		linkCmd := b.GenerateProtocolLinksCmd(m)
		linkMsg := linkCmd().(msg.NodeInfoMsg)

		if linkMsg.Err != nil {
			return msg.NodeInfoMsg{Err: linkMsg.Err}
		}

		var sb strings.Builder
		for _, link := range linkMsg.Links {
			sb.WriteString(link.URL + "\n")
		}
		rawContent := sb.String()

		base64Content := base64.StdEncoding.EncodeToString([]byte(rawContent))

		subInfo := &types.SubscriptionInfo{
			OnlineURL:  "(本機未部署訂閱後端，請使用離線訂閱)",
			OfflineURL: base64Content,
			UpdateTime: time.Now().Format("2006-01-02 15:04:05"),
			NodeCount:  len(linkMsg.Links),
		}

		return msg.NodeInfoMsg{
			Type:         "subscription",
			Subscription: subInfo,
		}
	}
}

// GenerateQRCodeCmd 生成二維碼 (支持容錯率)
func (b *CommandBuilder) GenerateQRCodeCmd(m *state.Manager, index int) tea.Cmd {
	return func() tea.Msg {
		links := m.Node().Links
		if len(links) == 0 || index < 0 || index >= len(links) {
			return msg.NodeInfoMsg{Err: fmt.Errorf("無效的鏈接索引")}
		}

		targetURL := links[index].URL

		level := m.Node().QRLevel
		if level == "" {
			level = "L"
		}

		ascii, err := generateASCIIQRCode(targetURL, level)
		if err != nil {
			return msg.NodeInfoMsg{Err: fmt.Errorf("生成二維碼失敗: %v", err)}
		}

		return msg.NodeInfoMsg{
			Type:    "qrcode",
			QRCode:  ascii,
			Content: targetURL,
			QRType:  "protocol",
		}
	}
}

func (b *CommandBuilder) copyToClipboard(content string) error {
	if content == "" {
		return fmt.Errorf("內容為空")
	}

	encoded := base64.StdEncoding.EncodeToString([]byte(content))
	osc52 := fmt.Sprintf("\x1b]52;c;%s\x07", encoded)
	fmt.Print(osc52)

	return nil
}

// BatchCopyLinksCmd 批量複製鏈接
func (b *CommandBuilder) BatchCopyLinksCmd(m *state.Manager, indices []int) tea.Cmd {
	return func() tea.Msg {
		var selected []string
		links := m.Node().Links

		if len(links) == 0 {
			return msg.NodeInfoMsg{Err: fmt.Errorf("沒有可複製的鏈接")}
		}

		for _, idx := range indices {
			if idx >= 0 && idx < len(links) {
				selected = append(selected, links[idx].URL)
			}
		}

		if len(selected) == 0 {
			return msg.NodeInfoMsg{Err: fmt.Errorf("未選擇有效鏈接")}
		}

		finalContent := strings.Join(selected, "\n")

		if err := b.copyToClipboard(finalContent); err != nil {
			return msg.NodeInfoMsg{Err: err}
		}

		return msg.CommandResultMsg{
			Success: true,
			Message: fmt.Sprintf("已複製節點鏈接 (%d 個)", len(selected)),
		}
	}
}

// GenerateSubscriptionQRCodeCmd 生成訂閱的二維碼
func (b *CommandBuilder) GenerateSubscriptionQRCodeCmd(m *state.Manager) tea.Cmd {
	return func() tea.Msg {
		sub := m.Node().Subscription
		if sub == nil || sub.OfflineURL == "" {
			return msg.NodeInfoMsg{Err: fmt.Errorf("請先生成訂閱信息")}
		}

		content := sub.OfflineURL
		if len(content) > 3000 {
			return msg.NodeInfoMsg{Err: fmt.Errorf("訂閱內容過長 (%d 字符)，\n 無法生成控制台二維碼，請直接複製 Base64", len(content))}
		}

		level := m.Node().QRLevel
		if level == "" {
			level = "L"
		}

		ascii, err := generateASCIIQRCode(content, level)
		if err != nil {
			return msg.NodeInfoMsg{Err: err}
		}

		return msg.NodeInfoMsg{
			Type:    "qrcode",
			QRCode:  ascii,
			Content: "Base64 Subscription Data",
			QRType:  "subscription",
		}
	}
}

func generateASCIIQRCode(content string, level string) (string, error) {
	if _, err := exec.LookPath("qrencode"); err != nil {
		return "", fmt.Errorf("未找到 qrencode 命令，\n 請安裝: apt install qrencode")
	}
	cmd := exec.Command("qrencode", "-t", "ANSIUTF8", "-l", level, "-o", "-", content)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", err
	}
	return string(out), nil
}

// CopySubscriptionOnlineCmd 複製在線訂閱鏈接
func (b *CommandBuilder) CopySubscriptionOnlineCmd(m *state.Manager) tea.Cmd {
	return func() tea.Msg {
		sub := m.Node().Subscription
		if sub == nil {
			return msg.NodeInfoMsg{Err: fmt.Errorf("訂閱數據未加載")}
		}

		if strings.Contains(sub.OnlineURL, "未部署") {
			return msg.NodeInfoMsg{Err: fmt.Errorf("無效的在線訂閱地址")}
		}

		if sub.OnlineURL == "" {
			return msg.NodeInfoMsg{Err: fmt.Errorf("在線訂閱地址為空")}
		}

		if err := b.copyToClipboard(sub.OnlineURL); err != nil {
			return msg.NodeInfoMsg{Err: err}
		}

		return msg.CommandResultMsg{
			Success: true,
			Message: "已複製訂閱鏈接 (在線訂閱)",
		}
	}
}

// CopySubscriptionOfflineCmd 複製離線訂閱內容
func (b *CommandBuilder) CopySubscriptionOfflineCmd(m *state.Manager) tea.Cmd {
	return func() tea.Msg {
		sub := m.Node().Subscription
		if sub == nil || sub.OfflineURL == "" {
			return msg.NodeInfoMsg{Err: fmt.Errorf("離線訂閱內容為空")}
		}

		if err := b.copyToClipboard(sub.OfflineURL); err != nil {
			return msg.NodeInfoMsg{Err: err}
		}

		return msg.CommandResultMsg{
			Success: true,
			Message: fmt.Sprintf("已複製離線訂閱鏈接 (離線內容 %d 字符)", len(sub.OfflineURL)),
		}
	}
}

// ExportClientConfigCmd 導出客戶端配置
func (b *CommandBuilder) ExportClientConfigCmd(m *state.Manager, target string, format string) tea.Cmd {
	return func() tea.Msg {
		cfg := m.Config().Config
		if cfg == nil {
			return msg.NodeInfoMsg{Err: fmt.Errorf("配置尚未加載，無法導出")}
		}

		baseHost := b.getBaseServerAddress(cfg)

		if baseHost == "" {
			baseHost = "<Server-IP>"
		}

		var (
			content  []byte
			filename string
			desc     string
			err      error
		)

		timestamp := time.Now().Format("20060102_150405")

		switch target {
		case "full", "sing-box":
			filename = fmt.Sprintf("prism_client_%s.json", timestamp)
			desc = "sing-box Client (v1.12+)"

			clientCfg := singbox.GenerateClientConfig(cfg, baseHost, b.protoFactory)

			content, err = json.MarshalIndent(clientCfg, "", "  ")
			if err != nil {
				return msg.NodeInfoMsg{Err: fmt.Errorf("序列化 JSON 失敗: %v", err)}
			}

		case "clash":
			filename = fmt.Sprintf("prism_client_clash_meta_%s.yaml", timestamp)
			desc = "Clash.Meta (Mihomo)"

			clashCfg := clash.GenerateClientConfig(cfg, baseHost, b.protoFactory)
			content, err = yaml.Marshal(clashCfg)
		default:
			return msg.NodeInfoMsg{Err: fmt.Errorf("未知目標: %s", target)}
		}

		filePath := filepath.Join("/tmp", filename)
		if err := os.WriteFile(filePath, content, 0644); err != nil {
			return msg.NodeInfoMsg{Err: fmt.Errorf("寫入失敗: %v", err)}
		}

		successInfo := fmt.Sprintf(
			"已導出 %s 配置！\n\n路徑: %s\n大小: %.2f KB\n\n下載命令:\nscp root@%s:%s",
			desc, filePath, float64(len(content))/1024.0, baseHost, filePath,
		)

		return msg.CommandResultMsg{Success: true, Message: successInfo}
	}
}

// GenerateNodeParamsCmd 生成詳細節點參數文本
func (b *CommandBuilder) GenerateNodeParamsCmd(m *state.Manager) tea.Cmd {
	return func() tea.Msg {
		cfg := m.Config().Config
		if cfg == nil {
			return msg.NodeParamsDataMsg{Err: fmt.Errorf("配置尚未加載")}
		}

		serverAddr := b.getBaseServerAddress(cfg)
		if serverAddr == "" {
			serverAddr = "<Server-IP>"
		}

		return msg.NodeParamsDataMsg{
			ServerIP: serverAddr,
		}
	}
}

// ==========================================
// 卸載相關命令
// ==========================================

// ScanUninstallInfoCmd 掃描卸載信息
func (b *CommandBuilder) ScanUninstallInfoCmd(m *state.Manager) tea.Cmd {
	return func() tea.Msg {
		u := m.Uninstall()

		info := &types.UninstallInfo{
			ConfirmStep: u.ConfirmStep,
			KeepConfig:  u.KeepConfig,
			KeepCerts:   u.KeepCerts,
			KeepBackups: u.KeepBackups,
			KeepLogs:    u.KeepLogs,

			ConfigPath: b.paths.ConfigFile,
			CertDir:    b.paths.CertDir,
			BackupDir:  b.paths.BackupDir,
			LogDir:     b.paths.LogDir,
		}

		if binPath, err := exec.LookPath("sing-box"); err == nil {
			info.CoreInstalled = true
			info.CorePath = binPath
		}

		if b.sysInfo != nil {
			svcStats, _ := b.sysInfo.GetServiceStats("sing-box")
			if svcStats != nil && svcStats.Status == "running" {
				info.ServiceRunning = true
			}
		}

		if _, err := os.Stat(info.ConfigPath); err == nil {
			info.ConfigExists = true
		}

		if entries, err := os.ReadDir(info.CertDir); err == nil {
			info.CertsCount = len(entries)
		}
		if entries, err := os.ReadDir(info.BackupDir); err == nil {
			info.BackupsCount = len(entries)
		}

		logFile := filepath.Join(info.LogDir, "prism.log")
		if logInfo, err := os.Stat(logFile); err == nil {
			info.LogSize = formatBytes(logInfo.Size())
		} else {
			info.LogSize = "0 B"
		}

		var totalSize int64

		if info.CoreInstalled {
			if binInfo, err := os.Stat(info.CorePath); err == nil {
				totalSize += binInfo.Size()
			}
		}

		if s, _ := calculateDirSize(b.paths.ConfigDir); s > 0 {
			totalSize += s
		}
		if s, _ := calculateDirSize(b.paths.CertDir); s > 0 {
			totalSize += s
		}
		if s, _ := calculateDirSize(b.paths.BackupDir); s > 0 {
			totalSize += s
		}
		if s, _ := calculateDirSize(b.paths.LogDir); s > 0 {
			totalSize += s
		}

		info.TotalSize = formatBytes(totalSize)

		return msg.UninstallInfoMsg{Info: info}
	}
}

// UninstallPrismCmd 執行卸載
func (b *CommandBuilder) UninstallPrismCmd(m *state.Manager) tea.Cmd {
	uState := m.Uninstall()
	keepConfig := uState.KeepConfig
	keepCerts := uState.KeepCerts
	keepBackups := uState.KeepBackups
	keepLogs := uState.KeepLogs

	baseDir := b.paths.BaseDir
	logDir := b.paths.LogDir

	certDirName := filepath.Base(b.paths.CertDir)
	backupDirName := filepath.Base(b.paths.BackupDir)
	configFileName := filepath.Base(b.paths.ConfigFile)

	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()

		steps := []types.UninstallStep{}

		addStep := func(name string, err error) {
			status := "success"
			msg := "完成"
			if err != nil {
				if os.IsNotExist(err) || strings.Contains(err.Error(), "no such file") {
					status = "success"
					msg = "已清理"
				} else {
					status = "failed"
					msg = err.Error()
				}
			}
			steps = append(steps, types.UninstallStep{Name: name, Status: status, Message: msg})
		}

		err := b.singboxSvc.Stop(ctx)
		if err != nil && strings.Contains(err.Error(), "not found") {
			err = nil
		}
		addStep("停止 sing-box 服務", err)

		out, err := b.executor.Execute(ctx, "systemctl", "disable", "sing-box")
		if err != nil && !strings.Contains(string(out), "does not exist") {
			addStep("禁用開機自啟", err)
		} else {
			addStep("禁用開機自啟", nil)
		}
		os.Remove("/etc/systemd/system/sing-box.service")
		b.executor.Execute(ctx, "systemctl", "daemon-reload")

		if binPath, err := exec.LookPath("sing-box"); err == nil {
			err := os.Remove(binPath)
			addStep("移除核心程序", err)
		} else {
			addStep("移除核心程序", nil)
		}

		if !keepLogs {
			err := os.RemoveAll(logDir)
			addStep("刪除日誌文件", err)
		} else {
			steps = append(steps, types.UninstallStep{Name: "日誌文件", Status: "success", Message: "已保留"})
		}

		keepAnyInBase := keepConfig || keepCerts || keepBackups

		if keepLogs && strings.HasPrefix(logDir, baseDir) {
			keepAnyInBase = true
		}

		if !keepAnyInBase {
			err := os.RemoveAll(baseDir)
			addStep("刪除程序目錄", err)

		} else {
			if keepConfig {
				steps = append(steps, types.UninstallStep{Name: "配置文件", Status: "success", Message: "已保留"})
			}
			if keepCerts {
				steps = append(steps, types.UninstallStep{Name: "證書文件", Status: "success", Message: "已保留"})
			}
			if keepBackups {
				steps = append(steps, types.UninstallStep{Name: "備份文件", Status: "success", Message: "已保留"})
			}

			entries, err := os.ReadDir(baseDir)
			if err == nil {
				for _, entry := range entries {
					name := entry.Name()
					fullPath := filepath.Join(baseDir, name)

					shouldKeep := false

					if keepConfig && name == configFileName {
						shouldKeep = true
					}

					if keepCerts && name == certDirName {
						shouldKeep = true
					}

					if keepBackups && name == backupDirName {
						shouldKeep = true
					}

					if keepLogs && fullPath == logDir {
						shouldKeep = true
					}

					if !shouldKeep {
						os.RemoveAll(fullPath)
					}
				}
				addStep("清理未保留文件", nil)
			}
		}

		return msg.UninstallCompleteMsg{Success: true, Steps: steps}
	}
}

// ========================================
// 輔助函數
// ========================================

func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%d秒", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%d分鐘", int(d.Minutes()))
	}
	if d < 24*time.Hour {
		hours := int(d.Hours())
		minutes := int(d.Minutes()) % 60
		return fmt.Sprintf("%d小時%d分鐘", hours, minutes)
	}
	days := int(d.Hours()) / 24
	hours := int(d.Hours()) % 24
	return fmt.Sprintf("%d天%d小時", days, hours)
}

// formatBytes 格式化字節大小
func formatBytes(bytes int64) string {
	// 處理負數情況（雖然文件大小通常不為負）
	if bytes < 0 {
		return "0 B"
	}
	if bytes < 1024 {
		return fmt.Sprintf("%d B", bytes)
	}

	// 使用浮點數計算
	fBytes := float64(bytes) / 1024.0
	units := []string{"KB", "MB", "GB", "TB", "PB"}
	unitIdx := 0

	for fBytes >= 1024 && unitIdx < len(units)-1 {
		fBytes /= 1024
		unitIdx++
	}
	return fmt.Sprintf("%.2f %s", fBytes, units[unitIdx])
}

func countLogLines(path string) int {
	file, err := os.Open(path)
	if err != nil {
		return 0
	}
	defer file.Close()
	count := 0
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		count++
	}
	return count
}

func countErrorLines(path string) int {
	file, err := os.Open(path)
	if err != nil {
		return 0
	}
	defer file.Close()
	count := 0
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, "ERROR") || strings.Contains(line, "FATAL") {
			count++
		}
	}
	return count
}

func calculateDirSize(path string) (int64, error) {
	var size int64
	err := filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !info.IsDir() {
			size += info.Size()
		}
		return nil
	})
	return size, err
}

// getPublicIP 獲取本機公網 IP
func getPublicIP() string {
	client := http.Client{
		Timeout: 3 * time.Second, // 設置超時防止卡住界面
	}

	urls := []string{
		"https://api.ipify.org?format=text",
		"https://ifconfig.me/ip",
		"https://icanhazip.com",
	}

	for _, url := range urls {
		resp, err := client.Get(url)
		if err != nil {
			continue
		}
		defer resp.Body.Close()

		ip, err := io.ReadAll(resp.Body)
		if err != nil {
			continue
		}
		return strings.TrimSpace(string(ip))
	}

	return ""
}

// getBaseServerAddress 獲取基礎服務器地址 (IP 或 手動設置的 Host)
func (b *CommandBuilder) getBaseServerAddress(cfg *domainConfig.Config) string {
	// 1. 如果配置文件手動設置了 Host (且不是 0.0.0.0)，優先使用
	if cfg.Server.Host != "0.0.0.0" && cfg.Server.Host != "" {
		return cfg.Server.Host
	}

	// 2. 否則自動獲取公網 IP
	return getPublicIP()
}
