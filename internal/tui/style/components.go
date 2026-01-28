package style

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var (
	// Logo 樣式 - 使用漸變
	LogoStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(Aurora2).
			Padding(0, 2)

	// Dashboard 標題
	DashboardTitleStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(Snow1).
				Background(Polar2).
				Padding(0, 2).
				Border(lipgloss.RoundedBorder()).
				BorderForeground(Aurora2)

	// 信息卡片
	CardStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(Aurora3).
			Padding(1, 2).
			Margin(0, 1)

	// 高亮卡片
	CardHighlightStyle = lipgloss.NewStyle().
				Border(lipgloss.ThickBorder()).
				BorderForeground(Aurora1).
				Padding(1, 2).
				Margin(0, 1).
				Background(Polar2)

	// 狀態指示器
	StatusIndicatorActive = lipgloss.NewStyle().
				Foreground(StatusGreen).
				Bold(true)

	StatusIndicatorInactive = lipgloss.NewStyle().
				Foreground(StatusRed).
				Bold(true)

	StatusIndicatorWarning = lipgloss.NewStyle().
				Foreground(StatusYellow).
				Bold(true)

	// 進度條樣式
	ProgressBarStyle = lipgloss.NewStyle().
				Foreground(Aurora2)

	ProgressBarEmptyStyle = lipgloss.NewStyle().
				Foreground(Polar4)

	// 表格樣式
	TableHeaderStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(Snow1).
				Background(Polar3).
				Padding(0, 1).
				Border(lipgloss.NormalBorder()).
				BorderForeground(Aurora3)

	TableCellStyle = lipgloss.NewStyle().
			Foreground(Snow2).
			Padding(0, 1)

	// 按鈕樣式
	ButtonStyle = lipgloss.NewStyle().
			Foreground(Snow1).
			Background(Aurora3).
			Padding(0, 2).
			Margin(0, 1).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(Aurora3)

	ButtonActiveStyle = lipgloss.NewStyle().
				Foreground(Polar1).
				Background(Aurora1).
				Padding(0, 2).
				Margin(0, 1).
				Border(lipgloss.ThickBorder()).
				BorderForeground(Aurora1).
				Bold(true)

	// 徽章樣式
	BadgeSuccessStyle = lipgloss.NewStyle().
				Foreground(Polar1).
				Background(StatusGreen).
				Padding(0, 1).
				Bold(true)

	BadgeErrorStyle = lipgloss.NewStyle().
			Foreground(Snow1).
			Background(StatusRed).
			Padding(0, 1).
			Bold(true)

	BadgeWarningStyle = lipgloss.NewStyle().
				Foreground(Polar1).
				Background(StatusYellow).
				Padding(0, 1).
				Bold(true)

	BadgeInfoStyle = lipgloss.NewStyle().
			Foreground(Snow1).
			Background(Aurora3).
			Padding(0, 1).
			Bold(true)
)

// RenderProgressBar 渲染進度條
func RenderProgressBar(percent float64, width int) string {
	if width < 2 {
		width = 20
	}

	filled := int(float64(width) * percent / 100.0)
	empty := width - filled

	bar := strings.Repeat("█", filled) + strings.Repeat("░", empty)
	percentage := fmt.Sprintf(" %5.1f%%", percent)

	return ProgressBarStyle.Render(bar) + percentage
}

// RenderSparkline 渲染迷你圖表
func RenderSparkline(data []float64) string {
	if len(data) == 0 {
		return ""
	}

	// 使用 Unicode 字符創建迷你圖表
	chars := []rune{'▁', '▂', '▃', '▄', '▅', '▆', '▇', '█'}

	// 找到最大值
	max := 0.0
	for _, v := range data {
		if v > max {
			max = v
		}
	}

	if max == 0 {
		max = 1
	}

	// 生成圖表
	var result strings.Builder
	for _, v := range data {
		index := int((v / max) * float64(len(chars)-1))
		if index >= len(chars) {
			index = len(chars) - 1
		}
		result.WriteRune(chars[index])
	}

	return ProgressBarStyle.Render(result.String())
}

// RenderBadge 渲染徽章
func RenderBadge(text string, badgeType string) string {
	switch badgeType {
	case "success":
		return BadgeSuccessStyle.Render(text)
	case "error":
		return BadgeErrorStyle.Render(text)
	case "warning":
		return BadgeWarningStyle.Render(text)
	case "info":
		return BadgeInfoStyle.Render(text)
	default:
		return text
	}
}

// RenderBox 渲染帶標題的盒子
func RenderBox(title, content string, width int) string {
	if width < 10 {
		width = 40
	}

	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(Aurora3).
		Width(width).
		Padding(1, 2)

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(Aurora1)

	return boxStyle.Render(
		titleStyle.Render(title) + "\n\n" + content,
	)
}
