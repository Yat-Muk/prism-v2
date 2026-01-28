package view

import (
	"fmt"
	"strings"

	"github.com/Yat-Muk/prism-v2/internal/tui/style"
	"github.com/Yat-Muk/prism-v2/internal/tui/types"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/lipgloss"
)

// RenderServiceHealth 渲染服務健康檢查結果
func RenderServiceHealth(result *types.HealthCheckResult, ti textinput.Model, statusMsg string) string {
	header := renderSubpageHeader("服務健康檢查")

	desc := lipgloss.NewStyle().
		Foreground(style.Snow2).
		Render(" 檢測 sing-box 服務運行狀況")

	divider := lipgloss.NewStyle().
		Foreground(style.Polar4).
		Render(strings.Repeat("─", 50))

	if result == nil {
		content := lipgloss.NewStyle().
			Foreground(style.Muted).
			Render("正在進行健康檢查...")

		footer := RenderInputFooter(ti)

		return lipgloss.JoinVertical(
			lipgloss.Left,
			header,
			desc,
			divider,
			"",
			content,
			footer,
		)
	}

	// 檢查項狀態
	labelStyle := lipgloss.NewStyle().Foreground(style.Snow3)

	var overallStyle lipgloss.Style
	var overallText string

	switch result.OverallStatus {
	case "healthy":
		overallStyle = lipgloss.NewStyle().Foreground(style.StatusGreen).Bold(true)
		overallText = "✓ 健康"
	case "warning":
		overallStyle = lipgloss.NewStyle().Foreground(style.StatusYellow).Bold(true)
		overallText = "⚠ 警告"
	case "error":
		overallStyle = lipgloss.NewStyle().Foreground(style.StatusRed).Bold(true)
		overallText = "✗ 錯誤"
	default:
		overallStyle = lipgloss.NewStyle().Foreground(style.Muted)
		overallText = "未知"
	}

	overallLine := fmt.Sprintf("%s %s",
		labelStyle.Render(" 總體狀態："),
		overallStyle.Render(overallText))

	// 問題列表
	var issuesText string
	if len(result.Issues) > 0 {
		issueStyle := lipgloss.NewStyle().Foreground(style.StatusRed)
		issuesText = "\n" + lipgloss.NewStyle().Foreground(style.Snow3).Render(" 發現問題：") + "\n"
		for i, issue := range result.Issues {
			issuesText += issueStyle.Render(fmt.Sprintf("  %d. %s\n", i+1, issue))
		}
	}

	// 建議列表
	var recommendText string
	if len(result.Recommendations) > 0 {
		recommendStyle := lipgloss.NewStyle().Foreground(style.Aurora2)
		recommendText = "\n" + lipgloss.NewStyle().Foreground(style.Snow3).Render(" 建議：") + "\n"
		for i, rec := range result.Recommendations {
			recommendText += recommendStyle.Render(fmt.Sprintf("  %d. %s\n", i+1, rec))
		}
	}

	statusBlock := RenderStatusMessage(statusMsg)

	footer := RenderInputFooter(ti)

	return lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		desc,
		divider,
		"",
		overallLine,
		issuesText,
		recommendText,
		statusBlock,
		footer,
	)
}
