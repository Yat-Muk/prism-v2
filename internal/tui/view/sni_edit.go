package view

import (
	"fmt"
	"strings"

	"github.com/Yat-Muk/prism-v2/internal/tui/style"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/lipgloss"
)

func RenderSNIEditView(currentSNI string, ti textinput.Model, statusMsg string) string {
	header := renderSubpageHeader("修改 SNI 域名")

	if currentSNI == "" {
		currentSNI = "(尚未設置)"
	}

	desc1 := lipgloss.NewStyle().
		Foreground(style.Snow2).
		Render(" 更換 Reality 偽裝目標域名")

	currentLine := lipgloss.NewStyle().
		Foreground(style.Snow2).
		Render(fmt.Sprintf(
			" 當前 SNI: %s",
			lipgloss.NewStyle().Foreground(style.Aurora4).Render(currentSNI),
		))

	// 信息區：說明 + 灰色分隔線 + 當前值
	infoSep := lipgloss.NewStyle().
		Foreground(style.Polar4).
		Render(strings.Repeat("─", 50))

	infoBlock := lipgloss.JoinVertical(
		lipgloss.Left,
		desc1,
		infoSep,
		currentLine,
	)

	items := []MenuItem{}

	menu := renderMenuWithAlignment(items, 0, "", false)

	instruction := lipgloss.NewStyle().
		Foreground(style.Snow3).
		Render(" 💡 輸入的新 SNI 偽裝域名，需支持 TLSv1.3")

	statusBlock := RenderStatusMessage(statusMsg)

	footer := RenderInputFooter(ti)

	return lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		infoBlock,
		menu,
		"",
		instruction,
		statusBlock,
		footer,
	)
}
