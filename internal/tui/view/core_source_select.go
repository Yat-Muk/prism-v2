package view

import (
	"fmt"
	"strings"

	"github.com/Yat-Muk/prism-v2/internal/tui/style"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/lipgloss"
)

// CoreSource 核心源
type CoreSource struct {
	Name string
	URL  string
}

// RenderCoreSourceSelect 渲染核心源選擇
func RenderCoreSourceSelect(sources []CoreSource, selectedIndex int, ti textinput.Model, statusMsg string) string {
	header := renderSubpageHeader("選擇下載源")

	desc := lipgloss.NewStyle().
		Foreground(style.Snow2).
		Render(" 選擇 sing-box 核心下載源")

	divider := lipgloss.NewStyle().
		Foreground(style.Polar4).
		Render(strings.Repeat("─", 50))

	// 顯示當前源信息
	currentName := "未知"
	if selectedIndex >= 0 && selectedIndex < len(sources) {
		currentName = sources[selectedIndex].Name
	}

	labelStyle := lipgloss.NewStyle().Foreground(style.Snow3)
	valueStyle := lipgloss.NewStyle().Foreground(style.Aurora4)

	infoBlock := fmt.Sprintf("%s %s",
		labelStyle.Render(" 當前下載源:"),
		valueStyle.Render(currentName),
	)

	// 構建源列表
	items := []MenuItem{
		{
			Num:       "",
			Text:      "",
			Desc:      "",
			TextColor: lipgloss.Color(""),
		},
	}
	for i, source := range sources {
		items = append(items, MenuItem{
			Num:       fmt.Sprintf("%d", i+1),
			Text:      source.Name,
			Desc:      source.URL,
			TextColor: style.Snow1,
		})
	}

	menu := renderMenuWithAlignment(items, 0, "", false)

	instruction := lipgloss.NewStyle().
		Foreground(style.Snow3).
		Render(" 💡 選擇速度最快的下載源")

	statusBlock := RenderStatusMessage(statusMsg)

	footer := RenderInputFooter(ti)

	return lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		desc,
		divider,
		infoBlock,
		menu,
		"",
		instruction,
		statusBlock,
		footer,
	)
}
