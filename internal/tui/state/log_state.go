package state

import (
	"github.com/charmbracelet/bubbles/viewport"
)

// LogState 日誌管理狀態
type LogState struct {
	// Viewport 用於處理長文本的滾動顯示
	Viewport viewport.Model

	// 用於判斷是否是第一次加載，以便初始化 Viewport 高度
	ViewportReady bool

	// 是否正在跟蹤日誌 (tail -f 模式)
	IsFollowing bool

	// 當前顯示的日誌內容 (緩存)
	Content string
}

func NewLogState() *LogState {
	return &LogState{
		ViewportReady: false,
		IsFollowing:   false,
	}
}

// UpdateContent 更新日誌內容並移動到底部
func (s *LogState) UpdateContent(content string) {
	s.Content = content
	s.Viewport.SetContent(content)
	// 加載新日誌時自動滾動到底部
	s.Viewport.GotoBottom()
}
