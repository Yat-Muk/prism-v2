package model

import (
	"strings"
	"testing"

	domainConfig "github.com/Yat-Muk/prism-v2/internal/domain/config"
	"github.com/Yat-Muk/prism-v2/internal/tui/constants"
	"github.com/Yat-Muk/prism-v2/internal/tui/handlers"
	"github.com/Yat-Muk/prism-v2/internal/tui/msg"
	"github.com/Yat-Muk/prism-v2/internal/tui/state"
	tea "github.com/charmbracelet/bubbletea"
	"go.uber.org/zap"
)

// setupTestRouter 初始化測試用的 Router
func setupTestRouter() *Router {
	logger := zap.NewNop()
	defaultCfg := domainConfig.DefaultConfig()

	// 1. 構建 State Config
	stateCfg := &state.Config{
		Log:           logger,
		InitialConfig: defaultCfg,
	}
	stateMgr := state.NewManager(stateCfg)

	// 2. 構建 Handler Config (模擬依賴)
	handlerCfg := &handlers.Config{
		Log:      logger,
		StateMgr: stateMgr,
		// 其他 Service 依賴可為 nil，CommandBuilder 內部會處理
	}

	// 3. 創建 Router
	return NewRouter(handlerCfg)
}

// TestRouter_Init 測試初始化命令
func TestRouter_Init(t *testing.T) {
	r := setupTestRouter()

	cmd := r.InitModel()
	if cmd == nil {
		t.Error("InitModel should return initial commands")
	}
}

// TestRouter_Update_KeyMsg 測試按鍵消息路由
func TestRouter_Update_KeyMsg(t *testing.T) {
	r := setupTestRouter()

	// 1. 初始狀態應為 MainMenuView
	if r.stateMgr.UI().CurrentView != state.MainMenuView {
		t.Errorf("Expected initial view MainMenuView, got %v", r.stateMgr.UI().CurrentView)
	}

	// 2. 發送按鍵 (進入 ConfigMenuView)
	// ✅ 修正：使用常量 KeyMain_Config 替代硬編碼的 "1"
	r.stateMgr.UI().TextInput.SetValue(constants.KeyMain_Config)
	keyMsg := tea.KeyMsg{Type: tea.KeyEnter}

	// 3. 調用 Update
	_, cmd := r.Update(keyMsg)

	// 4. 驗證視圖跳轉
	if r.stateMgr.UI().CurrentView != state.ConfigMenuView {
		t.Errorf("Expected view transition to ConfigMenuView, got %v", r.stateMgr.UI().CurrentView)
	}

	// 5. 驗證是否返回了 Cmd (這通常是 nil，因為切換視圖是純狀態變更)
	if cmd != nil {
		t.Log("Cmd generated from view switch (acceptable)")
	}
}

// TestRouter_Update_CustomMsg 測試自定義業務消息路由
func TestRouter_Update_CustomMsg(t *testing.T) {
	r := setupTestRouter()

	// 模擬核心檢查完成消息
	updateMsg := msg.CoreCheckMsg{
		HasUpdate:     true,
		LatestVersion: "v1.10.0",
		IsSilent:      false,
	}

	// 調用 Update
	r.Update(updateMsg)

	// 驗證狀態是否更新
	coreState := r.stateMgr.Core()
	if !coreState.HasUpdate {
		t.Error("CoreState.HasUpdate should be true")
	}
	if coreState.LatestVersion != "v1.10.0" {
		t.Errorf("Expected version v1.10.0, got %s", coreState.LatestVersion)
	}

	// 驗證狀態欄是否顯示提示
	statusMsg := r.stateMgr.UI().Status.Message
	if !strings.Contains(statusMsg, "發現新版本") {
		t.Errorf("Status bar should show update message, got: %s", statusMsg)
	}
}

// TestRouter_View 測試渲染函數防崩潰
func TestRouter_View(t *testing.T) {
	r := setupTestRouter()

	// 確保 View() 調用不會 Panic
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("View() panicked: %v", r)
		}
	}()

	output := r.View()
	if len(output) == 0 {
		t.Error("View() returned empty string")
	}
}

// TestRouter_WindowSize 測試窗口調整消息
func TestRouter_WindowSize(t *testing.T) {
	r := setupTestRouter()

	winMsg := tea.WindowSizeMsg{Width: 100, Height: 50}
	r.Update(winMsg)

	if r.stateMgr.UI().Width != 100 || r.stateMgr.UI().Height != 50 {
		t.Errorf("UI dimensions not updated. Got %dx%d", r.stateMgr.UI().Width, r.stateMgr.UI().Height)
	}
}
