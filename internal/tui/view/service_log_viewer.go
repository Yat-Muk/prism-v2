package view

import (
	"strings"

	"github.com/Yat-Muk/prism-v2/internal/tui/style"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/lipgloss"
)

// RenderServiceLogViewer 渲染服務日誌實時查看器
func RenderServiceLogViewer(logs []string, following bool, ti textinput.Model) string {
	header := renderSubpageHeader("服務實時日誌")

	divider := lipgloss.NewStyle().
		Foreground(style.Polar4).
		Render(strings.Repeat("─", 50))

	// 日誌內容
	logStyle := lipgloss.NewStyle().
		Foreground(style.Snow1).
		Width(50).
		MaxWidth(78)

	var logContent string
	if len(logs) == 0 {
		logContent = lipgloss.NewStyle().
			Foreground(style.Muted).
			Render("等待日誌輸出...")
	} else {
		var lines []string
		for _, log := range logs {
			// 根據日誌級別著色
			if strings.Contains(log, "ERROR") || strings.Contains(log, "FATAL") {
				lines = append(lines, lipgloss.NewStyle().Foreground(style.StatusRed).Render(log))
			} else if strings.Contains(log, "WARN") {
				lines = append(lines, lipgloss.NewStyle().Foreground(style.StatusYellow).Render(log))
			} else if strings.Contains(log, "DEBUG") {
				lines = append(lines, lipgloss.NewStyle().Foreground(style.Muted).Render(log))
			} else if strings.Contains(log, "INFO") {
				lines = append(lines, lipgloss.NewStyle().Foreground(style.StatusGreen).Render(log))
			} else {
				lines = append(lines, logStyle.Render(log))
			}
		}
		logContent = strings.Join(lines, "\n")
	}

	// 狀態欄
	statusStyle := lipgloss.NewStyle().Foreground(style.Aurora2).Bold(true)
	var statusText string
	if following {
		statusText = statusStyle.Render("● 實時滾動中...")
	} else {
		statusText = lipgloss.NewStyle().Foreground(style.StatusYellow).Render("⏸ 已暫停")
	}

	// 這裡使用通用的輸入 Footer，確保光標和 Esc 提示一致
	footer := RenderInputFooter(ti)

	return lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		divider,
		logContent,
		"",
		statusText,
		footer,
	)
}
