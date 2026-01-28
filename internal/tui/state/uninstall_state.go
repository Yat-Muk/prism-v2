package state

import (
	"github.com/Yat-Muk/prism-v2/internal/tui/types"
)

// UninstallState 卸載狀態
type UninstallState struct {
	Scanning    bool
	ConfirmStep int // 0: 選擇保留項, 1: 最終確認

	// 於存儲掃描結果 (防止 nil 崩潰)
	Info *types.UninstallInfo

	// 存儲卸載進度步驟
	Steps []types.UninstallStep

	// 用戶選擇保留的項目
	KeepConfig  bool
	KeepCerts   bool
	KeepBackups bool
	KeepLogs    bool

	// 執行狀態
	Uninstalling bool
	CurrentStep  int
	TotalSteps   int
}

// NewUninstallState 創建卸載狀態
func NewUninstallState() *UninstallState {
	return &UninstallState{
		Info:         nil,
		Scanning:     true,
		ConfirmStep:  0,
		KeepConfig:   false,
		KeepCerts:    false,
		KeepBackups:  false,
		KeepLogs:     false,
		Uninstalling: false,
		CurrentStep:  0,
		TotalSteps:   0,
	}
}

// Reset 重置狀態
func (s *UninstallState) Reset() {
	s.Info = nil
	s.Scanning = true
	s.ConfirmStep = 0
	s.KeepConfig = false
	s.KeepCerts = false
	s.KeepBackups = false
	s.KeepLogs = false
	s.Uninstalling = false
	s.CurrentStep = 0
	s.TotalSteps = 0
}

// NextConfirmStep 進入下一步
func (s *UninstallState) NextConfirmStep() {
	s.ConfirmStep++
}
