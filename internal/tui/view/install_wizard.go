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

// RenderInstallWizard 渲染安裝向導：協議選擇
func RenderInstallWizard(selected []int, ti textinput.Model, statusMsg string) string {
	// 頭部 + 提示
	header := renderSubpageHeader("安裝向導 · 選擇協議")

	hint := lipgloss.NewStyle().
		Foreground(style.Snow3).
		Render(" 選擇要安裝的協議")

	infoSep := lipgloss.NewStyle().
		Foreground(style.Polar4).
		Render(strings.Repeat("─", 50))

	infoBlock := lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		hint,
		infoSep,
	)

	// 已選中的協議 map
	enabled := make(map[int]bool)
	for _, n := range selected {
		enabled[n] = true
	}

	onText := lipgloss.NewStyle().Foreground(style.StatusGreen).Render(" ◉ 開啟")
	offText := lipgloss.NewStyle().Foreground(style.Snow3).Render(" ○ 關閉")

	// 從領域層獲取所有 ID
	allIDs := protocol.AllIDs()

	// 計算名稱欄最大寬度
	maxNameWidth := 0
	for _, id := range allIDs {
		w := runewidth.StringWidth(id.String())
		if w > maxNameWidth {
			maxNameWidth = w
		}
	}

	// 補齊空格 + 狀態
	pad := func(name string, isActive bool) string {
		w := runewidth.StringWidth(name)
		if w < maxNameWidth {
			name += strings.Repeat(" ", maxNameWidth-w)
		}
		stateText := offText
		if isActive {
			stateText = onText
		}
		return name + "  " + stateText
	}

	var items []MenuItem
	for _, id := range allIDs {

		items = append(items, MenuItem{
			Num:       fmt.Sprintf("%d", id), // 動態使用 ID
			Text:      pad(id.String(), enabled[int(id)]),
			Desc:      id.Tag(),
			TextColor: style.Snow1,
		})
	}

	menu := renderMenuWithAlignment(items, 0, "", false)

	instruction := lipgloss.NewStyle().
		Foreground(style.Snow3).
		Render(" 💡 多選用逗號分隔 (如: 2,5,7)，按回車開始安裝")

	statusBlock := RenderStatusMessage(statusMsg)

	// 使用新的渲染函數，正確顯示光標
	footer := RenderInputFooter(ti)

	return lipgloss.JoinVertical(
		lipgloss.Left,
		infoBlock,
		menu,
		"",
		instruction,
		statusBlock,
		footer,
	)
}
