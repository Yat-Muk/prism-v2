package view

import (
	"fmt"
	"strings"

	"github.com/Yat-Muk/prism-v2/internal/tui/constants"
	"github.com/Yat-Muk/prism-v2/internal/tui/style"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/lipgloss"
)

// RenderLogLevelEdit 渲染日誌級別編輯
func RenderLogLevelEdit(currentLevel string, ti textinput.Model, statusMsg string) string {
	header := renderSubpageHeader("修改日誌級別")

	desc := lipgloss.NewStyle().
		Foreground(style.Snow2).
		Render(" 選擇日誌輸出級別")

	divider := lipgloss.NewStyle().
		Foreground(style.Polar4).
		Render(strings.Repeat("─", 50))

	currentDisplay := fmt.Sprintf(" 當前輸出級別: %s",
		lipgloss.NewStyle().Foreground(style.StatusGreen).Render(currentLevel),
	)

	// 日誌級別選項
	type levelItem struct {
		key  string
		text string
	}
	levels := []levelItem{
		{constants.KeyLevel_Debug, "debug"},
		{constants.KeyLevel_Info, "info"},
		{constants.KeyLevel_Warn, "warn"},
		{constants.KeyLevel_Error, "error"},
	}

	items := []MenuItem{
		{
			Num:       "",
			Text:      "",
			Desc:      "",
			TextColor: lipgloss.Color(""),
		},
	}

	for _, item := range levels {

		items = append(items, MenuItem{
			Num:       item.key,
			Text:      item.text,
			Desc:      "",
			TextColor: style.Snow1,
		})
	}

	menu := renderMenuWithAlignment(items, 0, "", false)

	instruction := lipgloss.NewStyle().
		Foreground(style.Snow3).
		Render(" 💡 生產環境建議使用 info 或 warn 級別")

	statusBlock := RenderStatusMessage(statusMsg)

	footer := RenderInputFooter(ti)

	return lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		desc,
		divider,
		currentDisplay,
		menu,
		"",
		instruction,
		statusBlock,
		footer,
	)
}
