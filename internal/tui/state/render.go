package state

import (
	"fmt"
	"strings"

	domainConfig "github.com/Yat-Muk/prism-v2/internal/domain/config"
	"github.com/Yat-Muk/prism-v2/internal/tui/types"
	"github.com/Yat-Muk/prism-v2/internal/tui/view"
	"github.com/charmbracelet/bubbles/viewport"
	"github.com/charmbracelet/lipgloss"
)

// Render - 安全渲染視圖
func (m *Manager) Render() string {
	if m.ui.ConfigLoadState == ConfigNotLoaded {
		return view.RenderLoading("初始化配置中...")
	}

	width := m.ui.Width
	if width == 0 {
		width = 80
	}

	// 獲取全局狀態消息
	statusMsg := m.ui.Status.Message
	if m.ui.Status.Detail != "" {
		statusMsg = fmt.Sprintf("%s\n%s", statusMsg, m.ui.Status.Detail)
	}

	cursor := m.ui.Cursor
	ti := m.ui.TextInput // 獲取完整的 TextInput 模型

	switch m.ui.CurrentView {
	case MainMenuView:
		var stats *types.SystemStats
		var svcStats *types.ServiceStats
		if m.system != nil {
			stats = m.system.Stats
			svcStats = m.system.ServiceStats
		}
		return view.RenderMainView(
			stats,
			svcStats,
			cursor,
			width,
			m.core.IsInstalled,
			m.core.CoreVersion,
			m.core.HasUpdate,
			m.core.LatestVersion,
			m.core.ScriptVersion,
			m.core.ScriptLatestVersion,
			ti,
			statusMsg,
		)

	case InstallWizardView:
		return view.RenderInstallWizard(
			m.install.InstallProtocols,
			ti,
			statusMsg,
		)

	case InstallProgressView:
		return view.RenderInstallProgress(m.install.Logs, m.ui.Spinner)

	case ConfigMenuView:
		return view.RenderConfigMenu(cursor, ti, statusMsg)

	case ProtocolMenuView:
		return view.RenderProtocolSwitches(
			m.config.EnabledProtocols,
			ti,
			statusMsg,
		)

	case PortEditView:
		return view.RenderPortEdit(
			m.config.EnabledProtocols,
			m.port.CurrentPorts,
			m.port.Hy2HoppingRange,
			ti,
			statusMsg,
		)

	case Hy2PortModeView:
		hy2Port := 0
		if port, exists := m.port.CurrentPorts[3]; exists {
			hy2Port = port
		}

		return view.RenderHy2PortMode(
			hy2Port,
			m.port.Hy2HoppingRange,
			ti,
			statusMsg,
			m.port.PortEditingMode,
		)

	case SNIEditView:
		current := ""
		if cfg := m.config.GetConfig(); cfg != nil {
			current = cfg.Protocols.RealityVision.SNI
		}
		return view.RenderSNIEditView(current, ti, statusMsg)

	case UUIDEditView:
		current := ""
		if cfg := m.config.GetConfig(); cfg != nil {
			current = cfg.UUID
		}
		return view.RenderUUIDEditView(current, ti, statusMsg)

	case AnyTLSPaddingView:
		current := ""
		if cfg := m.config.GetConfig(); cfg != nil {
			current = cfg.Protocols.AnyTLS.PaddingMode
		}
		return view.RenderAnyTLSPaddingMenu(ti, current, statusMsg)

	case OutboundMenuView:
		v4, v6 := false, false
		if m.system != nil && m.system.Stats != nil {
			v4 = m.system.Stats.IPv4 != ""
			v6 = m.system.Stats.IPv6 != ""
		}

		strat := "prefer_ipv4"
		if cfg := m.config.GetConfig(); cfg != nil && cfg.Routing.DomainStrategy != "" {
			strat = cfg.Routing.DomainStrategy
		}
		return view.RenderOutboundMenu(cursor, strat, ti, v4, v6, statusMsg)

	case ToolsMenuView:
		return view.RenderToolsMenu(cursor, ti, statusMsg)

	case BackupMenuView:
		return view.RenderBackupRestoreMenu(
			cursor,
			m.backup.BackupList,
			ti,
			statusMsg,
			m.backup.SelectedBackup,
			m.backup.ConfirmMode,
			m.backup.PendingOp,
		)

	case LogMenuView:
		return view.RenderLogMenu(m.tools.LogInfo, ti, statusMsg)

	// 日誌查看器 (統一處理 Fail2Ban 和 常規日誌)
	case LogViewerView:
		// 1. 確保 Viewport 已初始化
		if !m.logState.ViewportReady {
			m.logState.Viewport = viewport.New(50, 15)
			m.logState.ViewportReady = true
		}

		// 2. 處理 Fail2Ban 列表
		if m.tools.IsViewingFail2BanLogs {
			return view.RenderFail2BanList(
				m.tools.Fail2BanLogOutput,
				ti,
				statusMsg,
			)
		}

		// 3. 處理常規日誌
		return view.RenderLogViewer(
			view.LogViewerModeFull,
			nil,
			m.logState.Viewport,
			m.logState.IsFollowing,
		)

	// ===== 證書相關視圖 =====
	case CertMenuView:
		return view.RenderCertMenu(
			m.cert.ACMEDomains,
			m.cert.SelfSignedDomains,
			cursor,
			ti,
			statusMsg,
		)

	case ACMEHTTPInputView:
		return view.RenderACMEHTTPInput(
			m.cert.HTTPStep,
			m.cert.ACMEEmail,
			ti,
			statusMsg,
		)

	case ACMEDNSProviderView:
		return view.RenderACMEDNSProvider(ti, statusMsg)

	case DNSCredentialInputView:
		return view.RenderDNSCredentialInput(
			m.cert.SelectedProvider,
			m.cert.DNSStep,
			ti,
			statusMsg,
		)

	case ProviderSwitchView:
		return view.RenderProviderSwitch(
			m.cert.CurrentCAProvider,
			ti,
			statusMsg,
		)

	case CertRenewView:
		return view.RenderCertRenew(
			m.cert.ACMEDomains,
			ti,
			statusMsg,
		)

	case CertDeleteView:
		return view.RenderCertDelete(
			m.cert.ACMEDomains,
			m.cert.CertConfirmMode,
			m.cert.CertToDelete,
			ti,
			statusMsg,
		)

	case CertStatusView:
		return view.RenderCertStatus(
			m.cert.CertList,
			ti,
			statusMsg,
		)

	case CertModeMenuView:
		cfg := m.config.GetConfig()
		if cfg == nil {
			cfg = domainConfig.DefaultConfig()
		}
		return view.RenderCertModeMenu(
			cfg,
			m.config.EnabledProtocols,
			m.cert.ACMEDomains,
			m.cert.SelfSignedDomains,
			cursor,
			ti,
			statusMsg,
		)

	// ===== 路由相關視圖 =====
	case RouteMenuView:
		return view.RenderRouteMenu(cursor, ti, statusMsg)

	case WARPRoutingView:
		cfg := m.config.GetConfig()
		var warpCfg *domainConfig.WARPConfig
		if cfg != nil {
			warpCfg = &cfg.Routing.WARP
		}
		return view.RenderWARPRouting(warpCfg, ti, statusMsg)

	case Socks5RoutingView:
		cfg := m.config.GetConfig()
		var socks5Cfg *domainConfig.Socks5Config
		if cfg != nil {
			socks5Cfg = &cfg.Routing.Socks5
		}
		return view.RenderSocks5Routing(socks5Cfg, ti, statusMsg)

	case Socks5InboundView:
		cfg := m.config.GetConfig()
		var inCfg *domainConfig.Socks5InboundConfig
		if cfg != nil {
			inCfg = &cfg.Routing.Socks5.Inbound
		}
		return view.RenderSocks5Inbound(inCfg, ti, statusMsg)

	case Socks5OutboundView:
		cfg := m.config.GetConfig()
		var outCfg *domainConfig.Socks5OutboundConfig
		if cfg != nil {
			outCfg = &cfg.Routing.Socks5.Outbound
		}
		return view.RenderSocks5Outbound(outCfg, ti, statusMsg)

	case IPv6RoutingView:
		cfg := m.config.GetConfig()
		var ipv6Cfg *domainConfig.IPv6SplitConfig
		if cfg != nil {
			ipv6Cfg = &cfg.Routing.IPv6Split
		}
		return view.RenderIPv6Routing(ipv6Cfg, ti, statusMsg)

	case DNSRoutingView:
		cfg := m.config.GetConfig()
		var dnsCfg *domainConfig.DNSRoutingConfig
		if cfg != nil {
			dnsCfg = &cfg.Routing.DNS
		}
		return view.RenderDNSRouting(dnsCfg, ti, statusMsg)

	case SNIProxyRoutingView:
		cfg := m.config.GetConfig()
		var sniCfg *domainConfig.SNIProxyConfig
		if cfg != nil {
			sniCfg = &cfg.Routing.SNIProxy
		}
		return view.RenderSNIProxyRouting(sniCfg, ti, statusMsg)

	// ===== 核心管理 =====
	case CoreMenuView:
		return view.RenderCoreMenu(
			m.core.CoreVersion,
			m.core.LatestVersion,
			m.core.HasUpdate,
			m.core.IsInstalled,
			m.core.ScriptVersion,
			ti,
			statusMsg,
		)

	case CoreVersionSelectView:
		return view.RenderCoreVersionSelect(
			m.core.AvailableVers,
			m.core.CoreVersion,
			m.core.LatestVersion,
			ti,
			statusMsg,
		)

	case CoreSourceSelectView:
		sources := []view.CoreSource{
			{Name: "GitHub Official", URL: "github.com/SagerNet/sing-box"},
			{Name: "GHProxy Mirror", URL: "ghproxy.com"},
		}
		idx := 0
		if m.core.UpdateSource != "github" {
			idx = 1
		}
		return view.RenderCoreSourceSelect(
			sources,
			idx,
			ti,
			statusMsg,
		)

	case ScriptUpdateView:
		return view.RenderScriptUpdate(
			m.core.ScriptVersion,
			m.core.ScriptLatestVersion,
			m.core.ScriptChangelog,
			m.core.IsCheckingScript,
			ti,
			statusMsg,
		)

	// ===== 服務管理 =====
	case ServiceMenuView:
		// ✅ 修正：使用 types 包的結構體
		var svcStats *types.ServiceStats
		if m.system != nil {
			svcStats = m.system.ServiceStats
		}
		return view.RenderServiceMenu(svcStats, m.service.AutoStart, ti, statusMsg)

	case ServiceLogView:
		logs := []string{"等待日誌..."}
		return view.RenderServiceLogViewer(logs, true, ti)

	case ServiceHealthView:
		// 假設 service.HealthCheck 已經適配為 types.HealthCheckResult，如果還未適配，這裡傳 nil 防止崩潰
		// 根據您的代碼，view.RenderServiceHealth 接收 *types.HealthCheckResult
		return view.RenderServiceHealth(m.service.HealthCheck, ti, statusMsg)

	// ===== 工具箱 =====
	case BBRMenuView:
		return view.RenderBBRMenu(
			m.tools.BBRInfo,
			m.tools.BBRInstallConfirm,
			m.tools.BBRInstallTarget,
			ti,
			statusMsg,
		)

	case Fail2BanMenuView:
		return view.RenderFail2BanMenu(
			m.tools.Fail2BanInfo,
			m.tools.Fail2BanLogOutput, // 封禁列表數據
			m.tools.Fail2BanInputMode, // 是否正在輸入
			ti,
			statusMsg,
		)

	case LogLevelEditView:
		currentLevel := "info"
		if cfg := m.config.GetConfig(); cfg != nil {
			currentLevel = cfg.Log.Level
		}
		return view.RenderLogLevelEdit(currentLevel, ti, statusMsg)

	case StreamingCheckView:
		return view.RenderStreamingCheck(
			m.tools.StreamingResult,
			m.tools.IsCheckingStreaming,
			ti,
			statusMsg,
		)

	case SwapMenuView:
		return view.RenderSwapMenu(
			m.tools.SwapInfo,
			m.tools.SwapInputMode,
			ti,
			statusMsg,
		)

	case SystemCleanupView:
		return view.RenderSystemCleanup(m.tools.CleanupInfo, ti, statusMsg)

	// ===== 節點信息與客戶端導出 =====
	case NodeInfoView:
		return view.RenderNodeInfo(m.Node().NodeInfo, ti, statusMsg)

	case ProtocolLinksView:
		// [參數查看模式]
		if m.Node().SelectionMode == "params" {
			cfg := m.config.GetConfig()
			if cfg == nil {
				return "配置未加載"
			}

			// 1. 修定義並計算 statusHeight
			statusHeight := strings.Count(statusMsg, "\n") + 1
			if statusMsg == "" {
				statusHeight = 0
			}

			// 2. 計算可用高度
			// 邏輯：屏幕總高度 - 狀態欄高度 - 1行底部留白
			availableHeight := m.ui.Height - statusHeight - 1

			// 3. 設置保底高度 (防止窗口極小時崩潰)
			if availableHeight < 5 {
				availableHeight = 5
			}

			// 4. 初始化或更新 Viewport
			if !m.Node().ViewportReady {
				m.Node().Viewport = viewport.New(width, availableHeight)
				m.Node().ViewportReady = true
			} else {
				m.Node().Viewport.Width = width
				m.Node().Viewport.Height = availableHeight
			}

			// 5. 獲取並設置內容
			content := view.RenderNodeParams(cfg, m.Node().ParamsIP)
			m.Node().Viewport.SetContent(content)

			// 6. 渲染視圖 (內容 + 狀態欄)
			statusStyle := lipgloss.NewStyle().
				Foreground(lipgloss.Color("241")).
				MarginTop(1)

			renderedStatus := statusStyle.Render(statusMsg)

			// 垂直拼接：上方 Viewport，下方狀態欄
			return lipgloss.JoinVertical(
				lipgloss.Left,
				m.Node().Viewport.View(),
				renderedStatus,
			)
		}

		// 普通鏈接列表模式
		return view.RenderProtocolLinks(
			m.Node().Links,
			m.Node().SelectionMode,
			ti,
			statusMsg,
		)

	case SubscriptionView:
		return view.RenderSubscription(m.Node().Subscription, ti, statusMsg)

	case QRCodeView:
		displayContent := m.Node().CurrentQRContent
		if m.Node().CurrentQRType == "protocol" && m.Node().CurrentQRIndex >= 0 {
			if m.Node().CurrentQRIndex < len(m.Node().Links) {
				displayContent = m.Node().Links[m.Node().CurrentQRIndex].Name
			}
		}

		return view.RenderQRCode(
			m.Node().CurrentQRCode,
			displayContent,
			m.Node().CurrentQRType,
			ti,
			statusMsg,
		)

	case ClientConfigView:
		return view.RenderClientConfig(m.Node().ClientConfig, ti, statusMsg)

		// ===== 卸載 =====
	case UninstallView:
		uState := m.Uninstall()
		if uState.Info != nil {
			uState.Info.ConfirmStep = uState.ConfirmStep
			uState.Info.KeepConfig = uState.KeepConfig
			uState.Info.KeepCerts = uState.KeepCerts
			uState.Info.KeepBackups = uState.KeepBackups
			uState.Info.KeepLogs = uState.KeepLogs
		}

		return view.RenderUninstall(uState.Info, ti, statusMsg)

	case UninstallProgressView:
		return view.RenderUninstallProgress(m.Uninstall().Steps, ti, statusMsg)

	default:
		return view.RenderMainView(
			nil, nil, cursor, width, false, "", false, "", "Dev", "", ti, statusMsg)
	}
}
