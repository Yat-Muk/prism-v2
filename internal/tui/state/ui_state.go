package state

import (
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	ttea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// View 定義視圖枚舉
type View int

const (
	// ===================================
	// 基礎視圖 (0-99)
	// ===================================
	MainMenuView View = iota
	LogView
	LogViewerView // 專門的日誌查看器 (全屏/滾動)

	// ===================================
	// 安裝嚮導 (100-199)
	// ===================================
	InstallWizardView
	ProtocolSelectionView
	PortConfigView
	DomainConfigView
	InstallConfirmView
	InstallProgressView
	InstallCompleteView

	// ===================================
	// 配置管理 (200-299)
	// ===================================
	ConfigMenuView
	ProtocolMenuView
	PortEditView
	TLSMenuView
	RealityVisionView
	Hysteria2View
	Hy2PortModeView
	TUICView
	ShadowTLSView
	BrutalView
	AnyTLSPaddingView
	SNIEditView
	UUIDEditView
	OutboundMenuView

	// ===================================
	// 路由管理 (300-399)
	// ===================================
	RouteMenuView
	WARPRoutingView
	Socks5RoutingView
	Socks5InboundView
	Socks5OutboundView
	IPv6RoutingView
	DNSRoutingView
	SNIProxyRoutingView

	// ===================================
	// 證書管理 (400-499)
	// ===================================
	CertMenuView
	CertGenerateView
	CertUploadView
	CertRenewView
	CertDeleteView
	CertStatusView
	CertModeMenuView
	ACMEHTTPInputView
	ACMEDNSProviderView
	DNSCredentialInputView
	ProviderSwitchView

	// ===================================
	// 核心與系統 (500-599)
	// ===================================
	CoreMenuView
	CoreVersionSelectView
	CoreSourceSelectView

	ServiceMenuView
	ServiceLogView
	ServiceHealthView

	ToolsMenuView
	ScriptUpdateView
	BBRMenuView
	Fail2BanMenuView
	SwapMenuView
	StreamingCheckView
	SystemCleanupView

	BackupMenuView
	BackupRestoreMenuView
	UninstallView
	UninstallProgressView

	// ===================================
	// 日誌管理 (600-699)
	// ===================================
	LogMenuView
	LogLevelEditView

	// ===================================
	// 節點信息 (700-799)
	// ===================================
	NodeInfoView
	ProtocolLinksView
	SubscriptionView
	QRCodeView
	ClientConfigView
)

// StatusType 狀態類型
type StatusType int

const (
	StatusReady StatusType = iota
	StatusSuccess
	StatusError
	StatusFatal
	StatusInfo
	StatusWarn
)

// StatusMsg 狀態欄消息
type StatusMsg struct {
	Type    StatusType
	Message string
	Detail  string
	Show    bool
}

// ConfigLoadState 配置加載狀態
type ConfigLoadState int

const (
	ConfigNotLoaded ConfigLoadState = iota
	ConfigLoading
	ConfigLoaded
	ConfigError
)

// UIState UI 核心狀態
type UIState struct {
	CurrentView     View
	PreviousView    View // 用於返回
	TextInput       textinput.Model
	Spinner         spinner.Model
	Width           int
	Height          int
	Cursor          int
	Blink           bool // 光標閃爍狀態
	Status          StatusMsg
	ConfigLoadState ConfigLoadState
}

// NewUIState 創建 UI 狀態
func NewUIState() *UIState {
	ti := textinput.New()
	ti.Prompt = ""
	ti.CharLimit = 100
	ti.Width = 50
	ti.Focus()

	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	return &UIState{
		CurrentView:     MainMenuView,
		TextInput:       ti,
		Width:           80,
		Height:          24,
		Cursor:          0,
		Blink:           false,
		Status:          StatusMsg{Type: StatusReady},
		ConfigLoadState: ConfigNotLoaded,
		Spinner:         s,
	}
}

// SwitchView 切換視圖
func (s *UIState) SwitchView(v View) ttea.Cmd {
	s.PreviousView = s.CurrentView // 記錄上一級，便於返回
	s.CurrentView = v
	s.TextInput.Reset()
	s.Cursor = 0 // 重置光標位置

	// 切換視圖時重置狀態欄（除非是錯誤狀態，保留給用戶看）
	if s.Status.Type != StatusError && s.Status.Type != StatusFatal {
		s.Status = StatusMsg{Type: StatusReady}
	}

	return s.TextInput.Focus()
}

// SetStatus 設置狀態欄消息
func (s *UIState) SetStatus(t StatusType, msg, detail string, show bool) {
	s.Status = StatusMsg{
		Type:    t,
		Message: msg,
		Detail:  detail,
		Show:    show,
	}
}

// UpdateInput 更新輸入框
func (s *UIState) UpdateInput(msg ttea.Msg) ttea.Cmd {
	var cmd ttea.Cmd
	s.TextInput, cmd = s.TextInput.Update(msg)
	return cmd
}

func (s *UIState) GetInputBuffer() string {
	return s.TextInput.Value()
}

func (s *UIState) ClearInput() {
	s.TextInput.Reset()
}

// UpdateSize 更新尺寸
func (s *UIState) UpdateSize(w, h int) {
	s.Width = w
	s.Height = h
}
