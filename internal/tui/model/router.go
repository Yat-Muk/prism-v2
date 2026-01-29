package model

import (
	"fmt"
	"runtime"
	"strings"
	"time"

	domainConfig "github.com/Yat-Muk/prism-v2/internal/domain/config"
	"github.com/Yat-Muk/prism-v2/internal/pkg/appctx"
	"github.com/Yat-Muk/prism-v2/internal/tui/handlers"
	"github.com/Yat-Muk/prism-v2/internal/tui/msg"
	"github.com/Yat-Muk/prism-v2/internal/tui/state"
	"github.com/Yat-Muk/prism-v2/internal/tui/style"
	"github.com/Yat-Muk/prism-v2/internal/tui/types"
	tea "github.com/charmbracelet/bubbletea"
	"go.uber.org/zap"
)

type TickMsg time.Time

// 光標閃爍消息
type CursorBlinkMsg struct{}

// 用於卸載完成後的延遲退出
type UninstallExitMsg struct{}

// Router 事件路由器
type Router struct {
	stateMgr   *state.Manager
	keyHandler *handlers.KeyHandler
	cmdBuilder *handlers.CommandBuilder
	log        *zap.Logger
	paths      *appctx.Paths
}

// NewRouter 創建路由器
func NewRouter(cfg *handlers.Config) *Router {
	// 1. 初始化 CommandBuilder
	cmdBuilder := handlers.NewCommandBuilder(
		cfg.Log,
		cfg.StateMgr,
		cfg.ConfigSvc,
		cfg.PortService,
		cfg.ProtocolService,
		cfg.SingboxService,
		cfg.CertService,
		cfg.BackupMgr,
		cfg.SysInfo,
		cfg.Paths,
		cfg.Executor,
		cfg.FirewallMgr,
		cfg.ProtoFactory,
	)

	// 2. 初始化 CertHandler
	certHandler := handlers.NewCertHandler(cmdBuilder)

	// 3. 初始化 KeyHandler
	keyHandler := handlers.NewKeyHandler(cfg.StateMgr, cmdBuilder, certHandler)

	return &Router{
		stateMgr:   cfg.StateMgr,
		keyHandler: keyHandler,
		cmdBuilder: cmdBuilder,
		log:        cfg.Log,
		paths:      cfg.Paths,
	}
}

// InitModel 用於 Model.Init 調用
func (r *Router) InitModel() tea.Cmd {
	return tea.Batch(
		r.stateMgr.UI().TextInput.Focus(),
		r.cmdBuilder.UpdateDataCmd(r.stateMgr),
		r.cmdBuilder.LoadConfigCmd(r.stateMgr),
		r.cmdBuilder.RefreshCertListCmd(),
		TickCmd(),
	)
}

// Update 適配 bubbletea 的 Update 簽名
func (r *Router) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	// 優先處理內部邏輯，如果返回 cmd 則執行
	if cmd := r.routeMessage(message); cmd != nil {
		return nil, cmd
	}
	return nil, nil
}

// View 適配 bubbletea 的 View 簽名
func (r *Router) View() string {
	return r.stateMgr.Render()
}

// routeMessage 內部路由邏輯
func (r *Router) routeMessage(message tea.Msg) tea.Cmd {
	m := r.stateMgr

	// 使用 msgType 作為變量名，它已經是轉換後的具體類型
	switch msgType := message.(type) {

	// 監聽窗口大小變化消息
	case tea.WindowSizeMsg:
		m.UI().Width = msgType.Width
		m.UI().Height = msgType.Height

		if m.Node().ViewportReady {
			m.Node().Viewport.Width = msgType.Width
			m.Node().Viewport.Height = msgType.Height - 5
		}
		return nil

	case tea.KeyMsg:
		if m.UI().CurrentView == state.UninstallView && m.Uninstall().ConfirmStep == 1 {
			if msgType.Type == tea.KeyEsc {
				m.Uninstall().ConfirmStep = 0
				m.UI().TextInput.Reset()
				m.UI().SetStatus(state.StatusInfo, "已取消確認，請重新選擇", "", false)
				return nil
			}

			if msgType.Type == tea.KeyEnter || msgType.Type == tea.KeyCtrlC {
				_, cmd := r.keyHandler.Handle(msgType, m)
				return cmd
			}

			return m.UI().UpdateInput(message)
		}

		_, cmd := r.keyHandler.Handle(msgType, m)
		return cmd

	case TickMsg:
		return tea.Batch(
			r.cmdBuilder.UpdateDataCmd(m),
			TickCmd(),
		)

	// 接收到退出消息，執行標準退出流程 (框架會自動恢復終端)
	case UninstallExitMsg:
		return tea.Quit

	case msg.DataUpdateMsg:
		core := m.Core()
		core.CoreVersion = msgType.CoreVersion
		core.LatestVersion = msgType.LatestVersion
		core.HasUpdate = msgType.HasUpdate
		core.IsInstalled = msgType.IsInstalled

		m.System().Stats = &msgType.Stats
		m.System().ServiceStats = &msgType.ServiceStats
		return nil

	case msg.ConfigLoadedMsg:
		ui := m.UI()
		cfg := m.Config()

		ui.ConfigLoadState = state.ConfigLoaded

		if msgType.Err != nil {
			ui.SetStatus(state.StatusFatal, fmt.Sprintf("配置加載失敗：%v", msgType.Err), "按 Ctrl+C 退出", true)

			cfg.UpdateConfig(domainConfig.DefaultConfig())
			return nil
		} else {
			cfg.UpdateConfig(msgType.Config)
			cfg.SyncPortsToMap(m.Port())

			if !msgType.Silent {
				ui.SetStatus(state.StatusSuccess, "配置加載成功", "", false)
			}
			return nil
		}

		// 處理核心更新檢查結果
	case msg.CoreCheckMsg:
		core := m.Core()
		core.HasUpdate = msgType.HasUpdate
		core.LatestVersion = msgType.LatestVersion

		if !msgType.IsSilent {
			ui := m.UI()
			if msgType.HasUpdate {
				ui.SetStatus(state.StatusSuccess, fmt.Sprintf("發現新版本: %s", msgType.LatestVersion), "建議更新", false)
			} else {
				ui.SetStatus(state.StatusInfo, "當前已是最新版本", "", false)
			}
		}
		return nil

		// 1. 第一步完成：配置生成完畢 -> 觸發下載核心
	case msg.ConfigUpdateMsg:
		ui := m.UI()
		// 1. 處理錯誤
		if msgType.Err != nil {
			ui.SetStatus(state.StatusError, fmt.Sprintf("配置更新失敗：%v", msgType.Err), "", false)
			// 如果是在安裝流程中出錯
			if ui.CurrentView == state.InstallProgressView {
				m.Install().AddLog(fmt.Sprintf("配置失敗: %v", msgType.Err))
				m.Install().AddLog("按 [Enter] 鍵返回主菜單")
				m.Install().IsFinished = true
			}
			return nil
		}

		// 2. 更新配置數據與狀態
		if msgType.NewConfig != nil {
			m.Config().UpdateConfig(msgType.NewConfig)
			m.Config().SyncPortsToMap(m.Port())

			// 全局攔截未保存狀態
			// 只要 Applied 為 false，就強制標記為"有未保存的更改"
			if !msgType.Applied {
				m.Config().HasUnsavedChanges = true
			} else {
				// 如果已應用(保存並重啟)，則清除標記
				m.Config().HasUnsavedChanges = false
			}
		}

		// 3. 特殊流程：安裝嚮導
		if ui.CurrentView == state.InstallProgressView {
			m.Install().AddLog("端口與參數生成完畢")
			m.Install().AddLog("開始下載 sing-box 核心組件...")
			return r.cmdBuilder.InstallCoreCmdFull("latest", "")
		}

		// 4. 用戶反饋 (狀態欄)
		if msgType.Message != "" {
			// 根據是否已應用，顯示不同的狀態顏色
			statusType := state.StatusSuccess
			if !msgType.Applied {
				statusType = state.StatusInfo // 未保存只顯示藍色提示，不是綠色成功
			}
			ui.SetStatus(statusType, msgType.Message, "", false)
		} else if !msgType.Applied {
			// 如果沒有消息且未保存，給一個默認提示
			ui.SetStatus(state.StatusInfo, "配置已更改 (未保存)", "", false)
		}

		// 5. 如果已應用，刷新全局數據 (例如重載後的狀態)
		if msgType.Applied {
			return tea.Batch(
				r.cmdBuilder.UpdateDataCmd(m),
				r.cmdBuilder.LoadConfigCmd(m),
			)
		}
		return nil

		// 2. 第二步完成：核心安裝完畢 -> 觸發寫入配置並啟動
	case msg.CoreInstallMsg:
		ui := m.UI()

		if msgType.Success {
			// === 成功分支 (安裝嚮導邏輯) ===
			if ui.CurrentView == state.InstallProgressView {
				m.Install().AddLog(fmt.Sprintf("核心下載與安裝成功 (v%s)", msgType.Version))
				m.Install().AddLog("正在註冊系統服務並寫入配置...")
				return r.cmdBuilder.InstallPrismCmd(m)
			}

			// === 成功分支 (手動更新場景) ===
			m.Core().CoreVersion = msgType.Version
			m.Core().IsInstalled = msgType.Installed
			m.Core().HasUpdate = false

			// 1. 設置成功/提示消息
			ui.SetStatus(state.StatusSuccess, msgType.Message, "", false)

			// 2. 智能判斷：是否需要切換視圖
			if ui.CurrentView == state.CoreMenuView {
				return tea.Batch(
					r.cmdBuilder.UpdateDataCmd(m), // 刷新數據
					ui.TextInput.Focus(),          // 保持光標閃爍
				)
			}

			// 如果是從其他頁面(如安裝嚮導)過來的，則必須切換
			cmd1 := ui.SwitchView(state.CoreMenuView)
			cmd2 := r.cmdBuilder.UpdateDataCmd(m)
			return tea.Batch(cmd1, cmd2)

		} else {
			// === 失敗分支 ===
			errText := fmt.Sprintf("更新失敗: %v", msgType.Err)
			ui.SetStatus(state.StatusError, errText, "請檢查網絡", false)

			if ui.CurrentView == state.InstallProgressView {
				m.Install().AddLog("核心安裝失敗: " + msgType.Err.Error())
				m.Install().AddLog("按 [Enter] 鍵返回主菜單")
				m.Install().IsFinished = true
				return nil
			}
		}
		return r.cmdBuilder.UpdateDataCmd(m)

		// 2. 監聽安裝結果 (鏈式反應的終點)
	case msg.InstallResultMsg:
		ui := m.UI()

		// 1. 標記流程結束
		m.Install().IsFinished = true

		if msgType.Err != nil {
			ui.SetStatus(state.StatusError, fmt.Sprintf("安裝失敗：%v", msgType.Err), "按 Enter 返回", false)
			if ui.CurrentView == state.InstallProgressView {
				m.Install().AddLog(fmt.Sprintf("安裝失敗: %v", msgType.Err))
				m.Install().AddLog("請截圖或複製錯誤日誌")
				m.Install().AddLog("按 [Enter] 鍵返回主菜單") // 提示用戶
			}
		} else {
			// 安裝成功
			if ui.CurrentView == state.InstallProgressView {
				m.Install().AddLog("核心安裝成功！")
				m.Install().AddLog("服務已啟動")
				m.Install().AddLog("所有步驟已完成")
				m.Install().AddLog("按 [Enter] 鍵返回主菜單") // 提示用戶
			}

			// 2. 不再自動跳轉，只更新狀態欄提示
			ui.SetStatus(state.StatusSuccess, "安裝成功！", "按 Enter 返回主菜單", false)
		}

		// 這裡不需要返回任何 cmd，靜靜等待用戶按鍵
		return nil

	// 3. 處理視圖跳轉消息
	case msg.ViewChangeMsg:
		// ✅ 修正：將 int 類型的 ViewID 顯式轉換為 state.View 類型
		return tea.Batch(
			m.UI().SwitchView(state.View(msgType.ViewID)),
			r.cmdBuilder.UpdateDataCmd(m),
		)

	case msg.UUIDGeneratedMsg:
		// 1. 更新內存配置
		m.Config().Config.UUID = msgType.UUID

		// 2. 標記為未保存 (Dirty)
		m.Config().HasUnsavedChanges = true
		m.Config().Dirty = true

		// 3. UI 反饋
		m.UI().SetStatus(state.StatusInfo, "UUID 已生成 (未保存)", msgType.UUID, false)

		// 4. 可選：清空輸入框
		m.UI().ClearInput()
		return nil

	case msg.ServiceResultMsg:
		ui := m.UI()
		if msgType.Err != nil {
			ui.SetStatus(state.StatusError, fmt.Sprintf("服務操作失敗：%v", msgType.Err), "", false)
		} else {
			actionText := msgType.Action
			switch msgType.Action {
			case "start":
				actionText = "啟動"
			case "stop":
				actionText = "停止"
			case "restart":
				actionText = "重啟"
			}
			ui.SetStatus(state.StatusSuccess, fmt.Sprintf("服務%s成功", actionText), "", false)
		}
		return r.cmdBuilder.UpdateDataCmd(m)

	case msg.CertRequestMsg:
		m.UI().SetStatus(state.StatusSuccess, "就緒", "", false)

		if msgType.Err != nil {
			errStr := msgType.Err.Error()

			var title, desc string

			if strings.Contains(errStr, "IP 不匹配") {
				title = errStr
				desc = "請檢查 DNS 解析是否已生效"
			} else if strings.Contains(errStr, "DNS") {
				title = errStr
				desc = "無法解析域名，請檢查輸入"
			} else {
				if len(errStr) < 50 {
					title = errStr
					desc = "證書申請失敗"
				} else {
					title = "證書申請失敗"
					desc = errStr
				}
			}

			m.UI().SetStatus(state.StatusError, title, desc, false)
			return nil
		} else {
			m.UI().SetStatus(state.StatusSuccess, "證書申請成功！", msgType.Domain, false)
			switchCmd := m.UI().SwitchView(state.CertMenuView)
			return tea.Batch(switchCmd, r.cmdBuilder.RefreshCertListCmd())
		}

	case msg.CertRenewMsg:
		ui := m.UI()
		if msgType.Err != nil {
			ui.SetStatus(state.StatusError, fmt.Sprintf("續期失敗：%v", msgType.Err), "", false)
		} else {
			ui.SetStatus(state.StatusSuccess, "證書續期成功", "", false)
		}
		return r.cmdBuilder.RefreshCertListCmd()

	case msg.CertDeleteMsg:
		ui := m.UI()
		if msgType.Err != nil {
			ui.SetStatus(state.StatusError, fmt.Sprintf("刪除失敗：%v", msgType.Err), "", false)
			return nil
		} else {
			ui.SetStatus(state.StatusSuccess, "證書已刪除", "", false)
			switchCmd := ui.SwitchView(state.CertMenuView)
			return tea.Batch(switchCmd, r.cmdBuilder.RefreshCertListCmd())
		}

	case msg.CertListRefreshMsg:
		m.Cert().RefreshCertList(
			msgType.ACMEDomains,
			msgType.SelfSignedDomains,
			msgType.CertList,
			msgType.CurrentCAProvider,
		)
		return nil

	case msg.ProviderSwitchMsg:
		ui := m.UI()
		if msgType.Err != nil {
			ui.SetStatus(
				state.StatusError,
				fmt.Sprintf("切換 CA 失敗：%v", msgType.Err),
				"",
				false,
			)
		} else {
			providerNames := map[string]string{
				"letsencrypt": "Let's Encrypt",
				"zerossl":     "ZeroSSL",
			}
			displayName := providerNames[msgType.Provider]
			if displayName == "" {
				displayName = msgType.Provider
			}

			ui.SetStatus(
				state.StatusSuccess,
				fmt.Sprintf("已切換到 %s，新證書將使用此 CA", displayName),
				"",
				false,
			)
		}
		return r.cmdBuilder.RefreshCertListCmd()

	case msg.RoutingConfigLoadedMsg:
		ui := m.UI()
		if msgType.Err != nil {
			ui.SetStatus(state.StatusError, fmt.Sprintf("加載路由配置失敗: %v", msgType.Err), "", false)
		}

		routing := m.Routing()
		switch msgType.Type {
		case "warp":
			if cfg, ok := msgType.Config.(domainConfig.WARPConfig); ok {
				routing.LoadWARPConfig(&cfg)
			}
		case "socks5":
			if cfg, ok := msgType.Config.(domainConfig.Socks5Config); ok {
				routing.LoadSocks5Config(&cfg)
			}
		case "ipv6":
			if cfg, ok := msgType.Config.(domainConfig.IPv6SplitConfig); ok {
				routing.LoadIPv6Config(&cfg)
			}
		case "dns":
			if cfg, ok := msgType.Config.(domainConfig.DNSRoutingConfig); ok {
				routing.LoadDNSConfig(&cfg)
			}
		case "sni_proxy":
			if cfg, ok := msgType.Config.(domainConfig.SNIProxyConfig); ok {
				routing.LoadSNIProxyConfig(&cfg)
			}
		}
		return nil

	case msg.BackupListMsg:
		ui := m.UI()
		m.Backup().SetBackupList(msgType.Entries)
		if msgType.Err != nil {
			ui.SetStatus(state.StatusError, fmt.Sprintf("獲取備份列表失敗：%v", msgType.Err), "", false)
		} else {
			ui.SetStatus(state.StatusInfo, "備份列表已刷新", "", false)
		}
		return nil

	case msg.BackupCreateMsg:
		ui := m.UI()
		if msgType.Err != nil {
			ui.SetStatus(state.StatusError, fmt.Sprintf("備份失敗：%v", msgType.Err), "", false)
		} else {
			ui.SetStatus(state.StatusSuccess, "配置已成功備份", "", false)
			return r.cmdBuilder.ListBackupsCmd(m, 5)
		}
		return nil

	case msg.BackupRestoreMsg:
		ui := m.UI()
		if msgType.Err != nil {
			ui.SetStatus(state.StatusFatal, fmt.Sprintf("恢復失敗：%v", msgType.Err), "", true)
		} else {
			ui.SetStatus(state.StatusSuccess, "配置已從備份恢復並應用", "", false)
		}
		return nil

	case msg.SystemInfoLoadedMsg:
		m.UI().SetStatus(state.StatusSuccess, fmt.Sprintf("已加載 %s 信息", msgType.Type), "", false)
		return nil

	case msg.StreamingCheckResultMsg:
		ui := m.UI()
		m.Tools().IsCheckingStreaming = false

		if msgType.Err != nil {
			ui.SetStatus(state.StatusError, "流媒體檢測失敗", msgType.Err.Error(), false)
		} else {
			m.Tools().StreamingResult = msgType.Result

			ui.SetStatus(state.StatusSuccess, "檢測完成", "", false)
		}

		return nil

	case msg.SwapInfoMsg:
		ui := m.UI()
		if msgType.Err != nil {
			ui.SetStatus(state.StatusError, "獲取 Swap 信息失敗", msgType.Err.Error(), false)
		} else {
			m.Tools().SwapInfo = msgType.Info
		}
		return nil

	case msg.Fail2BanInfoMsg:
		ui := m.UI()
		if msgType.Err != nil {
			ui.SetStatus(state.StatusError, "獲取 Fail2Ban 信息失敗", msgType.Err.Error(), false)
		} else {
			m.Tools().Fail2BanInfo = msgType.Info
		}
		return nil

	case msg.CleanupInfoMsg:
		ui := m.UI()
		if msgType.Err != nil {
			ui.SetStatus(state.StatusError, "系統掃描失敗", msgType.Err.Error(), false)
		} else {
			m.Tools().CleanupInfo = msgType.Info

			ui.SetStatus(state.StatusSuccess, "系統掃描完成", "", false)
		}
		return nil

	// 處理 BBR 信息更新
	case msg.BBRInfoMsg:
		ui := m.UI()
		if msgType.Err != nil {
			ui.SetStatus(state.StatusError, "獲取 BBR 信息失敗", msgType.Err.Error(), false)
		} else {
			m.Tools().BBRInfo = msgType.Info
		}
		return nil

	case msg.CommandResultMsg:
		ui := m.UI()

		if msgType.Data == "PORT80_CHECK" {
			if msgType.Success {
				ui.SetStatus(state.StatusSuccess, "環境檢測通過", "請輸入 ACME 賬戶郵箱", false)
				return ui.SwitchView(state.ACMEHTTPInputView)
			} else {
				errInfo := msgType.Message
				if msgType.Err != nil {
					errInfo = msgType.Err.Error()
				}
				ui.SetStatus(state.StatusError, "80 端口不可用", errInfo, false)
				return nil
			}
		}

		if msgType.Data == "CERT_MODE_SWITCH" {
			if msgType.Success {
				ui.SetStatus(state.StatusSuccess, msgType.Message, "", false)

				return tea.Batch(
					r.cmdBuilder.LoadConfigSilentCmd(m),
					r.cmdBuilder.RefreshCertListCmd(),
				)
			}
		}

		// 處理刪除 Swap 後的自動刷新
		if msgType.Data == "REFRESH_SWAP" {
			ui.SetStatus(state.StatusSuccess, msgType.Message, "", false)
			return r.cmdBuilder.LoadSwapInfoCmd(m)
		}

		// 處理 Fail2Ban 刷新
		if msgType.Data == "REFRESH_FAIL2BAN" {
			if msgType.Success {
				ui.SetStatus(state.StatusSuccess, msgType.Message, "", false)
				return r.cmdBuilder.LoadFail2BanInfoCmd(m)
			}
		}

		// 處理自動刷新列表
		if msgType.Data == "REFRESH_FAIL2BAN_LIST" {
			if msgType.Success {
				ui.SetStatus(state.StatusSuccess, msgType.Message, "", false)
				// 觸發獲取新列表
				return r.cmdBuilder.GetFail2BanListCmd(m)
			}
		}

		// 處理 BBR
		if msgType.Data == "REFRESH_BBR" {
			if msgType.Success {
				ui.SetStatus(state.StatusSuccess, msgType.Message, "", false)
				return r.cmdBuilder.LoadBBRInfoCmd(m)
			}
		}

		// 攔截 BBR2 不支持信號，進行引導
		if msgType.Data == "BBR2_NOT_SUPPORTED" {
			// 1. 檢測架構
			if runtime.GOARCH == "amd64" {
				// 2. 引導用戶
				m.Tools().BBRInstallConfirm = true
				m.Tools().BBRInstallTarget = "bbr2" // 標記目標為 BBR2
				ui.SetStatus(state.StatusWarn, "當前內核不支持 BBR2", "檢測到 x86_64 架構，輸入 YES 可安裝專用內核", true)
				// 自動聚焦輸入框
				return ui.TextInput.Focus()
			} else {
				// 非 amd64，直接報錯
				ui.SetStatus(state.StatusError, "當前架構不支持 BBR2", "BBR2 僅支持 x86_64", false)
				return nil
			}
		}

		if msgType.Success {
			ui.SetStatus(state.StatusSuccess, msgType.Message, "", false)
		} else {
			ui.SetStatus(state.StatusError, msgType.Message, "", false)
		}
		return nil

	case msg.LogInfoMsg:
		ui := m.UI()
		if msgType.Err != nil {
			ui.SetStatus(state.StatusError, fmt.Sprintf("加載日誌信息失敗: %v", msgType.Err), "", false)
		} else {
			m.Tools().LogInfo = &types.LogInfo{
				LogLevel:   msgType.LogLevel,
				LogPath:    msgType.LogPath,
				LogSize:    msgType.LogSize,
				TodayLines: msgType.TodayLines,
				ErrorCount: msgType.ErrorCount,
				RecentLogs: msgType.RecentLogs,
			}
		}
		return nil

	case msg.LogViewMsg:
		ui := m.UI()
		if msgType.Err != nil {
			ui.SetStatus(state.StatusError, fmt.Sprintf("獲取失敗: %v", msgType.Err), "", false)
			return nil
		}

		// 分支 A: Fail2Ban 列表模式
		if msgType.Mode == "fail2ban_list" {
			m.Tools().Fail2BanLogOutput = msgType.Logs

			// 只有當用戶明確是在"查看列表"(非輸入模式)時，才跳轉
			if !m.Tools().Fail2BanInputMode {
				m.Tools().IsViewingFail2BanLogs = true
				return ui.SwitchView(state.LogViewerView)
			}
			// 輸入模式下，什麼都不做(停留在當前頁面)
			return nil
		}

		// 分支 B: 標準日誌查看模式 (實時/完整/錯誤日誌)

		// 1. 轉換日誌數組為帶顏色的長字符串
		formattedContent := style.BuildColoredLogContent(msgType.Logs)

		// 2. 更新 LogState (寫入 Viewport 並滾動到底部)
		m.Log().UpdateContent(formattedContent)

		// 3. 設置是否為跟蹤模式 (決定狀態欄顯示內容)
		m.Log().IsFollowing = (msgType.Mode == "realtime")

		return ui.SwitchView(state.LogViewerView)

	case msg.CoreVersionsMsg:
		if msgType.Err != nil {
			m.UI().SetStatus(state.StatusError, fmt.Sprintf("加載版本列表失敗: %v", msgType.Err), "", false)
			m.UI().SwitchView(state.CoreMenuView)
		} else {
			m.Core().AvailableVers = msgType.Versions
		}
		return nil

	case msg.ScriptCheckMsg:
		m.Core().IsCheckingScript = false

		if msgType.Success {
			m.Core().ScriptLatestVersion = "v" + msgType.LatestVer
			m.Core().ScriptChangelog = msgType.Changelog
			m.UI().SetStatus(state.StatusSuccess, "發現新版本，請確認更新", "", false)
		} else {
			m.UI().SetStatus(state.StatusError, "檢查更新失敗", msgType.Err.Error(), false)
			m.Core().ScriptLatestVersion = ""
		}

		return nil

	case msg.ServiceHealthMsg:
		if msgType.Err != nil {
			m.UI().SetStatus(state.StatusError, fmt.Sprintf("健康檢查失敗: %v", msgType.Err), "", false)
			m.UI().SwitchView(state.ServiceMenuView)
		}
		return nil

	case msg.ServiceAutoStartMsg:
		if msgType.Err != nil {
			m.UI().SetStatus(state.StatusError, fmt.Sprintf("設置自啟動失敗: %v", msgType.Err), "", false)
		} else {
			m.Service().AutoStart = msgType.Enabled
			status := "已禁用"
			if msgType.Enabled {
				status = "已啟用"
			}
			m.UI().SetStatus(state.StatusSuccess, fmt.Sprintf("開機自啟%s", status), "", false)
		}
		return nil

	case msg.NodeInfoMsg:
		ui := m.UI()
		node := m.Node() // 獲取 NodeState

		if msgType.Err != nil {
			ui.SetStatus(state.StatusError, fmt.Sprintf("操作失敗: %v", msgType.Err), "", false)
			return nil
		}

		// 根據類型更新 State
		switch msgType.Type {
		case "protocol_links":
			node.Links = msgType.Links
			if msgType.Info != nil {
				node.NodeInfo = msgType.Info
			}
			ui.SetStatus(state.StatusSuccess, "", "", false)

		case "subscription":
			node.Subscription = msgType.Subscription
			ui.SetStatus(state.StatusSuccess, "訂閱信息已更新", "", false)

		case "qrcode":
			node.CurrentQRCode = msgType.QRCode
			node.CurrentQRContent = msgType.Content
			node.CurrentQRType = msgType.QRType

		case "client_config":
			node.ClientConfig = msgType.ClientConfig
			ui.SetStatus(state.StatusSuccess, "客戶端配置已導出", msgType.ClientConfig.FilePath, false)
		}
		return nil

	case msg.NodeParamsDataMsg:
		ui := m.UI()
		if msgType.Err != nil {
			ui.SetStatus(state.StatusError, fmt.Sprintf("獲取參數失敗: %v", msgType.Err), "", false)
			return nil
		}

		m.Node().SetParamsData(msgType.ServerIP)

		ui.SetStatus(state.StatusSuccess, "參數已加載", "使用 ↑ ↓ 鍵滾動查看，按 Esc 返回", false)
		return nil

	// ===============================================
	// 卸載相關邏輯
	// ===============================================

	// 處理卸載掃描完成消息
	case msg.UninstallInfoMsg:
		// 直接使用 msgType.Info，因為 msgType 已經是轉換好的類型
		m.Uninstall().Info = msgType.Info
		m.Uninstall().Scanning = false
		return nil

		// 處理卸載執行完成消息
	case msg.UninstallCompleteMsg:
		uState := m.Uninstall()
		uState.Uninstalling = false
		uState.Steps = msgType.Steps

		if msgType.Success {
			cmd1 := m.UI().SwitchView(state.UninstallProgressView)

			m.UI().SetStatus(state.StatusSuccess, "Prism 已成功卸載，程序將在 3 秒後退出", "", false)

			cmd2 := tea.Tick(3*time.Second, func(_ time.Time) tea.Msg {
				return UninstallExitMsg{}
			})

			return tea.Batch(cmd1, cmd2)
		} else {
			m.UI().SetStatus(state.StatusError, "卸載過程中發生錯誤", "", false)
			m.UI().SwitchView(state.UninstallProgressView)
		}
		return nil

	default:
		// 處理協議鏈接頁面的滾動
		if m.UI().CurrentView == state.ProtocolLinksView && m.Node().ViewportReady {
			var cmd tea.Cmd
			m.Node().Viewport, cmd = m.Node().Viewport.Update(message)

			// 同時處理通用輸入 (InputBox)
			inputCmd := m.UI().UpdateInput(message)
			return tea.Batch(cmd, inputCmd)
		}

		// 標準處理：同時更新 Spinner 和 TextInput
		var cmd tea.Cmd
		m.UI().Spinner, cmd = m.UI().Spinner.Update(message)

		// 處理輸入框更新
		inputCmd := m.UI().UpdateInput(message)

		return tea.Batch(cmd, inputCmd)
	}
}

// TickCmd 定時器
func TickCmd() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return TickMsg(t)
	})
}
