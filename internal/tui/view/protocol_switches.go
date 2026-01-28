package view

import (
	"fmt"
	"strings"

	"github.com/mattn/go-runewidth"

	"github.com/Yat-Muk/prism-v2/internal/domain/protocol"
	"github.com/Yat-Muk/prism-v2/internal/tui/style"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/lipgloss"
)

// RenderProtocolSwitches 配置與協議 > 協議管理
func RenderProtocolSwitches(selected []int, ti textinput.Model, statusMsg string) string {
	header := renderSubpageHeader("協議開關管理")

	enabled := make(map[int]bool)
	for _, n := range selected {
		enabled[n] = true
	}

	desc1 := lipgloss.NewStyle().
		Foreground(style.Snow2).
		Render(" 批量啟用或禁用協議，無需進入各協議詳細配置")

	infoSep := lipgloss.NewStyle().
		Foreground(style.Polar4).
		Render(strings.Repeat("─", 50))

	infoBlock := lipgloss.JoinVertical(
		lipgloss.Left,
		desc1,
		infoSep,
	)

	onText := lipgloss.NewStyle().Foreground(style.StatusGreen).Render("◉ 開啟")
	offText := lipgloss.NewStyle().Foreground(style.Snow3).Render("○ 關閉")

	// 使用 AllIDs
	allIDs := protocol.AllIDs()

	// 計算最大寬度
	maxNameWidth := 0
	for _, id := range allIDs {
		w := runewidth.StringWidth(id.String())
		if w > maxNameWidth {
			maxNameWidth = w
		}
	}

	padName := func(name string) string {
		w := runewidth.StringWidth(name)
		if w < maxNameWidth {
			name = name + strings.Repeat(" ", maxNameWidth-w)
		}
		return name
	}

	state := func(id int) string {
		if enabled[id] {
			return onText
		}
		return offText
	}

	var items []MenuItem
	for _, id := range allIDs {

		nameDisplay := padName(id.String()) + "  " + state(int(id))

		items = append(items, MenuItem{
			Num:       fmt.Sprintf("%d", id),
			Text:      nameDisplay,
			Desc:      id.Tag(),
			TextColor: style.Snow1,
		})
	}

	menu := renderMenuWithAlignment(items, 0, "", false)

	instruction := lipgloss.NewStyle().
		Foreground(style.Snow3).
		Render(" 💡 輸入編號切換狀態，修改後請記得「應用配置」")

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
