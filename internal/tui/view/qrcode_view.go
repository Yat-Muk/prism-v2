package view

import (
	"github.com/Yat-Muk/prism-v2/internal/tui/style"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/lipgloss"
)

// RenderQRCode 渲染二維碼視圖
func RenderQRCode(qrAscii string, content string, qrType string, ti textinput.Model, statusMsg string) string {
	var title string
	switch qrType {
	case "protocol":
		title = "協議掃碼"
	case "subscription":
		title = "訂閱掃碼"
	default:
		title = "二維碼"
	}

	header := renderSubpageHeader(title)

	var qrBlock string
	if qrAscii != "" {
		qrStyle := lipgloss.NewStyle().
			Align(lipgloss.Center).
			Foreground(style.White).
			Padding(0, 0)

		qrBlock = qrStyle.Render(qrAscii)
	} else {
		qrBlock = lipgloss.NewStyle().
			Foreground(style.Muted).
			Padding(2).
			Render("正在生成二維碼...")
	}

	displayContent := content
	if len(displayContent) > 60 {
		displayContent = displayContent[:28] + "..." + displayContent[len(displayContent)-28:]
	}

	contentBlock := lipgloss.NewStyle().
		Foreground(style.Aurora2).
		Align(lipgloss.Center).
		Render(displayContent)

	keyStyle := lipgloss.NewStyle().Foreground(style.Aurora4)
	descStyle := lipgloss.NewStyle().Foreground(style.Muted)

	helpBar := keyStyle.Render("Esc") + descStyle.Render(" 返回")

	mainContent := lipgloss.JoinVertical(
		lipgloss.Center,
		qrBlock,
		contentBlock,
		"",
		helpBar,
	)

	statusBlock := RenderStatusMessage(statusMsg)

	return lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		lipgloss.NewStyle().Padding(1, 0).Render(mainContent),
		statusBlock,
	)
}
