package model

import (
	tea "github.com/charmbracelet/bubbletea"
)

// Model TUI 核心模型
type Model struct {
	router *Router
}

// NewModel 創建新的 TUI Model
func NewModel(router *Router) *Model {
	return &Model{
		router: router,
	}
}

// Init 初始化
func (m *Model) Init() tea.Cmd {
	return m.router.InitModel()
}

// Update 更新循環
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	_, cmd := m.router.Update(msg)
	return m, cmd
}

// View 渲染視圖
func (m *Model) View() string {
	return m.router.View()
}
