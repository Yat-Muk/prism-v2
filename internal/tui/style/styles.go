package style

import (
	"github.com/charmbracelet/lipgloss"
)

var (
	// 標題樣式
	TitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(Primary).
			Padding(1, 2).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(Primary)

	// 菜單項樣式
	MenuItemStyle = lipgloss.NewStyle().
			Padding(0, 2)

	// 選中的菜單項
	SelectedMenuItemStyle = lipgloss.NewStyle().
				Padding(0, 2).
				Foreground(Text).
				Background(Primary).
				Bold(true)

	// 狀態樣式
	StatusBarStyle = lipgloss.NewStyle().
			Foreground(Muted).
			Padding(1, 2)

	// 幫助樣式
	HelpStyle = lipgloss.NewStyle().
			Foreground(Muted).
			Padding(1, 2)

	// 錯誤樣式
	ErrorStyle = lipgloss.NewStyle().
			Foreground(Error).
			Bold(true)

	// 成功樣式
	SuccessStyle = lipgloss.NewStyle().
			Foreground(Success).
			Bold(true)

	// 警告樣式
	WarningStyle = lipgloss.NewStyle().
			Foreground(Warning).
			Bold(true)

	// 信息樣式
	InfoStyle = lipgloss.NewStyle().
			Foreground(Info)

	// 面板樣式
	PanelStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(Secondary).
			Padding(1, 2).
			Margin(1, 2)

	// 列表樣式
	ListStyle = lipgloss.NewStyle().
			Padding(1, 2)
)

// GetStatusColor 根據狀態返回顏色
func GetStatusColor(status string) lipgloss.Color {
	switch status {
	case "running", "active", "enabled":
		return Success
	case "stopped", "inactive", "disabled":
		return Error
	case "starting", "stopping":
		return Warning
	default:
		return Muted
	}
}
