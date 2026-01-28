package view

import (
	"fmt"
	"strings"

	"github.com/Yat-Muk/prism-v2/internal/tui/style"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/lipgloss"
)

// RenderCoreVersionSelect 渲染核心版本選擇
func RenderCoreVersionSelect(
	versions []string,
	currentVersion string,
	latestVersion string,
	ti textinput.Model,
	statusMsg string,
) string {
	header := renderSubpageHeader("選擇版本")

	desc := lipgloss.NewStyle().
		Foreground(style.Snow2).
		Render(" 選擇要安裝的 sing-box 版本")

	divider := lipgloss.NewStyle().
		Foreground(style.Polar4).
		Render(strings.Repeat("─", 50))

	labelStyle := lipgloss.NewStyle().Foreground(style.Snow3)
	valueStyle := lipgloss.NewStyle().Foreground(style.Aurora2)

	infoBlock := fmt.Sprintf("%s %s",
		labelStyle.Render(" 當前安裝版本:"),
		valueStyle.Render(currentVersion),
	)

	// 顯示可用版本
	items := []MenuItem{
		{
			Num:       "",
			Text:      "",
			Desc:      "",
			TextColor: lipgloss.Color(""),
		},
	}

	// 添加版本列表（最多顯示 9 個）
	maxVersions := 9
	if len(versions) > maxVersions {
		versions = versions[:maxVersions]
	}

	for i, ver := range versions {
		text := ver
		var color lipgloss.Color

		if ver == latestVersion {
			text = fmt.Sprintf("%s (最新)", ver)
			color = style.StatusGreen
		} else if ver == currentVersion {
			text = fmt.Sprintf("%s (當前)", ver)
			color = style.Aurora2
		} else {
			color = style.Snow1
		}

		items = append(items, MenuItem{
			Num:       fmt.Sprintf("%d", i+1),
			Text:      text,
			Desc:      "",
			TextColor: color,
		})
	}

	menu := renderMenuWithAlignment(items, 0, "", false)

	instruction := lipgloss.NewStyle().
		Foreground(style.Snow3).
		Render(" 💡 輸入版本號 (如: 1.12.0) 或選擇列表中的版本")

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
