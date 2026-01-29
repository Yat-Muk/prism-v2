package handlers

import (
	"fmt"
	"net"
	"runtime"
	"sort"
	"strconv"
	"strings"

	"github.com/Yat-Muk/prism-v2/internal/domain/config"
	"github.com/Yat-Muk/prism-v2/internal/pkg/inputvalidator"
	"github.com/Yat-Muk/prism-v2/internal/tui/constants"
	"github.com/Yat-Muk/prism-v2/internal/tui/state"
	"github.com/Yat-Muk/prism-v2/internal/tui/types"
	tea "github.com/charmbracelet/bubbletea"
)

// KeyHandler 核心處理器：負責全局導航和請求分發
type KeyHandler struct {
	stateMgr    *state.Manager
	cmdBuilder  *CommandBuilder
	certHandler *CertHandler
}

func NewKeyHandler(
	stateMgr *state.Manager,
	cmdBuilder *CommandBuilder,
	certHandler *CertHandler,
) *KeyHandler {
	return &KeyHandler{
		stateMgr:    stateMgr,
		cmdBuilder:  cmdBuilder,
		certHandler: certHandler,
	}
}

// Handle 處理全局按鍵
func (h *KeyHandler) Handle(msg tea.KeyMsg, m *state.Manager) (*state.Manager, tea.Cmd) {
	if msg.Type == tea.KeyCtrlC {
		return m, tea.Quit
	}

	// 安裝進度頁的專用攔截邏輯
	if m.UI().CurrentView == state.InstallProgressView {
		if m.Install().IsFinished && msg.Type == tea.KeyEnter {
			return m, tea.Batch(
				m.UI().SwitchView(state.MainMenuView),
				h.cmdBuilder.UpdateDataCmd(m),
			)
		}
		return m, nil
	}

	currentView := m.UI().CurrentView

	// === 節點參數視圖專用攔截 ===
	if currentView == state.ProtocolLinksView && m.Node().SelectionMode == "params" {
		switch msg.String() {
		case "esc", "q":
			m.Node().SelectionMode = "links"
			m.UI().SetStatus(state.StatusInfo, "", "", false)
			return m, m.UI().SwitchView(state.ClientConfigView)

		default:
			var cmd tea.Cmd
			m.Node().Viewport, cmd = m.Node().Viewport.Update(msg)
			return m, cmd
		}
	}

	switch msg.Type {
	case tea.KeyEnter:
		return h.handleInputSubmit(m, currentView)

	case tea.KeyEsc:
		return h.handleInputEscape(m, currentView)

	default:
		// 所有輸入交給組件，UpdateInput 會返回閃爍計時器的 Cmd
		return m, m.UI().UpdateInput(msg)
	}
}

// ========================================
// 輔助函數
// ========================================

// markConfigChanged 標記配置已修改 (內存)
func (h *KeyHandler) markConfigChanged(m *state.Manager) {
	m.Config().HasUnsavedChanges = true
	m.Config().Dirty = true
}

func contains(slice []int, item int) bool {
	for _, v := range slice {
		if v == item {
			return true
		}
	}
	return false
}

// ========================================
// 核心分發邏輯 (Enter 觸發)
// ========================================

func (h *KeyHandler) handleInputSubmit(m *state.Manager, view state.View) (*state.Manager, tea.Cmd) {
	input := strings.TrimSpace(m.UI().GetInputBuffer())

	// 1. 攔截證書視圖，全權委派給 CertHandler
	if isCertView(view) {
		return h.certHandler.Handle(tea.KeyMsg{Type: tea.KeyEnter}, m)
	}

	// 2. 非證書視圖，執行標準清理邏輯
	m.UI().ClearInput()

	// 允許接收空輸入 (Enter鍵)
	if input == "" && view != state.InstallWizardView && view != state.StreamingCheckView {
		return m, nil
	}

	// 3. 根據視圖分發到具體的 submit 函數
	switch view {
	case state.MainMenuView:
		return h.submitMainMenu(m, input)

	// --- 安裝與配置 ---
	case state.InstallWizardView:
		return h.submitInstallWizard(m, input)
	case state.ConfigMenuView:
		return h.submitConfigMenu(m, input)
	case state.ProtocolMenuView:
		return h.submitProtocolMenu(m, input)
	case state.PortEditView:
		return h.submitPortEdit(m, input)
	case state.Hy2PortModeView:
		return h.submitHy2PortEdit(m, input)
	case state.SNIEditView:
		return h.submitSNIEdit(m, input)
	case state.UUIDEditView:
		return h.submitUUIDEdit(m, input)
	case state.AnyTLSPaddingView:
		return h.submitAnyTLSPadding(m, input)

	// --- 路由管理 ---
	case state.RouteMenuView:
		return h.submitRouteMenu(m, input)
	case state.OutboundMenuView:
		return h.submitOutboundMenu(m, input)
	case state.WARPRoutingView:
		return h.submitWARPRouting(m, input)
	case state.Socks5RoutingView:
		return h.submitSocks5Routing(m, input)
	case state.Socks5InboundView:
		return h.submitSocks5Inbound(m, input)
	case state.Socks5OutboundView:
		return h.submitSocks5Outbound(m, input)
	case state.IPv6RoutingView:
		return h.submitIPv6Routing(m, input)
	case state.DNSRoutingView:
		return h.submitDNSRouting(m, input)
	case state.SNIProxyRoutingView:
		return h.submitSNIProxyRouting(m, input)

	// --- 核心管理 ---
	case state.CoreMenuView:
		return h.submitCoreMenu(m, input)
	case state.CoreVersionSelectView:
		return h.submitCoreVersionSelect(m, input)
	case state.CoreSourceSelectView:
		return h.submitCoreSourceSelect(m, input)
	case state.ScriptUpdateView:
		return h.submitScriptUpdate(m, input)

	// --- 服務管理 ---
	case state.ServiceMenuView:
		return h.submitServiceMenu(m, input)
	case state.ServiceLogView:
		return m, nil
	case state.ServiceHealthView:
		return m, nil

	// --- 工具箱 ---
	case state.ToolsMenuView:
		return h.submitToolsMenu(m, input)
	case state.SwapMenuView:
		return h.submitSwapMenu(m, input)
	case state.Fail2BanMenuView:
		return h.submitFail2BanMenu(m, input)
	case state.BBRMenuView:
		return h.submitBBRMenu(m, input)
	case state.StreamingCheckView:
		return h.submitStreamingCheck(m, input)
	case state.SystemCleanupView:
		return h.submitSystemCleanup(m, input)
	case state.BackupMenuView, state.BackupRestoreMenuView:
		return h.submitBackupRestoreMenu(m, input)

	// --- 日誌管理 ---
	case state.LogMenuView:
		return h.submitLogMenu(m, input)
	case state.LogViewerView:
		if m.Tools().IsViewingFail2BanLogs {
			m.Tools().IsViewingFail2BanLogs = false
			return m, m.UI().SwitchView(state.Fail2BanMenuView)
		}
		return m, m.UI().SwitchView(state.LogMenuView)
	case state.LogLevelEditView:
		return h.submitLogLevelEdit(m, input)

	// --- 節點信息 ---
	case state.NodeInfoView:
		return h.submitNodeInfo(m, input)
	case state.ProtocolLinksView:
		return h.submitProtocolLinks(m, input)
	case state.SubscriptionView:
		return h.submitSubscription(m, input)
	case state.QRCodeView:
		return m, nil
	case state.ClientConfigView:
		return h.submitClientConfig(m, input)

	// --- 卸載 ---
	case state.UninstallView:
		return h.submitUninstall(m, input)
	case state.UninstallProgressView:
		return m, nil
	}

	return m, nil
}

func isCertView(v state.View) bool {
	switch v {
	case state.CertMenuView,
		state.ACMEHTTPInputView,
		state.ACMEDNSProviderView,
		state.DNSCredentialInputView,
		state.ProviderSwitchView,
		state.CertRenewView,
		state.CertDeleteView,
		state.CertStatusView,
		state.CertModeMenuView:
		return true
	}
	return false
}

// ========================================
// 具體業務邏輯 (Submits)
// ========================================

func (h *KeyHandler) submitMainMenu(m *state.Manager, input string) (*state.Manager, tea.Cmd) {
	switch strings.ToLower(input) {
	case constants.KeyMain_InstallWizard:
		m.Install().SelectAll()
		return m, m.UI().SwitchView(state.InstallWizardView)

	case constants.KeyMain_ServiceStart:
		m.UI().SetStatus(state.StatusInfo, "正在重啟服務...", "", true)
		return m, h.cmdBuilder.RestartServiceCmd()

	case constants.KeyMain_ServiceStop:
		m.UI().SetStatus(state.StatusInfo, "正在停止服務...", "", true)
		return m, h.cmdBuilder.StopServiceCmd()

	case constants.KeyMain_Config:
		return m, m.UI().SwitchView(state.ConfigMenuView)

	case constants.KeyMain_Cert:
		cmd1 := m.UI().SwitchView(state.CertMenuView)
		cmd2 := h.cmdBuilder.RefreshCertListCmd()
		return m, tea.Batch(cmd1, cmd2)

	case constants.KeyMain_Outbound:
		return m, m.UI().SwitchView(state.OutboundMenuView)

	case constants.KeyMain_Route:
		return m, m.UI().SwitchView(state.RouteMenuView)

	case constants.KeyMain_Core:
		return m, m.UI().SwitchView(state.CoreMenuView)

	case constants.KeyMain_Tools:
		return m, m.UI().SwitchView(state.ToolsMenuView)

	case constants.KeyMain_Log:
		return m, tea.Batch(
			m.UI().SwitchView(state.LogMenuView),
			h.cmdBuilder.LoadLogInfoCmd(m),
		)

	case constants.KeyMain_NodeInfo:
		cmd1 := m.UI().SwitchView(state.NodeInfoView)
		cmd2 := h.cmdBuilder.GenerateProtocolLinksCmd(m)
		return m, tea.Batch(cmd1, cmd2)

	case constants.KeyMain_Uninstall:
		m.Uninstall().Reset()
		cmd1 := m.UI().SwitchView(state.UninstallView)
		cmd2 := h.cmdBuilder.ScanUninstallInfoCmd(m)
		return m, tea.Batch(cmd1, cmd2)

	case constants.KeyMain_Quit:
		return m, tea.Quit

	default:
		m.UI().SetStatus(state.StatusError, "無效選擇", "", false)
	}
	return m, nil
}

// --- 安裝與配置 ---

// submitInstallWizard 安裝嚮導提交邏輯
func (h *KeyHandler) submitInstallWizard(m *state.Manager, input string) (*state.Manager, tea.Cmd) {
	if input != "" {
		// 執行切換邏輯 (利用現有的輔助函數)
		toggleProtocolsFromInput(m.Install(), input)

		// 關鍵修復：這裡返回 nil，讓界面停留在安裝嚮導，方便用戶繼續選擇其他協議
		m.UI().SetStatus(state.StatusInfo, "已更新選擇，請繼續輸入或按回車確認", "", true)
		return m, nil
	}

	// 1. 防呆檢查：必須至少選一個
	if m.Install().IsEmpty() {
		m.UI().SetStatus(state.StatusWarn, "請輸入序號選擇至少一個協議", "", false)
		return m, nil
	}

	// 2. 鎖定選擇，同步到全局配置 (DTO 映射)
	selected := m.Install().InstallProtocols
	m.Config().EnabledProtocols = selected

	p := &m.Config().Config.Protocols
	p.RealityVision.Enabled = contains(selected, 1)
	p.RealityGRPC.Enabled = contains(selected, 2)
	p.Hysteria2.Enabled = contains(selected, 3)
	p.TUIC.Enabled = contains(selected, 4)
	p.AnyTLS.Enabled = contains(selected, 5)
	p.AnyTLSReality.Enabled = contains(selected, 6)
	p.ShadowTLS.Enabled = contains(selected, 7)

	// 3. 設置狀態並開始全自動流程
	m.Install().ResetLogs()
	m.Install().AddLog("正在初始化安裝環境...")
	m.Install().AddLog("正在計算隨機端口與密鑰...")

	return m, tea.Batch(
		m.UI().SwitchView(state.InstallProgressView),
		h.cmdBuilder.ResetPortsCmd(m),
	)
}

// submitConfigMenu 配置菜單核心邏輯
func (h *KeyHandler) submitConfigMenu(m *state.Manager, input string) (*state.Manager, tea.Cmd) {
	cfgState := m.Config()

	// 1. 退出確認模式 (ExitConfirmMode - 按 Esc 後觸發)
	if cfgState.ExitConfirmMode {
		if strings.EqualFold(input, "Y") {
			// [應用] 保存 + 重啟 + 退出
			cfgState.ExitConfirmMode = false
			cfgState.HasUnsavedChanges = false
			m.UI().SetStatus(state.StatusInfo, "正在應用配置並重啟服務...", "", true)

			return m, tea.Batch(
				h.cmdBuilder.SaveConfigCmd(m),
				h.cmdBuilder.RestartServiceCmd(),
				m.UI().SwitchView(state.MainMenuView),
			)
		} else if strings.EqualFold(input, "N") {
			cfgState.ExitConfirmMode = false
			cfgState.HasUnsavedChanges = false
			m.UI().SetStatus(state.StatusInfo, "已放棄修改", "", false)

			return m, tea.Batch(
				m.UI().SwitchView(state.MainMenuView),
				h.cmdBuilder.LoadConfigCmd(m),
			)
		} else {
			// [取消退出]
			cfgState.ExitConfirmMode = false
			m.UI().SetStatus(state.StatusInfo, "已取消退出，請繼續編輯", "", false)
			return m, nil
		}
	}

	// 2. 重置確認模式 (ConfirmMode - 按 r 後觸發)
	if cfgState.ConfirmMode {
		if strings.EqualFold(input, "YES") {
			cfgState.ConfirmMode = false
			// 內存重置並標記修改
			m.Config().UpdateConfig(config.DefaultConfig())
			h.markConfigChanged(m)
			m.UI().SetStatus(state.StatusSuccess, "已重置為默認 (未保存)", "請按 Esc 退出並確認應用", false)
			return m, nil
		}
		cfgState.ConfirmMode = false
		m.UI().SetStatus(state.StatusInfo, "已取消重置", "", false)
		return m, nil
	}

	// 3. 普通菜單選項
	switch strings.ToLower(input) {
	case constants.KeyConfig_Protocol:
		return m, m.UI().SwitchView(state.ProtocolMenuView)
	case constants.KeyConfig_SNI:
		return m, m.UI().SwitchView(state.SNIEditView)
	case constants.KeyConfig_UUID:
		return m, m.UI().SwitchView(state.UUIDEditView)
	case constants.KeyConfig_Port:
		return m, m.UI().SwitchView(state.PortEditView)
	case constants.KeyConfig_Padding:
		return m, m.UI().SwitchView(state.AnyTLSPaddingView)

	case constants.KeyConfig_Reset: // "r"
		cfgState.ConfirmMode = true
		// 狀態提示持久顯示
		m.UI().SetStatus(state.StatusWarn, "⚠️  確定重置所有配置? 輸入 YES 確認", "", false)

	default:
		m.UI().SetStatus(state.StatusError, "無效選項", "", false)
	}
	return m, nil
}

func (h *KeyHandler) submitProtocolMenu(m *state.Manager, input string) (*state.Manager, tea.Cmd) {
	newEnabledIDs := toggleIntList(m.Config().EnabledProtocols, input)
	m.Config().EnabledProtocols = newEnabledIDs

	// 同步到內存 Config
	p := &m.Config().Config.Protocols
	p.RealityVision.Enabled = contains(newEnabledIDs, 1)
	p.RealityGRPC.Enabled = contains(newEnabledIDs, 2)
	p.Hysteria2.Enabled = contains(newEnabledIDs, 3)
	p.TUIC.Enabled = contains(newEnabledIDs, 4)
	p.AnyTLS.Enabled = contains(newEnabledIDs, 5)
	p.AnyTLSReality.Enabled = contains(newEnabledIDs, 6)
	p.ShadowTLS.Enabled = contains(newEnabledIDs, 7)

	h.markConfigChanged(m)
	m.UI().SetStatus(state.StatusInfo, "協議狀態已更新 (未保存)", "", false)
	return m, nil
}

func (h *KeyHandler) submitPortEdit(m *state.Manager, input string) (*state.Manager, tea.Cmd) {
	// 1. 如果處於編輯模式 (正在輸入端口值)
	if m.Port().PortEditingMode {
		protoID := m.Port().PortEditingProtocol
		m.Port().CancelPortEdit()

		h.markConfigChanged(m)
		m.UI().SetStatus(state.StatusInfo, "端口已更新 (未保存)", "", false)

		return m, h.cmdBuilder.UpdateSinglePortCmd(m, protoID, input)
	}

	// 2. 處理 "r" 重置
	if strings.EqualFold(input, constants.KeyPort_Reset) {
		m.Config().ConfirmMode = true
		m.UI().SetStatus(state.StatusWarn, "⚠️ 確認隨機重置所有端口？(Y/N)", "", true)
		return m, nil
	}

	// 3. 處理 "r" 確認邏輯
	if m.Config().ConfirmMode {
		if strings.EqualFold(input, "y") {
			m.Config().ConfirmMode = false
			m.UI().SetStatus(state.StatusInfo, "已重置所有端口 (未保存)", "", true)
			return m, h.cmdBuilder.ResetPortsCmd(m)
		} else {
			m.Config().ConfirmMode = false
			m.UI().SetStatus(state.StatusInfo, "已取消重置", "", false)
			return m, nil
		}
	}

	// 4. 處理選擇協議
	if inputNum, err := strconv.Atoi(input); err == nil {
		// 這是將"視覺序號"翻譯回"真實ID"的唯一橋梁
		enabledProtos := m.Config().EnabledProtocols
		sortedProtos := make([]int, len(enabledProtos))
		copy(sortedProtos, enabledProtos)
		sort.Ints(sortedProtos) // 必須排序，因為 UI 也是排過序的

		// 計算數組下標 (用戶輸入 1 -> index 0)
		index := inputNum - 1

		// 驗證序號是否有效
		if index < 0 || index >= len(sortedProtos) {
			m.UI().SetStatus(state.StatusError, "無效的序號，請輸入列表中的數字", "", false)
			return m, nil
		}

		// [獲取真實 ID] 這才是我們要操作的對象
		realProtoID := sortedProtos[index]

		// 特殊處理 Hysteria 2 (ID = 3) -> 進入二級菜單
		if realProtoID == 3 {
			return m, m.UI().SwitchView(state.Hy2PortModeView)
		}

		// 普通協議，直接進入端口編輯
		m.Port().StartPortEdit(realProtoID)

		// 獲取當前端口
		currentPort := m.Port().CurrentPorts[realProtoID]

		// 獲取協議名稱 (為了更好的用戶體驗，建議顯示名稱)
		m.UI().SetStatus(state.StatusInfo, fmt.Sprintf("請輸入新端口 (當前: %d)", currentPort), "支持隨機: random", true)
		return m, nil
	}

	return m, nil
}

func (h *KeyHandler) submitHy2PortEdit(m *state.Manager, input string) (*state.Manager, tea.Cmd) {
	if m.Port().PortEditingMode {
		if m.Port().Hy2EditingHopping {
			// 1. 調用 validator 包進行解析
			start, end, err := inputvalidator.ParsePortRange(input)

			if err != nil {
				m.UI().SetStatus(state.StatusError, err.Error(), "示例: 20000-30000", false)
				return m, nil
			}

			// 2. 業務執行
			m.Port().CancelPortEdit()
			// 記錄到 state (可選)
			m.Port().Hy2HoppingRange = input
			h.markConfigChanged(m)
			m.UI().SetStatus(state.StatusInfo, fmt.Sprintf("跳躍端口已設置為 %d-%d (未保存)", start, end), "", true)

			return m, h.cmdBuilder.UpdateHy2HoppingCmd(m, start, end)

		} else {
			// 提交主端口 (ID 3)
			m.Port().CancelPortEdit()
			h.markConfigChanged(m)
			m.UI().SetStatus(state.StatusInfo, "主端口已更新 (未保存)", input, true)
			return m, h.cmdBuilder.UpdateSinglePortCmd(m, 3, input)
		}
	}

	// 2. 菜單選擇模式
	switch input {
	case constants.KeyPort_Main: // "1"
		m.Port().StartPortEdit(3)
		currentPort := m.Port().CurrentPorts[3]
		m.UI().SetStatus(state.StatusInfo, fmt.Sprintf("請輸入 Hysteria 2 主端口 (當前: %d)", currentPort), "", true)
		return m, nil

	case constants.KeyPort_Hopping: // "2"
		m.Port().StartHy2HoppingEdit()
		currentRange := m.Port().Hy2HoppingRange
		if currentRange == "" {
			currentRange = "未設置"
		}
		m.UI().SetStatus(state.StatusInfo, fmt.Sprintf("請輸入端口跳躍範圍 (當前: %s)", currentRange), "示例: 20000-30000", true)
		return m, nil

	case constants.KeyPort_ClearHopping: // "3"
		m.UI().SetStatus(state.StatusInfo, "已清除端口跳躍 (未保存)", "", true)
		return m, h.cmdBuilder.ClearHy2HoppingCmd(m)
	}
	return m, nil
}

func (h *KeyHandler) submitSNIEdit(m *state.Manager, input string) (*state.Manager, tea.Cmd) {
	if inputvalidator.ValidateDomainInput(input) == nil {
		p := &m.Config().Config.Protocols
		p.RealityVision.SNI = input
		p.RealityGRPC.SNI = input
		p.AnyTLSReality.SNI = input
		p.ShadowTLS.SNI = input

		h.markConfigChanged(m)
		m.UI().SetStatus(state.StatusInfo, "SNI 已更新 (未保存)", "", false)
		return m, nil
	}
	m.UI().SetStatus(state.StatusError, "無效的域名", "", false)
	return m, nil
}

func (h *KeyHandler) submitUUIDEdit(m *state.Manager, input string) (*state.Manager, tea.Cmd) {
	// 分支 1: 自動生成
	if input == "1" {
		m.UI().SetStatus(state.StatusInfo, "正在生成 UUID...", "", true)
		// 委託給 CommandBuilder，Handler 保持純粹
		return m, h.cmdBuilder.GenerateUUIDCmd()
	}

	// 分支 2: 手動輸入
	if len(input) == 36 {
		m.Config().Config.UUID = input
		h.markConfigChanged(m) // 標記未保存
		m.UI().SetStatus(state.StatusInfo, "UUID 已更新 (未保存)", input, false)
		return m, nil
	}

	m.UI().SetStatus(state.StatusError, "格式錯誤", "請輸入 36 位 UUID 或選擇選項 1", false)
	return m, nil
}

func (h *KeyHandler) submitAnyTLSPadding(m *state.Manager, input string) (*state.Manager, tea.Cmd) {
	if n, err := strconv.Atoi(input); err == nil && n >= 1 && n <= 5 {
		modes := []string{"balanced", "minimal", "high_resist", "video", "official"}
		m.Config().Config.Protocols.AnyTLS.PaddingMode = modes[n-1]
		m.Config().Config.Protocols.AnyTLSReality.PaddingMode = modes[n-1]
		h.markConfigChanged(m)
		m.UI().SetStatus(state.StatusInfo, "Padding 策略已更新 (未保存)", "", false)
		return m, nil
	}
	return m, nil
}

// --- 出口策略 ---

func (h *KeyHandler) submitOutboundMenu(m *state.Manager, input string) (*state.Manager, tea.Cmd) {
	// 1. 獲取當前網絡狀態 (防禦性檢查)
	hasIPv4 := false
	hasIPv6 := false
	if m.System() != nil && m.System().Stats != nil {
		hasIPv4 = m.System().Stats.IPv4 != ""
		hasIPv6 = m.System().Stats.IPv6 != ""
	}

	strategy := ""
	switch input {
	case constants.KeyOutbound_PreferIPv4: // "1"
		if !hasIPv4 {
			m.UI().SetStatus(state.StatusError, "操作拒絕: 當前環境未檢測到 IPv4", "", false)
			return m, nil
		}
		strategy = "prefer_ipv4"

	case constants.KeyOutbound_PreferIPv6: // "2"
		if !hasIPv6 {
			m.UI().SetStatus(state.StatusError, "操作拒絕: 當前環境未檢測到 IPv6", "", false)
			return m, nil
		}
		strategy = "prefer_ipv6"

	case constants.KeyOutbound_IPv4Only: // "3"
		if !hasIPv4 {
			m.UI().SetStatus(state.StatusError, "操作拒絕: 當前環境未檢測到 IPv4", "", false)
			return m, nil
		}
		strategy = "ipv4_only"

	case constants.KeyOutbound_IPv6Only: // "4"
		if !hasIPv6 {
			m.UI().SetStatus(state.StatusError, "操作拒絕: 當前環境未檢測到 IPv6", "", false)
			return m, nil
		}
		strategy = "ipv6_only"
	}

	if strategy != "" {
		// 設置狀態提示
		m.UI().SetStatus(state.StatusInfo, "出站策略已更新 (未保存)", "請按 Esc 退出配置菜單以應用更改", true)
		return m, h.cmdBuilder.UpdateOutboundStrategyCmd(m, strategy)
	}

	return m, nil
}

// --- 路由管理 ---

func (h *KeyHandler) submitRouteMenu(m *state.Manager, input string) (*state.Manager, tea.Cmd) {
	switch input {
	case constants.KeyRoute_WARP:
		cmd1 := m.UI().SwitchView(state.WARPRoutingView)
		cmd2 := h.cmdBuilder.LoadWARPConfigCmd(m)
		return m, tea.Batch(cmd1, cmd2)

	case constants.KeyRoute_Socks5:
		cmd1 := m.UI().SwitchView(state.Socks5RoutingView)
		cmd2 := h.cmdBuilder.LoadSocks5ConfigCmd(m)
		return m, tea.Batch(cmd1, cmd2)

	case constants.KeyRoute_IPv6:
		cmd1 := m.UI().SwitchView(state.IPv6RoutingView)
		cmd2 := h.cmdBuilder.LoadIPv6ConfigCmd(m)
		return m, tea.Batch(cmd1, cmd2)

	case constants.KeyRoute_DNS:
		cmd1 := m.UI().SwitchView(state.DNSRoutingView)
		cmd2 := h.cmdBuilder.LoadDNSConfigCmd(m)
		return m, tea.Batch(cmd1, cmd2)

	case constants.KeyRoute_SNIProxy:
		cmd1 := m.UI().SwitchView(state.SNIProxyRoutingView)
		cmd2 := h.cmdBuilder.LoadSNIProxyConfigCmd(m)
		return m, tea.Batch(cmd1, cmd2)
	}
	return m, nil
}

func (h *KeyHandler) submitWARPRouting(m *state.Manager, input string) (*state.Manager, tea.Cmd) {
	// 1. 編輯模式捕獲
	if m.Routing().IsEditing() {
		field := m.Routing().EditingField
		m.Routing().StopEditing()

		val := input
		m.UI().ClearInput()

		if field == "license" {
			return m, h.cmdBuilder.UpdateWARPLicenseCmd(m, val)
		}

		if field == "domains" && val != "" {
			return m, h.cmdBuilder.AddRoutingDomainCmd(m, "warp", val)
		}

		m.UI().SetStatus(state.StatusInfo, "輸入已取消", "", false)
		return m, nil
	}

	switch input {
	case constants.KeyWARP_ToggleIPv4:
		// 統一開啟
		return m, h.cmdBuilder.SetWARPStateCmd(m, true)

	case constants.KeyWARP_ToggleIPv6:
		// 統一開啟 (WARP 隧道是單例的)
		return m, h.cmdBuilder.SetWARPStateCmd(m, true)

	case constants.KeyWARP_SetGlobal:
		// 獲取當前狀態並取反
		cfg := m.Config().GetConfig()
		if cfg != nil {
			currentGlobal := cfg.Routing.WARP.Global
			return m, h.cmdBuilder.SetWARPGlobalCmd(m, !currentGlobal)
		}
		return m, nil

	case constants.KeyWARP_SetLicense:
		m.Routing().StartEditing("warp", "license")
		m.UI().SetStatus(state.StatusInfo, "請輸入 WARP+ 密鑰", "留空則使用免費版 (按 Enter 確認)", true)
		return m, nil

	case constants.KeyWARP_SetDomains:
		m.Routing().StartEditing("warp", "domains")
		m.UI().SetStatus(state.StatusInfo, "請輸入分流域名", "例如: chatgpt.com (按 Enter 確認)", true)
		return m, nil

	case constants.KeyWARP_ShowConfig:
		return m, h.cmdBuilder.ShowWARPConfigCmd(m)

	case constants.KeyWARP_Disable:
		return m, h.cmdBuilder.SetWARPStateCmd(m, false)
	}
	return m, nil
}

func (h *KeyHandler) submitSocks5Routing(m *state.Manager, input string) (*state.Manager, tea.Cmd) {
	switch input {
	case constants.KeySocks5_Inbound:
		return m, m.UI().SwitchView(state.Socks5InboundView)
	case constants.KeySocks5_Outbound:
		return m, m.UI().SwitchView(state.Socks5OutboundView)
	case constants.KeySocks5_ShowConfig:
		return m, h.cmdBuilder.ShowSocks5ConfigCmd(m)
	case constants.KeySocks5_Uninstall:
		m.UI().SetStatus(state.StatusWarn, "確認卸載? 輸入 YES", "", true)
	}
	return m, nil
}

func (h *KeyHandler) submitSocks5Inbound(m *state.Manager, input string) (*state.Manager, tea.Cmd) {
	if m.Routing().IsEditing() {
		// 先暫存字段
		field := m.Routing().EditingField
		m.Routing().StopEditing()

		val := input
		m.UI().ClearInput()

		switch field {
		case "port":
			return m, h.cmdBuilder.UpdateSocks5InboundPortCmd(m, val)
		case "auth":
			return m, h.cmdBuilder.UpdateSocks5InboundAuthCmd(m, val)
		}

		m.UI().SetStatus(state.StatusInfo, "輸入已取消", "", false)
		return m, nil
	}

	switch input {
	case constants.KeySocks5In_Toggle:
		return m, h.cmdBuilder.ToggleSocks5InboundCmd(m)
	case constants.KeySocks5In_Port:
		m.Routing().StartEditing("socks5_inbound", "port")
		m.UI().SetStatus(state.StatusInfo, "請輸入端口號", "例如: 1080", true)
		return m, nil
	case constants.KeySocks5In_Auth:
		m.Routing().StartEditing("socks5_inbound", "auth")
		m.UI().SetStatus(state.StatusInfo, "請輸入認證信息", "格式 user:pass (留空清除)", true)
		return m, nil
	}
	return m, nil
}

func (h *KeyHandler) submitSocks5Outbound(m *state.Manager, input string) (*state.Manager, tea.Cmd) {
	if m.Routing().IsEditing() {
		// 先暫存字段
		field := m.Routing().EditingField
		m.Routing().StopEditing()

		val := input
		m.UI().ClearInput()

		switch field {
		case "server":
			return m, h.cmdBuilder.UpdateSocks5OutboundServerCmd(m, val)
		case "auth":
			return m, h.cmdBuilder.UpdateSocks5OutboundAuthCmd(m, val)
		}
		return m, nil
	}

	switch input {
	case constants.KeySocks5Out_Toggle:
		return m, h.cmdBuilder.ToggleSocks5OutboundCmd(m)
	case constants.KeySocks5Out_Server:
		m.Routing().StartEditing("socks5_outbound", "server")
		m.UI().SetStatus(state.StatusInfo, "請輸入落地機地址", "格式 IP:Port", true)
		return m, nil
	case constants.KeySocks5Out_Auth:
		m.Routing().StartEditing("socks5_outbound", "auth")
		m.UI().SetStatus(state.StatusInfo, "請輸入認證信息", "格式 User:Pass", true)
		return m, nil
	case constants.KeySocks5Out_Global:
		return m, h.cmdBuilder.SetSocks5GlobalCmd(m, true)
	}
	return m, nil
}

func (h *KeyHandler) submitIPv6Routing(m *state.Manager, input string) (*state.Manager, tea.Cmd) {
	if m.Routing().IsEditing() {
		// 先暫存字段
		field := m.Routing().EditingField
		m.Routing().StopEditing()

		val := input
		m.UI().ClearInput()

		if field == "domains" && val != "" {
			return m, h.cmdBuilder.AddRoutingDomainCmd(m, "ipv6", val)
		}
		return m, nil
	}

	switch input {
	case constants.KeyIPv6Split_Enable:
		return m, h.cmdBuilder.ToggleIPv6RoutingCmd(m, true)
	case constants.KeyIPv6Split_Disable:
		return m, h.cmdBuilder.ToggleIPv6RoutingCmd(m, false)
	case constants.KeyIPv6Split_SetGlobal:
		return m, h.cmdBuilder.SetIPv6GlobalCmd(m, true)
	case constants.KeyIPv6Split_SetDomain:
		m.Routing().StartEditing("ipv6", "domains")
		m.UI().SetStatus(state.StatusInfo, "輸入域名 (例如 netflix.com)", "", true)
		return m, nil
	}
	return m, nil
}

func (h *KeyHandler) submitDNSRouting(m *state.Manager, input string) (*state.Manager, tea.Cmd) {
	if m.Routing().IsEditing() {
		// 先暫存字段
		field := m.Routing().EditingField
		m.Routing().StopEditing()

		val := input
		m.UI().ClearInput()

		if field == "server" {
			return m, h.cmdBuilder.EnableRoutingWithTargetCmd(m, "dns", val)
		} else if field == "domains" {
			return m, h.cmdBuilder.AddRoutingDomainCmd(m, "dns", val)
		}
		return m, nil
	}

	switch input {
	case constants.KeyRouting_Enable:
		m.Routing().StartEditing("dns", "server")
		m.UI().SetStatus(state.StatusInfo, "請輸入 DNS 服務器 IP", "例如: 8.8.8.8", true)
		return m, nil
	case constants.KeyRouting_Disable:
		return m, h.cmdBuilder.DisableDNSRoutingCmd(m)
	case constants.KeyRouting_AddDomain:
		m.Routing().StartEditing("dns", "domains")
		m.UI().SetStatus(state.StatusInfo, "輸入分流域名", "", true)
		return m, nil
	}
	return m, nil
}

func (h *KeyHandler) submitSNIProxyRouting(m *state.Manager, input string) (*state.Manager, tea.Cmd) {
	if m.Routing().IsEditing() {
		// 先暫存字段
		field := m.Routing().EditingField
		m.Routing().StopEditing()

		val := input
		m.UI().ClearInput()

		if field == "target_ip" {
			return m, h.cmdBuilder.EnableRoutingWithTargetCmd(m, "sni", val)
		} else if field == "domains" {
			return m, h.cmdBuilder.AddRoutingDomainCmd(m, "sni", val)
		}
		return m, nil
	}

	switch input {
	case constants.KeyRouting_Enable:
		m.Routing().StartEditing("sni", "target_ip")
		m.UI().SetStatus(state.StatusInfo, "請輸入 SNI 代理目標 IP", "例如: 1.1.1.1", true)
		return m, nil
	case constants.KeyRouting_Disable:
		return m, h.cmdBuilder.DisableSNIProxyCmd(m)
	case constants.KeyRouting_AddDomain:
		m.Routing().StartEditing("sni", "domains")
		m.UI().SetStatus(state.StatusInfo, "請輸入分流域名", "", true)
		return m, nil
	case constants.KeyRouting_Show:
		return m, h.cmdBuilder.ShowSNIProxyRulesCmd(m)
	}
	return m, nil
}

// --- 核心管理 ---

func (h *KeyHandler) submitCoreMenu(m *state.Manager, input string) (*state.Manager, tea.Cmd) {
	if m.Core().IsInstalled {
		switch input {
		case constants.KeyCore_CheckUpdate:
			return m, h.cmdBuilder.CheckCoreUpdateCmd(m, false)

		case constants.KeyCore_Update:
			core := m.Core()
			if core.LatestVersion != "" && core.CoreVersion != "unknown" {
				if core.CoreVersion == core.LatestVersion {
					m.UI().SetStatus(state.StatusInfo, fmt.Sprintf("當前已是最新版本 (%s)", core.CoreVersion), "無需更新", false)
					return m, nil
				}
			}

			m.UI().SetStatus(state.StatusInfo, "正在獲取最新版本並更新...", "下載可能需要幾分鐘，請稍候", true)

			return m, h.cmdBuilder.UpdateCoreCmd(m, "")

		case constants.KeyCore_Reinstall:
			m.UI().SetStatus(state.StatusInfo, "正在重新安裝當前版本...", "請稍候", true)
			return m, h.cmdBuilder.ReinstallCoreCmd(m)

		case constants.KeyCore_SelectVersion:
			cmd1 := m.UI().SwitchView(state.CoreVersionSelectView)
			cmd2 := h.cmdBuilder.LoadCoreVersionsCmd(m)
			return m, tea.Batch(cmd1, cmd2)

		case constants.KeyCore_SelectSource:
			return m, m.UI().SwitchView(state.CoreSourceSelectView)

		case constants.KeyScript_Update:
			m.Core().IsCheckingScript = true
			cmd1 := m.UI().SwitchView(state.ScriptUpdateView)
			cmd2 := h.cmdBuilder.CheckScriptUpdateCmd(m)
			return m, tea.Batch(cmd1, cmd2)

		case constants.KeyCore_Uninstall:
			m.UI().SetStatus(state.StatusWarn, "正在卸載核心及服務...", "請稍候", true)
			return m, h.cmdBuilder.UninstallCoreCmd(m)
		}
	} else {
		// 未安裝狀態下的菜單
		switch input {
		case constants.KeyCore_InstallLatest: // 對應 View 中的 "1"
			return m, h.cmdBuilder.InstallCoreCmdFull("latest", "")

		case constants.KeyCore_SelectVersion: // 對應 View 中的 "2"
			cmd1 := m.UI().SwitchView(state.CoreVersionSelectView)
			cmd2 := h.cmdBuilder.LoadCoreVersionsCmd(m)
			return m, tea.Batch(cmd1, cmd2)

		case constants.KeyCore_InstallDev: // 對應 View 中的 "3"
			// 安裝開發版邏輯
			m.UI().SetStatus(state.StatusInfo, "開發版安裝功能暫未開放", "", false)
			return m, nil
		}
	}
	return m, nil
}

// [新增] 處理腳本更新確認界面的按鍵
func (h *KeyHandler) submitScriptUpdate(m *state.Manager, input string) (*state.Manager, tea.Cmd) {
	switch input {
	case constants.KeyScriptUpdate_Confirm: // "1"
		// 只有檢測到新版本才允許更新
		if m.Core().ScriptLatestVersion == "" {
			return m, nil
		}
		m.UI().SetStatus(state.StatusInfo, "正在下載並更新腳本...", "更新完成後程序將重啟", true)
		return m, h.cmdBuilder.UpdateScriptExecCmd(m)

	case constants.KeyScriptUpdate_Cancel: // "2"
		// 取消並返回
		m.Core().IsCheckingScript = false
		return m, m.UI().SwitchView(state.CoreMenuView)
	}
	return m, nil
}

func (h *KeyHandler) submitCoreVersionSelect(m *state.Manager, input string) (*state.Manager, tea.Cmd) {
	if input == "" {
		return m, nil
	}

	versions := m.Core().AvailableVers
	targetVersion := ""

	// 1. 嘗試解析為列表序號 (1-9)
	if idx, err := strconv.Atoi(input); err == nil {
		if idx >= 1 && idx <= len(versions) {
			targetVersion = versions[idx-1]
		} else {
			// 用戶輸入了數字但不在列表範圍內，視為手動輸入版本號
			targetVersion = input
		}
	} else {
		// 2. 解析失敗，說明輸入包含非數字字符 (如 "v1.11.0")
		targetVersion = input
	}

	// 3. 執行安裝
	if targetVersion != "" {
		m.UI().SetStatus(state.StatusInfo, "正在安裝核心: "+targetVersion, "請稍候...", true)

		return m, tea.Batch(
			m.UI().SwitchView(state.CoreMenuView),
			h.cmdBuilder.InstallCoreCmdFull(targetVersion, m.Core().CoreVersion),
		)
	}

	return m, nil
}

func (h *KeyHandler) submitCoreSourceSelect(m *state.Manager, input string) (*state.Manager, tea.Cmd) {

	// 假設用戶輸入 "1" 或 "2"
	switch input {
	case "1":
		return m, tea.Batch(
			h.cmdBuilder.SetCoreSourceCmd(m, "github"),
			m.UI().SwitchView(state.CoreMenuView),
		)
	case "2":
		return m, tea.Batch(
			h.cmdBuilder.SetCoreSourceCmd(m, "ghproxy"),
			m.UI().SwitchView(state.CoreMenuView),
		)
	}
	return m, nil
}

// --- 服務管理 ---

func (h *KeyHandler) submitServiceMenu(m *state.Manager, input string) (*state.Manager, tea.Cmd) {
	switch input {
	case constants.KeyService_Restart:
		return m, h.cmdBuilder.RestartServiceCmd()
	case constants.KeyService_Stop:
		return m, h.cmdBuilder.StopServiceCmd()
	case constants.KeyService_Log:
		cmd1 := m.UI().SwitchView(state.ServiceLogView)
		cmd2 := h.cmdBuilder.FollowServiceLogCmd(m)
		return m, tea.Batch(cmd1, cmd2)
	case constants.KeyService_Refresh:
		return m, h.cmdBuilder.UpdateDataCmd(m)
	case constants.KeyService_AutoStart:
		return m, h.cmdBuilder.ToggleAutoStartCmd(m)
	case constants.KeyService_Health:
		cmd1 := m.UI().SwitchView(state.ServiceHealthView)
		cmd2 := h.cmdBuilder.ServiceHealthCheckCmd(m)
		return m, tea.Batch(cmd1, cmd2)
	}
	return m, nil
}

// --- 工具箱 ---

func (h *KeyHandler) submitToolsMenu(m *state.Manager, input string) (*state.Manager, tea.Cmd) {
	switch input {
	case constants.KeyTools_Streaming:
		// 進入流媒體檢測
		m.Tools().StreamingResult = nil
		m.Tools().IsCheckingStreaming = true

		return m, tea.Batch(
			m.UI().SwitchView(state.StreamingCheckView),
			h.cmdBuilder.CheckStreamingCmd(m, true),
		)

	case constants.KeyTools_Swap:
		return m, m.UI().SwitchView(state.SwapMenuView)

	case constants.KeyTools_Fail2Ban:
		// 切換視圖並異步加載 Fail2Ban 信息
		return m, tea.Batch(
			m.UI().SwitchView(state.Fail2BanMenuView),
			h.cmdBuilder.LoadFail2BanInfoCmd(m),
		)

	case constants.KeyTools_TimeSync:
		m.UI().SetStatus(state.StatusInfo, "正在校準時間...", "", true)
		return m, h.cmdBuilder.TimeSyncCmd()

	case constants.KeyTools_BBR:
		return m, tea.Batch(
			m.UI().SwitchView(state.BBRMenuView),
			h.cmdBuilder.LoadBBRInfoCmd(m),
		)

	case constants.KeyTools_Cleanup:
		// 進入頁面並立即掃描
		cmd1 := m.UI().SwitchView(state.SystemCleanupView)
		cmd2 := h.cmdBuilder.ScanSystemCleanupCmd()
		return m, tea.Batch(cmd1, cmd2)

	case constants.KeyTools_Backup:
		// 跳轉到備份菜單
		return m, tea.Batch(
			m.UI().SwitchView(state.BackupMenuView),
			h.cmdBuilder.ListBackupsCmd(m, 5),
		)
	}
	return m, nil
}

func (h *KeyHandler) submitSwapMenu(m *state.Manager, input string) (*state.Manager, tea.Cmd) {
	switch input {
	case constants.KeySwap_Create: // "1"
		m.UI().SetStatus(state.StatusInfo, "正在創建 1GB Swap...", "請稍候", true)
		return m, h.cmdBuilder.CreateSwapCmd(m, 1)

	case constants.KeySwap_Custom: // "2"
		m.UI().SetStatus(state.StatusInfo, "正在創建 2GB Swap...", "請稍候", true)
		return m, h.cmdBuilder.CreateSwapCmd(m, 2)

	case constants.KeySwap_Delete: // "3"
		m.UI().SetStatus(state.StatusWarn, "正在刪除 Swap...", "", true)

		currentPath := ""
		if m.Tools().SwapInfo != nil {
			currentPath = m.Tools().SwapInfo.SwapFile
		}
		return m, h.cmdBuilder.DeleteSwapCmd(currentPath)

	}
	return m, nil
}

// submitFail2BanMenu 處理菜單提交
func (h *KeyHandler) submitFail2BanMenu(m *state.Manager, input string) (*state.Manager, tea.Cmd) {
	info := m.Tools().Fail2BanInfo
	if info == nil {
		info = &types.Fail2BanInfo{}
	}

	// 1. 處理解封 IP 輸入
	if m.Tools().Fail2BanInputMode {
		ip := strings.TrimSpace(input)
		m.Tools().Fail2BanInputMode = false
		m.UI().ClearInput()

		if ip == "" {
			m.UI().SetStatus(state.StatusInfo, "已取消", "", false)
			return m, nil
		}

		// [修復] 驗證 IP 格式
		if net.ParseIP(ip) == nil {
			m.UI().SetStatus(state.StatusError, "無效的 IP 地址格式", "", false)
			return m, nil
		}

		m.UI().SetStatus(state.StatusInfo, "正在解封...", ip, true)
		return m, h.cmdBuilder.UnbanIPCmd(ip)
	}

	// 2. 處理配置修改流程 (步驟 1: 輸入重試次數)
	if m.Tools().Fail2BanConfigStep == 1 {
		if _, err := strconv.Atoi(input); err != nil {
			m.UI().SetStatus(state.StatusError, "請輸入有效的數字", "", false)
			return m, nil
		}
		m.Tools().Fail2BanTempMaxRetry = input

		// 進入步驟 2
		m.Tools().Fail2BanConfigStep = 2
		m.UI().ClearInput()
		m.UI().SetStatus(state.StatusInfo, "請輸入封禁時長", "例如: 1h, 30m, 1d", true)
		return m, nil
	}

	// 3. 處理配置修改流程 (步驟 2: 輸入封禁時間 -> 提交)
	if m.Tools().Fail2BanConfigStep == 2 {
		if input == "" {
			input = "1h" // 默認值
		}
		m.Tools().Fail2BanConfigStep = 0 // 結束流程
		m.UI().ClearInput()
		m.UI().SetStatus(state.StatusInfo, "正在應用配置...", "", true)

		return m, h.cmdBuilder.UpdateFail2BanConfigCmd(m.Tools().Fail2BanTempMaxRetry, input)
	}

	// 4. 主菜單選項
	switch input {
	case constants.KeyFail2Ban_Install: // "1"
		if info.Installed {
			m.UI().SetStatus(state.StatusInfo, "已安裝", "", false)
			return m, nil
		}
		m.UI().SetStatus(state.StatusInfo, "正在安裝...", "可能需要幾分鐘", true)
		return m, h.cmdBuilder.InstallFail2BanCmd(m)

	case constants.KeyFail2Ban_Toggle: // "2"
		if !info.Installed {
			m.UI().SetStatus(state.StatusError, "請先安裝", "", false)
			return m, nil
		}
		status := "啟動"
		if info.Running {
			status = "停止"
		}
		m.UI().SetStatus(state.StatusInfo, fmt.Sprintf("正在%s服務...", status), "", true)
		return m, h.cmdBuilder.ToggleFail2BanCmd(info.Running)

	case constants.KeyFail2Ban_List: // "3"
		if !info.Running {
			m.UI().SetStatus(state.StatusError, "服務未運行", "", false)
			return m, nil
		}
		// [關鍵] 標記為 Fail2Ban 模式，以便 Esc 返回正確頁面
		m.Tools().IsViewingFail2BanLogs = true
		m.UI().SetStatus(state.StatusInfo, "正在獲取列表...", "", true)
		return m, tea.Batch(
			m.UI().SwitchView(state.LogViewerView),
			h.cmdBuilder.GetFail2BanListCmd(m),
		)

	case constants.KeyFail2Ban_Unban: // "4"
		if !info.Running {
			m.UI().SetStatus(state.StatusError, "服務未運行", "", false)
			return m, nil
		}

		// 1. 開啟輸入模式 (這會觸發 View 層隱藏菜單)
		m.Tools().Fail2BanInputMode = true

		// 2. 清空舊數據
		m.Tools().Fail2BanLogOutput = []string{}

		// 3. 設置提示
		m.UI().SetStatus(state.StatusInfo, "請輸入要解封的 IP", "", true)

		// 4. 批量執行：聚焦輸入框 + 獲取列表數據
		return m, tea.Batch(
			m.UI().TextInput.Focus(),
			h.cmdBuilder.GetFail2BanListCmd(m),
		)

	case constants.KeyFail2Ban_Config: // "5"
		if !info.Installed {
			m.UI().SetStatus(state.StatusError, "請先安裝", "", false)
			return m, nil
		}
		m.Tools().Fail2BanConfigStep = 1
		m.UI().SetStatus(state.StatusInfo, "請輸入最大錯誤重試次數", "默認: 5", true)
		return m, m.UI().TextInput.Focus()

	case constants.KeyFail2Ban_Uninstall: // "6"
		if !info.Installed {
			m.UI().SetStatus(state.StatusError, "未安裝", "", false)
			return m, nil
		}
		m.UI().SetStatus(state.StatusWarn, "正在卸載...", "", true)
		return m, h.cmdBuilder.UninstallFail2BanCmd()
	}
	return m, nil
}

// submitBBRMenu 處理 BBR 菜單提交
func (h *KeyHandler) submitBBRMenu(m *state.Manager, input string) (*state.Manager, tea.Cmd) {
	// 1. 處理確認輸入邏輯 (通用於 XanMod 和 BBR2)
	if m.Tools().BBRInstallConfirm {
		m.Tools().BBRInstallConfirm = false
		m.UI().ClearInput()
		target := m.Tools().BBRInstallTarget

		if input == "YES" {
			if target == "xanmod" {
				m.UI().SetStatus(state.StatusWarn, "正在安裝 XanMod 內核...", "請勿關閉終端", true)
				return m, h.cmdBuilder.InstallXanModCmd()
			} else if target == "bbr2" {
				m.UI().SetStatus(state.StatusWarn, "正在安裝 BBR2 專用內核...", "請勿關閉終端", true)
				return m, h.cmdBuilder.InstallBBR2KernelCmd()
			}
		} else {
			m.UI().SetStatus(state.StatusInfo, "操作已取消", "", false)
		}
		return m, nil
	}

	// 2. 正常菜單邏輯
	switch input {
	case constants.KeyBBR_Original: // "1"
		m.UI().SetStatus(state.StatusInfo, "正在啟用原版 BBR...", "", true)
		return m, h.cmdBuilder.EnableBBRCmd("bbr")

	case constants.KeyBBR_BBR2: // "2"
		m.UI().SetStatus(state.StatusInfo, "正在檢查內核兼容性...", "", true)
		return m, h.cmdBuilder.EnableBBRCmd("bbr2")

	case constants.KeyBBR_XanMod: // "3"
		if runtime.GOARCH != "amd64" {
			m.UI().SetStatus(
				state.StatusError,
				"不支持的系統架構",
				fmt.Sprintf("XanMod 內核僅支持 x86_64 (amd64),\n 檢測到當前為 %s", runtime.GOARCH),
				false,
			)
			return m, nil
		}

		// 架構通過，才進入確認模式
		m.Tools().BBRInstallConfirm = true
		m.Tools().BBRInstallTarget = "xanmod"
		m.UI().SetStatus(state.StatusFatal, "安裝 XanMod 內核警告", "輸入 YES 確認安裝", true)
		return m, m.UI().TextInput.Focus()

	case constants.KeyBBR_Disable: // "4"
		m.UI().SetStatus(state.StatusInfo, "正在禁用 BBR...", "", true)
		return m, h.cmdBuilder.DisableBBRCmd()
	}

	return m, nil
}

func (h *KeyHandler) submitStreamingCheck(m *state.Manager, input string) (*state.Manager, tea.Cmd) {
	if m.Tools().IsCheckingStreaming {
		return m, nil
	}

	m.UI().SetStatus(state.StatusInfo, "正在重新檢測...", "", true)
	m.Tools().StreamingResult = nil
	m.Tools().IsCheckingStreaming = true

	return m, h.cmdBuilder.CheckStreamingCmd(m, true)
}

func (h *KeyHandler) submitSystemCleanup(m *state.Manager, input string) (*state.Manager, tea.Cmd) {
	switch input {
	case constants.KeyCleanup_Scan:
		m.UI().SetStatus(state.StatusInfo, "正在掃描系統垃圾...", "", true)
		return m, h.cmdBuilder.ScanSystemCleanupCmd()

	case constants.KeyCleanup_Log:
		m.UI().SetStatus(state.StatusWarn, "正在清理日誌...", "", true)
		return m, h.cmdBuilder.CleanSystemCmd(m, "log")

	case constants.KeyCleanup_Pkg:
		m.UI().SetStatus(state.StatusWarn, "正在清理包緩存...", "", true)
		return m, h.cmdBuilder.CleanSystemCmd(m, "pkg")

	case constants.KeyCleanup_Temp:
		m.UI().SetStatus(state.StatusWarn, "正在清理臨時文件...", "", true)
		return m, h.cmdBuilder.CleanSystemCmd(m, "temp")

	case constants.KeyCleanup_All:
		m.UI().SetStatus(state.StatusWarn, "正在執行深度清理...", "", true)
		return m, h.cmdBuilder.CleanSystemCmd(m, "all")
	}
	return m, nil
}

// 備份管理
func (h *KeyHandler) submitBackupRestoreMenu(m *state.Manager, input string) (*state.Manager, tea.Cmd) {
	// 1. 處理 "確認操作" (YES/NO)
	if m.Backup().ConfirmMode {
		if strings.EqualFold(input, "YES") {
			m.Backup().ConfirmMode = false
			idx := m.Backup().BackupListCursor // 這是列表索引

			// 根據操作類型分流
			if m.Backup().PendingOp == "delete" {
				// --- 執行刪除 ---
				list := m.Backup().BackupList
				if idx >= 0 && idx < len(list) {
					targetName := list[idx].Name
					m.UI().SetStatus(state.StatusWarn, "正在刪除備份...", targetName, true)
					// 調用刪除命令 (這會觸發列表刷新)
					return m, h.cmdBuilder.DeleteBackupCmd(m, targetName)
				}
			} else {
				// --- 執行恢復 (默認) ---
				m.UI().SetStatus(state.StatusWarn, "正在恢復備份...", "服務將自動重啟", true)
				return m, h.cmdBuilder.RestoreBackupCmd(m, idx)
			}
		}

		// 取消操作
		m.Backup().ConfirmMode = false
		m.UI().SetStatus(state.StatusInfo, "操作已取消", "", false)
		return m, nil
	}

	// 2. 處理 "選擇備份編號" (輸入數字)
	if m.Backup().SelectingIndex {
		list := m.Backup().BackupList
		if idx, err := strconv.Atoi(input); err == nil && idx >= 1 && idx <= len(list) {
			m.Backup().SelectingIndex = false
			m.Backup().BackupListCursor = idx - 1 // 轉換為 0-based 索引
			m.Backup().ConfirmMode = true

			targetName := list[idx-1].Name

			// 根據操作類型顯示不同的確認提示
			if m.Backup().PendingOp == "delete" {
				m.UI().SetStatus(
					state.StatusFatal,
					fmt.Sprintf("⚠️  確認刪除 %s ?", targetName),
					"輸入 YES 確認 (此操作不可恢復)",
					true,
				)
			} else {
				m.UI().SetStatus(
					state.StatusWarn,
					fmt.Sprintf("確認恢復 %s ?", targetName),
					"輸入 YES 確認 (當前配置將被覆蓋)",
					true,
				)
			}
			return m, nil
		}

		m.Backup().SelectingIndex = false
		m.UI().SetStatus(state.StatusError, "無效的編號", "", false)
		return m, nil
	}

	// 3. 主菜單選項
	switch input {
	case constants.KeyBackup_Create: // "1"
		m.UI().SetStatus(state.StatusInfo, "正在創建備份...", "", true)
		return m, h.cmdBuilder.CreateBackupCmd(m)

	case constants.KeyBackup_Restore: // "2"
		if len(m.Backup().BackupList) == 0 {
			m.UI().SetStatus(state.StatusError, "沒有可用的備份文件", "", false)
			return m, nil
		}
		m.Backup().PendingOp = "restore" // [標記] 設置為恢復模式
		m.Backup().SelectingIndex = true
		m.UI().SetStatus(state.StatusInfo, "請輸入要 [恢復] 的備份編號:", "", true)
		return m, nil

	case constants.KeyBackup_Delete: // "3" (確保常量已定義，或直接寫 "3")
		if len(m.Backup().BackupList) == 0 {
			m.UI().SetStatus(state.StatusError, "沒有可用的備份文件", "", false)
			return m, nil
		}
		m.Backup().PendingOp = "delete" // [標記] 設置為刪除模式
		m.Backup().SelectingIndex = true
		m.UI().SetStatus(state.StatusInfo, "請輸入要 [刪除] 的備份編號:", "", true)
		return m, nil
	}
	return m, nil
}

// --- 日誌管理 ---

func (h *KeyHandler) submitLogMenu(m *state.Manager, input string) (*state.Manager, tea.Cmd) {
	// 輔助函數：切換前清空舊日誌，避免顯示"殘影"
	prepareLogView := func() tea.Cmd {
		// 清空內容並重置狀態
		m.Log().UpdateContent("")
		m.Log().IsFollowing = false
		return m.UI().SwitchView(state.LogViewerView)
	}

	switch input {
	case constants.KeyLog_Realtime:
		cmd1 := prepareLogView()
		cmd2 := h.cmdBuilder.ViewRealtimeLogCmd(m)
		m.UI().SetStatus(state.StatusInfo, "正在獲取實時日誌...", "", true)
		return m, tea.Batch(cmd1, cmd2)

	case constants.KeyLog_Full:
		cmd1 := prepareLogView()
		cmd2 := h.cmdBuilder.ViewFullLogCmd(m)
		m.UI().SetStatus(state.StatusInfo, "正在讀取日誌文件...", "大文件可能需要稍等片刻", true)
		return m, tea.Batch(cmd1, cmd2)

	case constants.KeyLog_Error:
		cmd1 := prepareLogView()
		cmd2 := h.cmdBuilder.ViewErrorLogCmd(m)
		m.UI().SetStatus(state.StatusWarn, "正在搜索錯誤日誌...", "", true)
		return m, tea.Batch(cmd1, cmd2)

	case constants.KeyLog_Level:
		return m, m.UI().SwitchView(state.LogLevelEditView)

	case constants.KeyLog_Export:
		m.UI().SetStatus(state.StatusInfo, "正在導出日誌...", "", true)
		return m, h.cmdBuilder.ExportLogCmd(m)

	case constants.KeyLog_Clear:
		m.UI().SetStatus(state.StatusWarn, "⚠️  正在清空日誌文件...", "", true)
		return m, tea.Batch(
			h.cmdBuilder.ClearLogCmd(m),
			h.cmdBuilder.LoadLogInfoCmd(m),
		)
	}
	return m, nil
}

func (h *KeyHandler) submitLogLevelEdit(m *state.Manager, input string) (*state.Manager, tea.Cmd) {
	level := ""
	switch input {
	case constants.KeyLevel_Debug:
		level = "debug"
	case constants.KeyLevel_Info:
		level = "info"
	case constants.KeyLevel_Warn:
		level = "warn"
	case constants.KeyLevel_Error:
		level = "error"
	}
	if level != "" {
		m.UI().SetStatus(state.StatusInfo, "正在更新日誌級別...", "", true)
		return m, h.cmdBuilder.ChangeLogLevelCmd(m, level)
	}
	return m, nil
}

// --- 節點信息 ---

func (h *KeyHandler) submitNodeInfo(m *state.Manager, input string) (*state.Manager, tea.Cmd) {
	switch input {
	case constants.KeyNode_Links:
		m.Node().SelectionMode = "links"
		cmd1 := m.UI().SwitchView(state.ProtocolLinksView)
		cmd2 := h.cmdBuilder.GenerateProtocolLinksCmd(m)
		return m, tea.Batch(cmd1, cmd2)

	case constants.KeyNode_QRCode:
		m.Node().SelectionMode = "qrcode"
		cmd1 := m.UI().SwitchView(state.ProtocolLinksView)
		cmd2 := h.cmdBuilder.GenerateProtocolLinksCmd(m)
		m.UI().SetStatus(state.StatusInfo, "請選擇一個協議查看二維碼", "", true)
		return m, tea.Batch(cmd1, cmd2)

	case constants.KeyNode_Subscription:
		cmd1 := m.UI().SwitchView(state.SubscriptionView)
		cmd2 := h.cmdBuilder.GenerateSubscriptionCmd(m)
		return m, tea.Batch(cmd1, cmd2)

	case constants.KeyNode_ClientConfig:
		return m, m.UI().SwitchView(state.ClientConfigView)

	case constants.KeyNode_Copy:
		count := len(m.Node().Links)
		if count == 0 {
			m.UI().SetStatus(state.StatusError, "鏈接尚未生成，請先進入[查看協議鏈接]", "", false)
			return m, nil
		}

		var indices []int
		for i := 0; i < count; i++ {
			indices = append(indices, i)
		}
		m.UI().SetStatus(state.StatusSuccess, "所有鏈接已複製到剪貼板", "", false)
		return m, h.cmdBuilder.BatchCopyLinksCmd(m, indices)
	}

	return m, nil
}

func (h *KeyHandler) submitProtocolLinks(m *state.Manager, input string) (*state.Manager, tea.Cmd) {
	input = strings.TrimSpace(strings.ToLower(input))
	mode := m.Node().SelectionMode

	// ----------------------------------------------------
	// 模式 A: 二維碼選擇模式 (qrcode)
	// ----------------------------------------------------
	if mode == "qrcode" {
		// 1. 禁止 "a" (全選)
		if input == constants.KeyNode_Copy {
			m.UI().SetStatus(state.StatusError, "二維碼模式不支持批量操作，請輸入單個序號", "", false)
			return m, nil
		}

		// 2. 禁止多選 (逗號檢查)
		if strings.Contains(input, ",") {
			m.UI().SetStatus(state.StatusError, "請輸入單個序號 (不支持多選)", "", false)
			return m, nil
		}

		// 3. 處理單個數字
		if idx, err := strconv.Atoi(input); err == nil && idx >= 1 {
			// 初始化二維碼設置
			m.Node().QRInvert = false
			m.Node().QRLevel = "L"
			m.Node().CurrentQRIndex = idx - 1

			m.Node().CurrentQRType = "protocol"

			cmd1 := m.UI().SwitchView(state.QRCodeView)
			cmd2 := h.cmdBuilder.GenerateQRCodeCmd(m, idx-1)
			return m, tea.Batch(cmd1, cmd2)
		}

		return m, nil
	}

	// ----------------------------------------------------
	// 模式 B: 鏈接查看模式 (links) - 保持原有功能
	// ----------------------------------------------------

	// 1. 處理 "a" 全選
	if input == constants.KeyNode_Copy {
		count := len(m.Node().Links)
		var indices []int
		for i := 0; i < count; i++ {
			indices = append(indices, i)
		}
		m.UI().SetStatus(state.StatusSuccess, "所有鏈接已複製到剪貼板", "", false)
		return m, h.cmdBuilder.BatchCopyLinksCmd(m, indices)
	}

	// 2. 兼容 "q+數字" 快捷跳轉 (雖然現在有專門的菜單，但保留快捷鍵也無妨)
	if strings.HasPrefix(input, "q") {
		idxStr := strings.TrimPrefix(input, "q")
		if idx, err := strconv.Atoi(idxStr); err == nil && idx >= 1 {
			m.Node().QRInvert = false
			m.Node().QRLevel = "L"
			m.Node().CurrentQRIndex = idx - 1

			m.Node().CurrentQRType = "protocol"

			cmd1 := m.UI().SwitchView(state.QRCodeView)
			cmd2 := h.cmdBuilder.GenerateQRCodeCmd(m, idx-1)
			return m, tea.Batch(cmd1, cmd2)
		}
	}

	// 3. 處理數字/多選複製 (如 1,3)
	parts := strings.Split(input, ",")
	var indices []int
	valid := false

	for _, p := range parts {
		if idx, err := strconv.Atoi(strings.TrimSpace(p)); err == nil && idx >= 1 {
			indices = append(indices, idx-1)
			valid = true
		}
	}

	if valid {
		m.UI().SetStatus(state.StatusSuccess, "選中鏈接已複製", "", false)
		return m, h.cmdBuilder.BatchCopyLinksCmd(m, indices)
	}

	return m, nil
}

// submitQRCodeView 處理二維碼視圖的按鍵
func (h *KeyHandler) submitQRCodeView(m *state.Manager, input string) (*state.Manager, tea.Cmd) {
	return m, nil
}

func (h *KeyHandler) submitSubscription(m *state.Manager, input string) (*state.Manager, tea.Cmd) {
	switch input {
	case constants.KeySubscription_CopyOnline: // "1"
		m.UI().SetStatus(state.StatusInfo, "正在發送複製指令...", "", true)
		return m, h.cmdBuilder.CopySubscriptionOnlineCmd(m)

	case constants.KeySubscription_CopyOffline: // "2"
		m.UI().SetStatus(state.StatusInfo, "正在發送複製指令 (內容較長請稍候)...", "", true)
		return m, h.cmdBuilder.CopySubscriptionOfflineCmd(m)

	case constants.KeySubscription_Refresh: // "3"
		return m, h.cmdBuilder.GenerateSubscriptionCmd(m)

	case constants.KeySubscription_QRCode: // "4"
		m.Node().CurrentQRType = "subscription"

		cmd1 := m.UI().SwitchView(state.QRCodeView)
		cmd2 := h.cmdBuilder.GenerateSubscriptionQRCodeCmd(m)
		return m, tea.Batch(cmd1, cmd2)
	}
	return m, nil
}

func (h *KeyHandler) submitClientConfig(m *state.Manager, input string) (*state.Manager, tea.Cmd) {
	switch input {
	case constants.KeyExport_Full:
		return m, h.cmdBuilder.ExportClientConfigCmd(m, "full", "json")
	case constants.KeyExport_Clash:
		return m, h.cmdBuilder.ExportClientConfigCmd(m, "clash", "yaml")
	case constants.KeyExport_Custom:
		m.Node().SelectionMode = "params"
		cmd1 := m.UI().SwitchView(state.ProtocolLinksView)
		cmd2 := h.cmdBuilder.GenerateNodeParamsCmd(m)
		return m, tea.Batch(cmd1, cmd2)
	}
	return m, nil
}

// --- 卸載 ---

// 處理卸載界面的按鍵
func (h *KeyHandler) submitUninstall(m *state.Manager, input string) (*state.Manager, tea.Cmd) {
	uninstall := m.Uninstall()

	if uninstall.ConfirmStep == 1 {
		if strings.TrimSpace(input) == "UNINSTALL" {
			cmd1 := m.UI().SwitchView(state.UninstallProgressView)
			cmd2 := h.cmdBuilder.UninstallPrismCmd(m)
			return m, tea.Batch(cmd1, cmd2)
		}
		return m, nil
	}

	switch input {
	case constants.KeyUninstall_KeepConfig:
		uninstall.KeepConfig = !uninstall.KeepConfig
	case constants.KeyUninstall_KeepCert:
		uninstall.KeepCerts = !uninstall.KeepCerts
	case constants.KeyUninstall_KeepBackup:
		uninstall.KeepBackups = !uninstall.KeepBackups
	case constants.KeyUninstall_KeepLog:
		uninstall.KeepLogs = !uninstall.KeepLogs
	case constants.KeyUninstall_ConfirmStep:
		uninstall.NextConfirmStep()
		return m, m.UI().TextInput.Focus()
	}

	return m, nil
}

// ========================================
// 輔助函數 (Esc 處理)
// ========================================

func (h *KeyHandler) handleInputEscape(m *state.Manager, view state.View) (*state.Manager, tea.Cmd) {
	// [全局優化] 按 Esc 返回時，先強制清除輸入框和狀態欄
	m.UI().SetStatus(state.StatusInfo, "", "", false)

	if m.UI().GetInputBuffer() != "" {
		m.UI().ClearInput()
		return m, m.UI().TextInput.Focus()
	}

	// 1. 如果已經在退出確認模式，再次按 Esc 取消退出
	if m.Config().ExitConfirmMode {
		m.Config().ExitConfirmMode = false
		m.UI().SetStatus(state.StatusInfo, "已取消退出", "", false)
		return m, m.UI().TextInput.Focus()
	}

	// 2. 如果在重置確認模式，按 Esc 取消
	if m.Config().ConfirmMode {
		m.Config().ConfirmMode = false
		m.UI().SetStatus(state.StatusInfo, "已取消重置", "", false)
		return m, m.UI().TextInput.Focus()
	}

	// 3. 標準狀態重置
	if m.Port().PortEditingMode {
		m.Port().CancelPortEdit()
	}
	if m.Cert().CertConfirmMode {
		m.Cert().ResetCertDeletion()
	}
	if view == state.DNSCredentialInputView {
		if m.Cert().DNSStep > 0 {
			m.Cert().PrevDNSStep()
			return m, m.UI().TextInput.Focus()
		}
	}
	if m.Backup().SelectingIndex || m.Backup().ConfirmMode {
		m.Backup().SelectingIndex = false
		m.Backup().ConfirmMode = false
		m.UI().SetStatus(state.StatusInfo, "已取消", "", false)
		return m, m.UI().TextInput.Focus()
	}

	// 4. 配置菜單退出攔截
	if view == state.ConfigMenuView && m.Config().HasUnsavedChanges {
		m.Config().ExitConfirmMode = true
		m.UI().SetStatus(state.StatusWarn, "⚠️  配置已更改，是否應用並重啟服務？ (Y/N)", "", false)
		return m, m.UI().TextInput.Focus()
	}

	// Fail2Ban 配置流程中按 Esc 取消
	if m.Tools().Fail2BanConfigStep > 0 || m.Tools().Fail2BanInputMode {
		m.Tools().Fail2BanConfigStep = 0
		m.Tools().Fail2BanInputMode = false
		m.UI().SetStatus(state.StatusInfo, "已取消操作", "", false)
		return m, m.UI().TextInput.Focus()
	}

	// BBR 確認模式下按 Esc 取消
	if m.Tools().BBRInstallConfirm {
		m.Tools().BBRInstallConfirm = false
		m.UI().SetStatus(state.StatusInfo, "已取消安裝", "", false)
		return m, m.UI().TextInput.Focus()
	}

	// 5. 標準退出路由
	switch view {
	case state.MainMenuView:
		m.UI().SetStatus(state.StatusInfo, "已在主菜單，按 q 退出程序", "", false)
		return m, nil

	case state.ConfigMenuView,
		state.CertMenuView,
		state.OutboundMenuView,
		state.RouteMenuView,
		state.CoreMenuView,
		state.ToolsMenuView,
		state.LogMenuView,
		state.NodeInfoView,
		state.ServiceMenuView,
		state.InstallWizardView:
		return m, m.UI().SwitchView(state.MainMenuView)

	case state.ProtocolMenuView,
		state.SNIEditView,
		state.UUIDEditView,
		state.PortEditView,
		state.AnyTLSPaddingView:
		return m, m.UI().SwitchView(state.ConfigMenuView)

	case state.Hy2PortModeView:
		return m, m.UI().SwitchView(state.PortEditView)

	case state.ACMEHTTPInputView:
		if m.Cert().HTTPStep > 0 {
			m.Cert().PrevHTTPStep()
			m.UI().SetStatus(state.StatusInfo, "已返回上一步", "請輸入郵箱地址", false)
			return m, m.UI().TextInput.Focus()
		}
		return m, m.UI().SwitchView(state.CertMenuView)

	case state.ACMEDNSProviderView,
		state.DNSCredentialInputView,
		state.ProviderSwitchView,
		state.CertRenewView,
		state.CertDeleteView,
		state.CertStatusView,
		state.CertModeMenuView:
		return m, m.UI().SwitchView(state.CertMenuView)

	case state.WARPRoutingView,
		state.Socks5RoutingView,
		state.IPv6RoutingView,
		state.DNSRoutingView,
		state.SNIProxyRoutingView:
		return m, m.UI().SwitchView(state.RouteMenuView)

	case state.Socks5InboundView,
		state.Socks5OutboundView:
		return m, m.UI().SwitchView(state.Socks5RoutingView)

	case state.ScriptUpdateView:
		m.Core().IsCheckingScript = false
		return m, m.UI().SwitchView(state.CoreMenuView)

	case state.CoreVersionSelectView,
		state.CoreSourceSelectView:
		return m, m.UI().SwitchView(state.CoreMenuView)

	case state.ServiceLogView,
		state.ServiceHealthView:
		return m, m.UI().SwitchView(state.ServiceMenuView)

	case state.SwapMenuView,
		state.Fail2BanMenuView,
		state.BBRMenuView,
		state.StreamingCheckView,
		state.SystemCleanupView,
		state.BackupMenuView,
		state.BackupRestoreMenuView:
		return m, m.UI().SwitchView(state.ToolsMenuView)

	case state.LogViewerView:
		if m.Tools().IsViewingFail2BanLogs {
			m.Tools().IsViewingFail2BanLogs = false
			return m, m.UI().SwitchView(state.Fail2BanMenuView)
		}
		return m, m.UI().SwitchView(state.LogMenuView)

	case state.LogLevelEditView:
		return m, m.UI().SwitchView(state.LogMenuView)

	case state.ProtocolLinksView,
		state.SubscriptionView,
		state.ClientConfigView:
		return m, m.UI().SwitchView(state.NodeInfoView)

	case state.QRCodeView:
		if m.Node().CurrentQRType == "subscription" {
			return m, m.UI().SwitchView(state.SubscriptionView)
		}
		return m, m.UI().SwitchView(state.ProtocolLinksView)

	case state.UninstallView:
		if m.Uninstall().ConfirmStep == 1 {
			m.Uninstall().ConfirmStep = 0
			m.UI().SetStatus(state.StatusInfo, "", "", false)
			return m, m.UI().TextInput.Focus()
		}
		return m, m.UI().SwitchView(state.MainMenuView)

	case state.UninstallProgressView:
		return m, m.UI().SwitchView(state.MainMenuView)

	default:
		return m, m.UI().SwitchView(state.MainMenuView)
	}
}

func toggleProtocolsFromInput(s *state.InstallState, input string) {
	parts := strings.Split(input, ",")
	for _, p := range parts {
		if n, err := strconv.Atoi(strings.TrimSpace(p)); err == nil {
			curr := s.InstallProtocols
			var newP []int
			found := false
			for _, id := range curr {
				if id == n {
					found = true
				} else {
					newP = append(newP, id)
				}
			}
			if !found {
				newP = append(curr, n)
			}
			s.InstallProtocols = newP
		}
	}
}

func toggleIntList(current []int, input string) []int {
	m := make(map[int]bool)
	for _, id := range current {
		m[id] = true
	}
	parts := strings.Split(input, ",")
	for _, p := range parts {
		if n, err := strconv.Atoi(strings.TrimSpace(p)); err == nil {
			if m[n] {
				delete(m, n)
			} else {
				m[n] = true
			}
		}
	}
	var res []int
	for id := range m {
		res = append(res, id)
	}
	return res
}
