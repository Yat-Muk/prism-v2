package view

import (
	"fmt"
	"strings"

	"github.com/Yat-Muk/prism-v2/internal/tui/constants"
	"github.com/Yat-Muk/prism-v2/internal/tui/style"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/lipgloss"
)

func RenderUUIDEditView(currentUUID string, ti textinput.Model, statusMsg string) string {
	header := renderSubpageHeader("修改 UUID")

	if currentUUID == "" {
		currentUUID = "(尚未設置)"
	}

	desc1 := lipgloss.NewStyle().
		Foreground(style.Snow2).
		Render(" 修改所有協議共用的用戶標識符 (UUID)")

	currentLine := lipgloss.NewStyle().
		Foreground(style.Snow2).
		Render(fmt.Sprintf(
			" 當前 UUID: %s",
			lipgloss.NewStyle().Foreground(style.Aurora4).Render(currentUUID),
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

	// 用標準 MenuItem 渲染選項，[推薦] 自動黃色
	items := []MenuItem{
		{"", "", "", lipgloss.Color("")},
		{constants.KeyUUID_Generate, "自動生成新的 UUID ", "[推薦]", style.Snow1},
		{constants.KeyUUID_Manual, "手動輸入 UUID", "", style.Snow1},
	}

	menu := renderMenuWithAlignment(items, 0, "", false)

	statusBlock := RenderStatusMessage(statusMsg)

	footer := RenderInputFooter(ti)

	return lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		infoBlock,
		menu,
		statusBlock,
		footer,
	)
}
