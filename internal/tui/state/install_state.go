package state

import (
	"sort"
	"strconv"
	"strings"
)

// InstallState 安裝嚮導狀態管理
type InstallState struct {
	InstallProtocols []int    // 待安裝的協議列表（1-7）
	Logs             []string // 日誌字段
	IsFinished       bool     // 標記安裝流程是否結束 (無論成功失敗)
}

// NewInstallState 創建安裝狀態管理器
func NewInstallState() *InstallState {
	return &InstallState{
		InstallProtocols: []int{},
		Logs:             []string{}, // 初始化
	}
}

// 添加日誌的方法
func (s *InstallState) AddLog(text string) {
	s.Logs = append(s.Logs, text)
}

// 清空日誌
func (s *InstallState) ResetLogs() {
	s.Logs = []string{}
	s.IsFinished = false
}

// ToggleProtocol 切換協議選中狀態 (邏輯方法保留)
func (s *InstallState) ToggleProtocol(protoID int) {
	for i, id := range s.InstallProtocols {
		if id == protoID {
			s.InstallProtocols = append(s.InstallProtocols[:i], s.InstallProtocols[i+1:]...)
			return
		}
	}
	s.InstallProtocols = append(s.InstallProtocols, protoID)
	sort.Ints(s.InstallProtocols)
}

// ToggleProtocols 批量切換
func (s *InstallState) ToggleProtocols(input string) {
	parts := strings.Split(input, ",")
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if n, err := strconv.Atoi(p); err == nil && n >= 1 && n <= 7 {
			s.ToggleProtocol(n)
		}
	}
}

// ClearSelection 清空選擇
func (s *InstallState) ClearSelection() {
	s.InstallProtocols = []int{}
}

// SelectAll 選擇所有協議
func (s *InstallState) SelectAll() {
	s.InstallProtocols = []int{1, 3, 4}
}

// IsEmpty 是否沒有選中任何協議
func (s *InstallState) IsEmpty() bool {
	return len(s.InstallProtocols) == 0
}
