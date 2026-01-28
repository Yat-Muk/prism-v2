package handlers

import (
	"fmt"
	"testing"

	domainConfig "github.com/Yat-Muk/prism-v2/internal/domain/config"
	"github.com/Yat-Muk/prism-v2/internal/tui/constants"
	"github.com/Yat-Muk/prism-v2/internal/tui/state"
	tea "github.com/charmbracelet/bubbletea"
	"go.uber.org/zap"
)

// setupTestEnv 初始化測試環境
func setupTestEnv() (*state.Manager, *KeyHandler) {
	// 1. 準備基礎依賴
	logger := zap.NewNop()
	defaultCfg := domainConfig.DefaultConfig()

	// 2. 構建 state.Config
	stateCfg := &state.Config{
		Log:           logger,
		InitialConfig: defaultCfg,
	}

	// 3. 創建 State Manager
	sm := state.NewManager(stateCfg)

	// 4. 構建 Handler
	kh := NewKeyHandler(sm, &CommandBuilder{}, &CertHandler{})

	return sm, kh
}

// helper: 模擬用戶輸入並按下 Enter
func sendKey(h *KeyHandler, m *state.Manager, input string) (*state.Manager, tea.Cmd) {
	// 1. ✅ 核心修復：使用 TextInput 組件的 SetValue 方法設置內容
	m.UI().TextInput.SetValue(input)

	// 2. 發送 Enter 鍵觸發提交邏輯
	return h.Handle(tea.KeyMsg{Type: tea.KeyEnter}, m)
}

// TestNavigation_MainMenu 測試主菜單導航
func TestNavigation_MainMenu(t *testing.T) {
	m, h := setupTestEnv()
	m.UI().SwitchView(state.MainMenuView)

	tests := []struct {
		input    string
		wantView state.View
	}{
		{constants.KeyMain_Config, state.ConfigMenuView},
		{constants.KeyMain_InstallWizard, state.InstallWizardView},
		{constants.KeyMain_Outbound, state.OutboundMenuView},
		{constants.KeyMain_Route, state.RouteMenuView},
		{constants.KeyMain_Core, state.CoreMenuView},
		{constants.KeyMain_Tools, state.ToolsMenuView},
		{constants.KeyMain_Log, state.LogMenuView},
		{constants.KeyMain_Uninstall, state.UninstallView},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			m.UI().SwitchView(state.MainMenuView)

			_, _ = sendKey(h, m, tt.input)

			if m.UI().CurrentView != tt.wantView {
				t.Errorf("輸入 %s: 預期視圖 %v, 實際 %v", tt.input, tt.wantView, m.UI().CurrentView)
			}
		})
	}
}

// TestNavigation_Escape 測試 Esc 返回邏輯
func TestNavigation_Escape(t *testing.T) {
	m, h := setupTestEnv()

	// 使用 View(-1) 模擬未知的視圖
	unknownView := state.View(-1)

	tests := []struct {
		startView state.View
		wantView  state.View
	}{
		{state.ConfigMenuView, state.MainMenuView},
		{state.ProtocolMenuView, state.ConfigMenuView},
		{state.PortEditView, state.ConfigMenuView},
		{state.Hy2PortModeView, state.PortEditView},
		{state.ServiceLogView, state.ServiceMenuView},
		{unknownView, state.MainMenuView}, // 默認兜底
	}

	for _, tt := range tests {
		name := fmt.Sprintf("View(%d)", tt.startView)
		t.Run(name, func(t *testing.T) {
			m.UI().SwitchView(tt.startView)

			h.Handle(tea.KeyMsg{Type: tea.KeyEsc}, m)

			if m.UI().CurrentView != tt.wantView {
				t.Errorf("從 %v 按 Esc: 預期返回 %v, 實際 %v", tt.startView, tt.wantView, m.UI().CurrentView)
			}
		})
	}
}

// TestConfig_ProtocolToggle 測試協議開關邏輯
func TestConfig_ProtocolToggle(t *testing.T) {
	m, h := setupTestEnv()
	m.UI().SwitchView(state.ProtocolMenuView)

	// 初始狀態: 假設開啟了 [1, 2]
	m.Config().EnabledProtocols = []int{1, 2}

	// 操作: 輸入 "2, 3" (應該關閉 2，開啟 3) -> 結果應為 [1, 3]
	sendKey(h, m, "2, 3")

	enabled := m.Config().EnabledProtocols
	has1, has2, has3 := false, false, false
	for _, id := range enabled {
		if id == 1 {
			has1 = true
		}
		if id == 2 {
			has2 = true
		}
		if id == 3 {
			has3 = true
		}
	}

	if !has1 || has2 || !has3 {
		t.Errorf("協議切換失敗。預期 [1, 3], 實際 %v", enabled)
	}

	// 驗證 Config 結構體同步
	if !m.Config().Config.Protocols.Hysteria2.Enabled {
		t.Error("Hysteria2 在 Config 結構體中的狀態未同步開啟")
	}

	if !m.Config().HasUnsavedChanges {
		t.Error("配置修改後應標記為未保存 (HasUnsavedChanges)")
	}
}

// TestInstallWizard_Logic 測試安裝嚮導邏輯
func TestInstallWizard_Logic(t *testing.T) {
	m, h := setupTestEnv()
	m.UI().SwitchView(state.InstallWizardView)

	// 場景 1: 輸入為空且未選擇任何協議 -> 報警
	m.Install().InstallProtocols = []int{}
	sendKey(h, m, "")

	if m.UI().Status.Type != state.StatusWarn {
		t.Error("提交空選擇時應顯示警告狀態")
	}

	// 場景 2: 輸入 "1, 4" -> 更新選擇但不跳轉
	sendKey(h, m, "1, 4")

	has4 := false
	for _, id := range m.Install().InstallProtocols {
		if id == 4 {
			has4 = true
		}
	}
	if !has4 {
		t.Error("安裝選擇列表未更新")
	}
	if m.UI().CurrentView != state.InstallWizardView {
		t.Error("更新選擇後應停留在嚮導視圖")
	}

	// 場景 3: 確認安裝 -> 跳轉進度
	sendKey(h, m, "")
	if m.UI().CurrentView != state.InstallProgressView {
		t.Errorf("應跳轉至 InstallProgressView, 實際為 %v", m.UI().CurrentView)
	}
}

// TestPortEdit_Flow 測試端口編輯流程
func TestPortEdit_Flow(t *testing.T) {
	m, h := setupTestEnv()
	m.UI().SwitchView(state.PortEditView)

	m.Config().EnabledProtocols = []int{1, 4}

	// 1. 進入編輯模式
	sendKey(h, m, "1")

	if !m.Port().PortEditingMode {
		t.Error("應進入端口編輯模式")
	}

	// 2. 提交新端口
	_, cmd := sendKey(h, m, "8443")

	if m.Port().PortEditingMode {
		t.Error("提交後應退出編輯模式")
	}
	if cmd == nil {
		t.Error("應生成更新端口的 Cmd")
	}
}

// TestConfig_ResetConfirm 測試配置重置確認流程
func TestConfig_ResetConfirm(t *testing.T) {
	m, h := setupTestEnv()
	m.UI().SwitchView(state.ConfigMenuView)

	// 1. 觸發重置
	sendKey(h, m, "r")
	if !m.Config().ConfirmMode {
		t.Error("應進入確認模式")
	}

	// 2. 取消
	sendKey(h, m, "NO")
	if m.Config().ConfirmMode {
		t.Error("應退出確認模式")
	}

	// 3. 確認
	sendKey(h, m, "r")
	sendKey(h, m, "YES")
	if m.Config().ConfirmMode {
		t.Error("確認 YES 後應退出確認模式")
	}
}

// TestEsc_ComplexStates 測試複雜狀態下的 Esc 行為
func TestEsc_ComplexStates(t *testing.T) {
	m, h := setupTestEnv()

	// 場景 1: 取消端口編輯
	m.UI().SwitchView(state.PortEditView)
	m.Port().StartPortEdit(1)

	h.Handle(tea.KeyMsg{Type: tea.KeyEsc}, m)

	if m.Port().PortEditingMode {
		t.Error("按 Esc 應取消端口編輯模式")
	}

	// 場景 2: 退出未保存的配置菜單
	m.UI().SwitchView(state.ConfigMenuView)
	m.Config().HasUnsavedChanges = true

	h.Handle(tea.KeyMsg{Type: tea.KeyEsc}, m)

	if !m.Config().ExitConfirmMode {
		t.Error("離開未保存的配置菜單時應進入 ExitConfirmMode")
	}
}
