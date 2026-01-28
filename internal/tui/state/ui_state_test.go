package state

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUIState_SwitchView(t *testing.T) {
	ui := NewUIState()

	ui.SwitchView(CertMenuView)

	assert.Equal(t, CertMenuView, ui.CurrentView) // 改為 CurrentView
	assert.Equal(t, "", ui.GetInputBuffer())
	assert.Equal(t, StatusReady, ui.Status.Type) // 改為 Status
}

func TestUIState_InputOperations(t *testing.T) {
	// 由於我們去掉了 AppendInput 這種模擬方法，依賴 bubbletea 的 Update，
	// 單元測試這裡主要測試封裝的 Get/Clear 方法
	ui := NewUIState()

	// 直接操作底層 Model 模擬輸入
	ui.TextInput.SetValue("ab")
	assert.Equal(t, "ab", ui.GetInputBuffer())

	ui.ClearInput()
	assert.Equal(t, "", ui.GetInputBuffer())
}

// 端口編輯測試移至 PortState 相關測試，或者適配新的字段訪問方式
func TestPortState_Editing(t *testing.T) {
	ps := NewPortState()

	ps.StartPortEdit(3)
	assert.True(t, ps.PortEditingMode) // 直接訪問字段
	assert.Equal(t, 3, ps.PortEditingProtocol)

	ps.StartHy2HoppingEdit()
	assert.True(t, ps.Hy2EditingHopping)
}
